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
func (r *IpAddressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpAddress{}
	conditionMessage := ""
	objectDeleted := false

	err := r.Client.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		if objectDeleted {
			logger.Info("reconcile loop ended, object was deleted")
			return
		}
		if err := r.UpdateConditions(ctx, o, conditionMessage); err != nil {
			reconcileErr = errors.Join(reconcileErr, err)
		}
		logger.Info("reconcile loop ended")
	}()

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(o, IpAddressFinalizerName) {
			if !o.Spec.PreserveInNetbox {
				if o.Status.IpAddressId != 0 {
					if err := r.NetboxClient.DeleteIpAddress(o.Status.IpAddressId); err != nil {
						conditionMessage = err.Error()
						return ctrl.Result{Requeue: true}, nil
					}
				}
			}

			logger.V(4).Info("removing the finalizer")
			removed := controllerutil.RemoveFinalizer(o, IpAddressFinalizerName)
			if !removed {
				return ctrl.Result{}, errors.New("failed to remove the finalizer")
			}

			if err = r.Update(ctx, o); err != nil {
				return ctrl.Result{}, err
			}
		}

		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// if PreserveIpInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(o, IpAddressFinalizerName) {
		logger.V(4).Info("adding the finalizer")
		controllerutil.AddFinalizer(o, IpAddressFinalizerName)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. try to lock lease of parent prefix if IpAddressUrl is not set in status
	// and IpAddress is owned by an IpAddressClaim
	or := o.ObjectMeta.OwnerReferences
	if len(or) > 0 && o.Status.IpAddressUrl == "" {
		// get ip address claim owner
		orLookupKey := types.NamespacedName{
			Name:      or[0].Name,
			Namespace: req.Namespace,
		}
		ipAddressClaim := &netboxv1.IpAddressClaim{}
		if err = r.Client.Get(ctx, orLookupKey, ipAddressClaim); err != nil {
			return ctrl.Result{}, err
		}

		// get name of parent prefix
		leaseLockerNSN := types.NamespacedName{
			Name:      convertCIDRToLeaseLockName(ipAddressClaim.Spec.ParentPrefix),
			Namespace: r.OperatorNamespace,
		}
		ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.NamespacedName.String())
		if err != nil {
			return ctrl.Result{}, err
		}

		locked := ll.TryLock(ctx)
		if !locked {
			conditionMessage = fmt.Sprintf("failed to lock parent prefix %s", ipAddressClaim.Spec.ParentPrefix)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		defer ll.UnlockWithRetry(ctx)
		logger.V(4).Info("successfully locked parent prefix", "prefix", ipAddressClaim.Spec.ParentPrefix)
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
		o.Status.SyncState = netboxv1.SyncStateFailed
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.IpAddressId == 0 {
			// if there is a restoration hash mismatch and the IpAddressId status field is not set,
			// delete the ip address so it can be recreated by the ip address claim controller
			logger.Info("restoration hash mismatch, deleting ip address custom resource", "ipaddress", o.Spec.IpAddress)
			if err = r.Client.Delete(ctx, o); err != nil {
				conditionMessage = "failed to delete IpAddress CR with restoration hash mismatch"
				return ctrl.Result{Requeue: true}, nil
			}
			objectDeleted = true
			return ctrl.Result{}, nil
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// 3. update annotations
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[IPManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		logger.Error(err, "failed to generate managed custom fields annotation")
		return ctrl.Result{Requeue: true}, nil
	}

	if err = accessor.SetAnnotations(o, annotations); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	// 4. update status fields (set after r.Update to avoid being overwritten by API response)
	o.Status.SyncState = netboxv1.SyncStateSucceeded
	o.Status.IpAddressId = netboxIpAddressModel.ID
	o.Status.IpAddressUrl = config.GetBaseUrl() + "/ipam/ip-addresses/" + strconv.FormatInt(netboxIpAddressModel.ID, 10)

	// check if created ip address contains entire description from spec
	_, found := strings.CutPrefix(netboxIpAddressModel.Description, req.NamespacedName.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "IpDescriptionTruncated", "ip address was created with truncated description")
	}

	logger.V(4).Info("reserved ip address in netbox", "ip", o.Spec.IpAddress)

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
		},
	}, nil
}

func (r *IpAddressReconciler) UpdateConditions(ctx context.Context, o *netboxv1.IpAddress, conditionMessage string) error {
	var additionalMsgs []string
	if conditionMessage != "" {
		additionalMsgs = []string{conditionMessage}
	}

	switch {
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyFalse, corev1.EventTypeNormal, nil, additionalMsgs...)
	case o.Status.IpAddressUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyFalse, corev1.EventTypeWarning, nil, additionalMsgs...)
	case o.Status.SyncState == netboxv1.SyncStateFailed:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyFalse, corev1.EventTypeWarning, nil, additionalMsgs...)
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyTrue, corev1.EventTypeNormal, nil, additionalMsgs...)
	}

	if err := r.Status().Update(ctx, o); err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}
