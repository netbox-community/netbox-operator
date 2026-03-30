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
	"math"
	"strconv"
	"strings"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/swisscom/leaselocker"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	NetboxClient        *api.NetboxCompositeClient
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
func (r *IpRangeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpRange{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change (IpRangeId, conditions, etc.).
	statusBase := o.DeepCopy()

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, reconcileResult, reconcileErr)
	}()

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(o, IpRangeFinalizerName) {
			return ctrl.Result{}, nil
		}

		if !o.Spec.PreserveInNetbox {
			if o.Status.IpRangeId > math.MaxInt32 {
				return ctrl.Result{}, fmt.Errorf("reconciliation of ip ranges with id's larger than 2147483647 is not supported")
			}
			if err := r.NetboxClient.DeleteIpRange(ctx, int32(o.Status.IpRangeId)); err != nil {
				return ctrl.Result{Requeue: true}, NewDomainError("failed to delete ip range in netbox: %w", err)
			}
		}

		return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, IpRangeFinalizerName)
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
	or := o.OwnerReferences
	var ll *leaselocker.LeaseLocker
	var cancelLock context.CancelFunc
	if len(or) > 0 && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {

		leaseLockerNSN, owner, parentPrefix, err := r.getLeaseLockerNSNandOwner(ctx, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, owner)
		if err != nil {
			return ctrl.Result{}, err
		}

		var lockCtx context.Context
		lockCtx, cancelLock = context.WithTimeout(ctx, lockAcquireTimeout)
		defer func() {
			if cancelLock != nil {
				cancelLock()
			}
		}()

		// create lock
		locked := ll.TryLock(lockCtx)
		if !locked {
			errorMsg := fmt.Sprintf("failed to lock parent prefix %s", parentPrefix)
			r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, NewDomainError("%s", errorMsg)
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

	netboxIpRangeModel, err := r.NetboxClient.ReserveOrUpdateIpRange(ctx, ipRangeModel)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.IpRangeId == 0 {
			logger.Info("restoration hash mismatch, deleting ip range custom resource", "ip-range-start", o.Spec.StartAddress, "ip-range-end", o.Spec.EndAddress)
			if deleteErr := r.Delete(ctx, o); deleteErr != nil {
				return ctrl.Result{Requeue: true}, NewDomainError("failed to delete IpRange CR with restoration hash mismatch: %w", deleteErr)
			}
			// Object deleted - status update in deferred function will be ignored via client.IgnoreNotFound
			return ctrl.Result{}, nil
		}

		return ctrl.Result{Requeue: true}, NewDomainError("failed to reserve or update ip range in netbox: %w", err)
	}

	// 3. unlock lease of parent prefix
	if ll != nil {
		cancelLock()
		ll.UnlockWithRetry(ctx)
	}

	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[IPRManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		return ctrl.Result{Requeue: true}, NewDomainError("failed to generate managed custom fields annotation: %w", err)
	}

	// snapshot before annotation mutation for merge-patch
	patch := client.MergeFrom(o.DeepCopy())

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// patch object to store lastIpRangeMetadata annotation
	err = r.Patch(ctx, o, patch)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update status fields (set after r.Patch to avoid being overwritten by API response)
	o.Status.IpRangeId = int64(netboxIpRangeModel.GetId())
	o.Status.IpRangeUrl = config.GetBaseUrl() + "/ipam/ip-ranges/" + strconv.FormatInt(int64(netboxIpRangeModel.GetId()), 10)

	logger.Info("reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpRangeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpRange{}).
		Complete(r)
}

// updateStatus updates the IpRange status conditions based on the current state of the object.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *IpRangeReconciler) updateStatus(ctx context.Context, o *netboxv1.IpRange, statusBase *netboxv1.IpRange, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	if apierrors.IsConflict(err) {
		// Object was modified concurrently — skip status update, will retry on requeue
		return IgnoreDomainError(result, err)
	}

	logger.V(4).Info("updating iprange status")

	switch {
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpRangeReadyFalse, corev1.EventTypeNormal, reconcileErr)
	case o.Status.IpRangeUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpRangeReadyFalse, corev1.EventTypeWarning, reconcileErr)
	case reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpRangeReadyFalse, corev1.EventTypeWarning, reconcileErr)
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpRangeReadyTrue, corev1.EventTypeNormal, nil)
	}

	// Align resource version so the patch targets the latest revision
	statusBase.SetResourceVersion(o.GetResourceVersion())
	statusPatch := client.MergeFrom(statusBase)
	patchErr := r.Status().Patch(ctx, o, statusPatch)
	if patchErr != nil {
		patchErr = client.IgnoreNotFound(patchErr)
		if patchErr != nil {
			err = errors.Join(err, patchErr)
		}
	}

	return IgnoreDomainError(result, err)
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

	description := api.TruncateDescription(req.String() + " // " + o.Spec.Description)

	// check if created ip range contains entire description from spec
	_, found := strings.CutPrefix(description, req.String()+" // "+o.Spec.Description)
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
	err := r.Get(ctx, orLookupKey, ipRangeClaim)
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
