/*
Copyright 2026 Swisscom (Schweiz) AG.

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
	"github.com/netbox-community/netbox-operator/pkg/scheduler"

	"github.com/swisscom/leaselocker"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const VlanFinalizerName = "vlan.netbox.dev/finalizer"
const VlanManagedCustomFieldsAnnotationName = "vlan.netbox.dev/managed-custom-fields"

// VlanReconciler reconciles a Vlan object
type VlanReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=vlans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=vlans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=vlans/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.Vlan{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change (VlanId, conditions, etc.).
	statusBase := o.DeepCopy()

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.Info("reconcile loop finished")
	}()

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(o, VlanFinalizerName) {
			return ctrl.Result{}, nil
		}

		if !o.Spec.PreserveInNetbox {
			if o.Status.VlanId > math.MaxInt32 {
				return ctrl.Result{}, fmt.Errorf("reconciliation of vlans with id's larger than 2147483647 is not supported")
			}
			if err := r.NetboxClient.DeleteVlan(ctx, int32(o.Status.VlanId)); err != nil {
				return ctrl.Result{}, NewDomainError("failed to delete vlan in netbox: %w", err)
			}
		}

		return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, VlanFinalizerName)
	}

	// if PreserveInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox {
		err = addFinalizer(ctx, r.Client, o, VlanFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. try to lock lease of the VID range if Vlan status condition is
	// not true, is owned by a range-based VlanClaim, and hasn't been created
	// in NetBox yet. Claims with an explicit VID don't draw from a shared
	// pool, so there's nothing to serialize against.
	or := o.OwnerReferences
	var ll *leaselocker.LeaseLocker
	var cancelLock context.CancelFunc
	if len(or) > 0 && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		leaseLockerNSN, owner, rangeDesc, ok, err := r.getLeaseLockerNSNandOwner(ctx, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		if ok {
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
				errorMsg := fmt.Sprintf("failed to lock vid range %s", rangeDesc)
				r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "FailedToLockVidRange", errorMsg)
				return ctrl.Result{
					RequeueAfter: 2 * time.Second,
				}, NewDomainError("%s", errorMsg)
			}
			logger.V(4).Info(fmt.Sprintf("successfully locked vid range %s", rangeDesc))
		}
	}

	// 2. reserve or update vlan in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	vlanModel, err := r.generateNetboxVlanModelFromVlanSpec(o, req, annotations[VlanManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxVlanModel, statusUpToDate, err := r.NetboxClient.ReserveOrUpdateVlan(ctx, vlanModel, o)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.VlanId == 0 {
			logger.Info("conflict in claimed vlan, deleting vlan custom resource", "vid",
				o.Spec.Vid, "error", err)
			if deleteErr := r.Delete(ctx, o); deleteErr != nil {
				return ctrl.Result{}, NewDomainError("failed to delete Vlan CR with conflict: %w", deleteErr)
			}
			// Object deleted - status update in deferred function will be ignored via client.IgnoreNotFound
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, NewDomainError("%w", err)
	}

	// 3. unlock lease of the vid range
	if ll != nil {
		cancelLock()
		ll.UnlockWithRetry(ctx)
	}

	// 4. if no change, then end loop
	if statusUpToDate {
		return ctrl.Result{}, nil
	}

	// 4.1 update annotation
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[VlanManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		return ctrl.Result{}, NewDomainError("failed to generate managed custom fields annotation: %w", err)
	}

	// snapshot before annotation mutation for merge-patch
	patch := client.MergeFrom(o.DeepCopy())

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// patch object to store managed custom fields annotation
	err = r.Patch(ctx, o, patch)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update status fields (set after r.Patch to avoid being overwritten by API response)
	o.Status.VlanId = int64(netboxVlanModel.GetId())
	o.Status.VlanUrl = config.GetBaseUrl() + "/ipam/vlans/" + strconv.FormatInt(int64(netboxVlanModel.GetId()), 10)
	if netboxVlanModel.LastUpdated.IsSet() {
		o.Status.LastUpdated = metav1.NewTime(*netboxVlanModel.LastUpdated.Get())
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.Vlan{}).
		Complete(r)
}

// updateStatus updates the Vlan status conditions based on the current state of the object.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *VlanReconciler) updateStatus(ctx context.Context, o *netboxv1.Vlan, statusBase *netboxv1.Vlan, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	if apierrors.IsConflict(err) {
		// Object was modified concurrently — skip status update, will retry on requeue
		return IgnoreDomainError(result, err)
	}

	logger.V(4).Info("updating vlan status")

	switch {
	case !o.DeletionTimestamp.IsZero() && reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanReadyFalseDeletionFailed, corev1.EventTypeWarning, reconcileErr)
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanReadyFalseDeletionInProgress, corev1.EventTypeNormal, nil)
	case o.Status.VlanUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("vid: %d", o.Spec.Vid))
	case reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("vid: %d", o.Spec.Vid))
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanReadyTrue, corev1.EventTypeNormal, nil)
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

func (r *VlanReconciler) generateNetboxVlanModelFromVlanSpec(o *netboxv1.Vlan, req ctrl.Request, lastVlanMetadata string) (*models.Vlan, error) {
	// unmarshal lastVlanMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastVlanMetadata != "" {
		if err := json.Unmarshal([]byte(lastVlanMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastVlanMetadata annotation: %w", err)
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

	// check if created vlan contains entire description from spec
	_, found := strings.CutPrefix(description, req.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "VlanDescriptionTruncated", "vlan was created with truncated description")
	}

	return &models.Vlan{
		Name:   o.Spec.Name,
		Vid:    o.Spec.Vid,
		Status: o.Spec.Status,
		Metadata: &models.NetboxMetadata{
			Comments:    o.Spec.Comments,
			Custom:      netboxCustomFields,
			Description: description,
			Site:        o.Spec.Site,
			Tenant:      o.Spec.Tenant,
		},
	}, nil
}

// getLeaseLockerNSNandOwner returns the lease lock identity for the Vlan's
// owning claim's VID range. ok is false when the owning claim uses an
// explicit VID instead of a range, in which case there's no shared pool to
// lock.
func (r *VlanReconciler) getLeaseLockerNSNandOwner(ctx context.Context, o *netboxv1.Vlan) (nsn types.NamespacedName, owner string, rangeDesc string, ok bool, err error) {
	orLookupKey := types.NamespacedName{
		Name:      o.ObjectMeta.OwnerReferences[0].Name,
		Namespace: o.Namespace,
	}

	vlanClaim := &netboxv1.VlanClaim{}
	err = r.Get(ctx, orLookupKey, vlanClaim)
	if err != nil {
		return types.NamespacedName{}, "", "", false, err
	}

	if vlanClaim.Spec.VidRangeStart == 0 && vlanClaim.Spec.VidRangeEnd == 0 {
		return types.NamespacedName{}, "", "", false, nil
	}

	rangeDesc = fmt.Sprintf("%s:%d-%d", vlanClaim.Spec.Site, vlanClaim.Spec.VidRangeStart, vlanClaim.Spec.VidRangeEnd)

	leaseLockerNSN := types.NamespacedName{
		Name:      convertVlanRangeToLeaseLockName(vlanClaim.Spec.Site, vlanClaim.Spec.VidRangeStart, vlanClaim.Spec.VidRangeEnd),
		Namespace: r.OperatorNamespace,
	}

	return leaseLockerNSN, orLookupKey.String(), rangeDesc, true, nil
}
