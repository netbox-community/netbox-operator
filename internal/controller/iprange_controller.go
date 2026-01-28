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

	nclient "github.com/netbox-community/go-netbox/v4"
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

const IpRangeFinalizerName = "iprange.netbox.dev/finalizer"
const IPRManagedCustomFieldsAnnotationName = "iprange.netbox.dev/managed-custom-fields"

// IpRangeReconciler reconciles a IpRange object
type IpRangeReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	NetboxClientV4      *nclient.APIClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=ipranges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=ipranges/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=ipranges/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpRangeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpRange{}
	err := r.Client.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(o, IpRangeFinalizerName) {
			if !o.Spec.PreserveInNetbox {
				err := api.DeleteIpRange(ctx, r.NetboxClientV4, o.Status.IpRangeId)
				if err != nil {
					err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeReadyFalseDeletionFailed,
						corev1.EventTypeWarning, err)
					if err != nil {
						return ctrl.Result{}, err
					}
					return ctrl.Result{Requeue: true}, nil
				}
			}
		}

		return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, IpRangeFinalizerName)
	}

	// Set ready to false initially
	if apismeta.FindStatusCondition(o.Status.Conditions, netboxv1.ConditionReadyFalseNewResource.Type) == nil {
		err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionReadyFalseNewResource, corev1.EventTypeNormal, nil)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to initialise Ready condition: %w, ", err)
		}
	}

	// if PreserveIpInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox {
		err = addFinalizer(ctx, r.Client, o, IpRangeFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. try to lock lease of parent prefix if IpRange status condition is not true
	// and IpRange is owned by an IpRangeClaim
	or := o.ObjectMeta.OwnerReferences
	var ll *leaselocker.LeaseLocker
	if len(or) > 0 && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {

		leaseLockerNSN, owner, parentPrefix, err := r.getLeaseLockerNSNandOwner(ctx, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, owner)
		if err != nil {
			return ctrl.Result{}, err
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// create lock
		locked := ll.TryLock(lockCtx)
		if !locked {
			errorMsg := fmt.Sprintf("failed to lock parent prefix %s", parentPrefix)
			r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		logger.V(4).Info(fmt.Sprintf("successfully locked parent prefix %s", parentPrefix))
	}

	// 2. reserve or update ip range in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	ipRangeModel, err := r.generateNetboxIpRangeModelFromIpRangeSpec(o, req, annotations[IPRManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxIpRangeModel, err := api.ReserveOrUpdateIpRange(ctx, r.NetboxClient, r.NetboxClientV4, ipRangeModel)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.IpRangeId == 0 {
			// if there is a restoration hash mismatch and the IpRangeId status field is not set,
			// delete the ip range so it can be recreated by the ip range claim controller
			// this will only affect resources that are created by a claim controller (and have a restoration hash custom field
			logger.Info("restoration hash mismatch, deleting ip range custom resource", "ip-range-start", o.Spec.StartAddress, "ip-range-end", o.Spec.EndAddress)
			err = r.Client.Delete(ctx, o)
			if err != nil {
				if err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeReadyFalse,
					corev1.EventTypeWarning, err); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, nil
		}

		if err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeReadyFalse,
			corev1.EventTypeWarning, err, fmt.Sprintf("range: %s-%s", o.Spec.StartAddress, o.Spec.EndAddress)); err != nil {
			return ctrl.Result{}, err
		}

		// The decision to not return the error message (just logging it) is to not trigger printing the stacktrace on api errors
		return ctrl.Result{Requeue: true}, nil
	}

	// 3. unlock lease of parent prefix
	if ll != nil {
		ll.UnlockWithRetry(ctx)
	}

	// 4. update status fields
	o.Status.IpRangeId = int64(netboxIpRangeModel.GetId())
	o.Status.IpRangeUrl = config.GetBaseUrl() + "/ipam/ip-ranges/" + strconv.FormatInt(int64(netboxIpRangeModel.GetId()), 10)
	err = r.Client.Status().Update(ctx, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[IPRManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		logger.Error(err, "failed to update last metadata annotation")
		return ctrl.Result{Requeue: true}, nil
	}

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update object to store lastIpRangeMetadata annotation
	err = r.Update(ctx, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeReadyTrue, corev1.EventTypeNormal, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpRangeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpRange{}).
		Complete(r)
}

func (r *IpRangeReconciler) generateNetboxIpRangeModelFromIpRangeSpec(o *netboxv1.IpRange, req ctrl.Request, lastIpRangeMetadata string) (*models.IpRange, error) {
	// unmarshal lastIpRangeMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastIpRangeMetadata != "" {
		if err := json.Unmarshal([]byte(lastIpRangeMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastIpRangeMetadata annotation: %w", err)
		}
	}

	netboxCustomFields := make(map[string]string)
	if len(o.Spec.CustomFields) > 0 {
		netboxCustomFields = maps.Clone(o.Spec.CustomFields)
	}

	// if a custom field was removed from the spec, add it with an empty value

	for key := range lastAppliedCustomFields {
		_, ok := netboxCustomFields[key]
		if !ok {
			netboxCustomFields[key] = ""
		}
	}

	description := api.TruncateDescription(req.NamespacedName.String() + " // " + o.Spec.Description)

	// check if created ip range contains entire description from spec
	_, found := strings.CutPrefix(description, req.NamespacedName.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "IpRangeDescriptionTruncated", "ip range was created with truncated description")
	}

	return &models.IpRange{
		StartAddress: o.Spec.StartAddress,
		EndAddress:   o.Spec.EndAddress,
		Metadata: &models.NetboxMetadata{
			Comments:    o.Spec.Comments,
			Custom:      netboxCustomFields,
			Description: description,
			Tenant:      o.Spec.Tenant,
		},
	}, nil
}

func (r *IpRangeReconciler) getLeaseLockerNSNandOwner(ctx context.Context, o *netboxv1.IpRange) (types.NamespacedName, string, string, error) {

	orLookupKey := types.NamespacedName{
		Name:      o.ObjectMeta.OwnerReferences[0].Name,
		Namespace: o.Namespace,
	}

	ipRangeClaim := &netboxv1.IpRangeClaim{}
	err := r.Client.Get(ctx, orLookupKey, ipRangeClaim)
	if err != nil {
		return types.NamespacedName{}, "", "", err
	}

	// get name of parent prefix
	leaseLockerNSN := types.NamespacedName{
		Name:      convertCIDRToLeaseLockName(ipRangeClaim.Spec.ParentPrefix),
		Namespace: r.OperatorNamespace,
	}

	return leaseLockerNSN, orLookupKey.String(), ipRangeClaim.Spec.ParentPrefix, nil
}
