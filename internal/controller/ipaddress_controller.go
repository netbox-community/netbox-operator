/*
Copyright 2024 Swisscom (Schweiz) AG.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/swisscom/leaselocker"
	corev1 "k8s.io/api/core/v1"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const IpAddressFinalizerName = "ipaddress.netbox.dev/finalizer"
const IPManagedCustomFieldsAnnotationName = "ipaddress.netbox.dev/managed-custom-fields"

// IpAddressReconciler reconciles a IpAddress object
type IpAddressReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddresses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpAddressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpAddress{}
	err := r.Client.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(o, IpAddressFinalizerName) {
			if !o.Spec.PreserveInNetbox {
				err := r.NetboxClient.DeleteIpAddress(o.Status.IpAddressId)
				if err != nil {
					if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpaddressReadyFalseDeletionFailed, corev1.EventTypeWarning, err); errReport != nil {
						return ctrl.Result{}, errReport
					}
					return ctrl.Result{Requeue: true}, nil
				}
			}

			debugLogger.Info("removing the finalizer")
			removed := controllerutil.RemoveFinalizer(o, IpAddressFinalizerName)
			if !removed {
				return ctrl.Result{}, errors.New("failed to remove the finalizer")
			}

			err = r.Update(ctx, o)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// if PreserveIpInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(o, IpAddressFinalizerName) {
		debugLogger.Info("adding the finalizer")
		controllerutil.AddFinalizer(o, IpAddressFinalizerName)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set ready to false initially
	if apismeta.FindStatusCondition(o.Status.Conditions, netboxv1.ConditionReadyFalseNewResource.Type) == nil {
		err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionReadyFalseNewResource, corev1.EventTypeNormal, nil)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to initialise Ready condition: %w, ", err)
		}
	}

	// 1. try to lock lease of parent prefix if IpAddress status condition is not true
	// and IpAddress is owned by an IpAddressClaim
	or := o.ObjectMeta.OwnerReferences
	var ll *leaselocker.LeaseLocker
	if len(or) > 0 /* len(nil array) = 0 */ && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		// get ip address claim
		orLookupKey := types.NamespacedName{
			Name:      or[0].Name,
			Namespace: req.Namespace,
		}
		ipAddressClaim := &netboxv1.IpAddressClaim{}
		err = r.Client.Get(ctx, orLookupKey, ipAddressClaim)
		if err != nil {
			return ctrl.Result{}, err
		}

		// get name of parent prefix
		leaseLockerNSN := types.NamespacedName{
			Name:      convertCIDRToLeaseLockName(ipAddressClaim.Spec.ParentPrefix),
			Namespace: r.OperatorNamespace,
		}
		ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.NamespacedName.String())
		if err != nil {
			return ctrl.Result{}, err
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// create lock
		locked := ll.TryLock(lockCtx)
		if !locked {
			errorMsg := fmt.Sprintf("failed to lock parent prefix %s", ipAddressClaim.Spec.ParentPrefix)
			r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", ipAddressClaim.Spec.ParentPrefix))
	}

	// 2. reserve or update ip address in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	ipAddressModel, err := generateNetboxIpAddressModelFromIpAddressSpec(&o.Spec, req, annotations[IPManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxIpAddressModel, err := r.NetboxClient.ReserveOrUpdateIpAddress(ipAddressModel)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.IpAddressId == 0 {
			// if there is a restoration hash mismatch and the IpAddressId status field is not set,
			// delete the ip address so it can be recreated by the ip address claim controller
			// this will only affect resources that are created by a claim controller (and have a restoration hash custom field
			logger.Info("restoration hash mismatch, deleting ip address custom resource", "ipaddress", o.Spec.IpAddress)
			err = r.Client.Delete(ctx, o)
			if err != nil {
				if updateStatusErr := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpaddressReadyFalse,
					corev1.EventTypeWarning, err); updateStatusErr != nil {
					return ctrl.Result{}, fmt.Errorf("failed to update ip address status: %w, "+
						"after deletion of ip address cr failed: %w", updateStatusErr, err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, nil

		}

		if updateStatusErr := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpaddressReadyFalse,
			corev1.EventTypeWarning, err, o.Spec.IpAddress); updateStatusErr != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update ip address status: %w, "+
				"after reservation of ip in netbox failed: %w", updateStatusErr, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 3. unlock lease of parent prefix
	if ll != nil {
		ll.UnlockWithRetry(ctx)
	}

	// 4. update status fields
	o.Status.IpAddressId = netboxIpAddressModel.ID
	o.Status.IpAddressUrl = config.GetBaseUrl() + "/ipam/ip-addresses/" + strconv.FormatInt(netboxIpAddressModel.ID, 10)
	err = r.Client.Status().Update(ctx, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[IPManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		logger.Error(err, "failed to update last metadata annotation")
		return ctrl.Result{Requeue: true}, nil
	}

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update object to store lastIpAddressMetadata annotation
	if err := r.Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	// check if created ip address contains entire description from spec
	_, found := strings.CutPrefix(netboxIpAddressModel.Description, req.NamespacedName.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "IpDescriptionTruncated", "ip address was created with truncated description")
	}

	debugLogger.Info(fmt.Sprintf("reserved ip address in netbox, ip: %s", o.Spec.IpAddress))

	err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpaddressReadyTrue, corev1.EventTypeNormal, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpAddressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpAddress{}).
		Complete(r)
}

func generateNetboxIpAddressModelFromIpAddressSpec(spec *netboxv1.IpAddressSpec, req ctrl.Request, lastIpAddressMetadata string) (*models.IPAddress, error) {
	// unmarshal lastIpAddressMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastIpAddressMetadata != "" {
		if err := json.Unmarshal([]byte(lastIpAddressMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastIpAddressMetadata annotation: %w", err)
		}
	}

	netboxCustomFields := make(map[string]string)
	if len(spec.CustomFields) > 0 {
		netboxCustomFields = maps.Clone(spec.CustomFields)
	}

	// if a custom field was removed from the spec, add it with an empty value
	for key := range lastAppliedCustomFields {
		_, ok := netboxCustomFields[key]
		if !ok {
			netboxCustomFields[key] = ""
		}
	}

	return &models.IPAddress{
		IpAddress: spec.IpAddress,
		Metadata: &models.NetboxMetadata{
			Comments:    spec.Comments,
			Custom:      netboxCustomFields,
			Description: req.NamespacedName.String() + " // " + spec.Description,
			Tenant:      spec.Tenant,
			Tags:        convertAPITagsToModelTags(spec.Tags),
		},
	}, nil
}
