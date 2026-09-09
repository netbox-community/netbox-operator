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
	"errors"
	"fmt"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/scheduler"

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

const VlanClaimFinalizerName = "vlanclaim.netbox.dev/finalizer"

// VlanClaimReconciler reconciles a VlanClaim object
type VlanClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=vlanclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=vlanclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=vlanclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VlanClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.VlanClaim{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change.
	statusBase := o.DeepCopy()

	vlan := &netboxv1.Vlan{}
	vlanLookupKey := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		err = r.Get(ctx, vlanLookupKey, vlan)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, VlanClaimFinalizerName)
		}

		if err = r.Delete(ctx, vlan); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// requeue if owned vlan was still found
		return ctrl.Result{Requeue: true}, nil
	}

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, vlanLookupKey, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.Info("reconcile loop finished")
	}()

	err = r.Get(ctx, vlanLookupKey, vlan)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		logger.V(4).Info("vlan object matching vlan claim was not found, creating new vlan object")

		vid, cancelLock, res, err := r.restoreOrAssignVlanAndSetCondition(ctx, o)
		if cancelLock != nil {
			defer cancelLock()
		}
		if vid == nil {
			return res, err
		}

		// create the Vlan CR
		vlanResource := generateVlanFromVlanClaim(ctx, o, *vid)
		err = controllerutil.SetControllerReference(o, vlanResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = addFinalizer(ctx, r.Client, o, VlanClaimFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, vlanResource)
		if err != nil {
			return ctrl.Result{}, NewDomainError("failed to create Vlan: %w", err)
		}
	} else {
		// update spec of Vlan object
		logger.V(4).Info("update vlan resource")
		vlan.Spec = generateVlanSpec(o, vlan.Spec.Vid, logger)
		err = controllerutil.SetControllerReference(o, vlan, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Update(ctx, vlan)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VlanClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.VlanClaim{}).
		Owns(&netboxv1.Vlan{}).
		Complete(r)
}

// updateStatus updates the VlanClaim status based on the current state of the owned Vlan.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *VlanClaimReconciler) updateStatus(ctx context.Context, claim *netboxv1.VlanClaim, statusBase *netboxv1.VlanClaim, lookupKey types.NamespacedName, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	// Ensure status update is always called, even on early returns
	defer func() {
		if apierrors.IsConflict(err) {
			// Object was modified concurrently — skip status update, will retry on requeue
			result, err = IgnoreDomainError(result, err)
			return
		}
		// Align resource version so the patch targets the latest revision
		statusBase.SetResourceVersion(claim.GetResourceVersion())
		statusPatch := client.MergeFrom(statusBase)
		patchErr := r.Status().Patch(ctx, claim, statusPatch)
		if patchErr != nil {
			patchErr = client.IgnoreNotFound(patchErr)
			if patchErr != nil {
				err = errors.Join(err, patchErr)
			}
		}
		result, err = IgnoreDomainError(result, err)
	}()

	logger.V(4).Info("updating vlanclaim status")

	// Fetch the latest Vlan object
	vlan := &netboxv1.Vlan{}
	err = r.Client.Get(ctx, lookupKey, vlan)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Vlan doesn't exist yet
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionVlanAssignedFalse, corev1.EventTypeWarning, reconcileErr)
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionVlanClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
			// Preserve original result (e.g. RequeueAfter from lock contention)
			if result.IsZero() {
				result = ctrl.Result{RequeueAfter: 1 * time.Second}
			}
			err = nil
			return result, err
		}
		err = fmt.Errorf("failed to get Vlan for status update: %w", err)
		return result, err
	}

	// Vlan exists - report successful assignment if not already reported
	if apismeta.FindStatusCondition(claim.Status.Conditions, netboxv1.ConditionVlanAssignedTrue.Type) == nil || apismeta.IsStatusConditionFalse(claim.Status.Conditions, netboxv1.ConditionVlanAssignedTrue.Type) {
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionVlanAssignedTrue, corev1.EventTypeNormal,
			nil, fmt.Sprintf(" assigned vid: %d", vlan.Spec.Vid))
	}
	// Update status based on Vlan readiness
	if apismeta.IsStatusConditionTrue(vlan.Status.Conditions, netboxv1.ConditionVlanReadyTrue.Type) {
		logger.V(4).Info("vlan status ready true")
		claim.Status.Vid = vlan.Spec.Vid
		claim.Status.VlanName = vlan.Name
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionVlanClaimReadyTrue, corev1.EventTypeNormal, nil)
	} else {
		logger.V(4).Info("vlan status ready false")
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionVlanClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
	}

	return result, err
}

// tryLockVlanVid serializes concurrent claims — range-based or
// explicit-VID — against the shared per-site VID namespace (see
// vlanIdentifierLockName), so that no two claims for the same site can be
// assigned the same VID.
func (r *VlanClaimReconciler) tryLockVlanVid(ctx context.Context, o *netboxv1.VlanClaim) (ll *leaselocker.LeaseLocker, cleanup context.CancelFunc, res ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	identifierDesc := describeVlanClaimIdentifier(o)
	leaseLockerNSN := types.NamespacedName{
		Name:      vlanIdentifierLockName(o.Spec.Site),
		Namespace: r.OperatorNamespace,
	}

	claimNSN := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}

	ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, claimNSN.String())
	if err != nil {
		return nil, nil, ctrl.Result{}, err
	}

	lockCtx, cancel := context.WithTimeout(ctx, lockAcquireTimeout)

	locked := ll.TryLock(lockCtx)
	if !locked {
		cancel()
		logger.Info(fmt.Sprintf("failed to lock vlan vids for %s", identifierDesc))
		r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockVidRange", "failed to lock vlan vids for %s",
			identifierDesc)
		return nil, nil, ctrl.Result{RequeueAfter: 2 * time.Second}, NewDomainError("failed to lock vlan vids for %s", identifierDesc)
	}
	logger.V(4).Info(fmt.Sprintf("successfully locked vlan vids for %s", identifierDesc))

	cleanup = func() {
		cancel()
		ll.UnlockWithRetry(ctx)
	}
	return ll, cleanup, ctrl.Result{}, nil
}

// describeVlanClaimIdentifier renders the claim's requested VID(s) for
// log/event messages.
func describeVlanClaimIdentifier(o *netboxv1.VlanClaim) string {
	if o.Spec.VidRangeStart == 0 && o.Spec.VidRangeEnd == 0 {
		return fmt.Sprintf("%s:%d", o.Spec.Site, o.Spec.Vid)
	}
	return fmt.Sprintf("%s:%d-%d", o.Spec.Site, o.Spec.VidRangeStart, o.Spec.VidRangeEnd)
}

func (r *VlanClaimReconciler) restoreOrAssignVlanAndSetCondition(ctx context.Context, o *netboxv1.VlanClaim) (*int32, context.CancelFunc, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	_, cancelLock, res, err := r.tryLockVlanVid(ctx, o)
	if err != nil {
		return nil, nil, res, err
	}

	h := generateVlanRestorationHash(o)
	vlanModel, err := r.NetboxClient.RestoreExistingVlanByHash(ctx, h)
	if err != nil {
		return nil, cancelLock, ctrl.Result{}, NewDomainError("%w", err)
	}

	if vlanModel != nil {
		logger.V(4).Info(fmt.Sprintf("reassign reserved vlan vid from netbox: %d", vlanModel.Vid))
		return &vlanModel.Vid, cancelLock, ctrl.Result{}, nil
	}

	// vlan cannot be restored from netbox. We need to resolve the VID range
	// to check for availability. An explicit vid is treated as a range of one,
	//  so it goes through the same free/used scan as a range claim.
	// GetAvailableVlanByClaim errors with ErrVlanRangeExhausted if that single VID
	// is already taken.
	vidRangeStart, vidRangeEnd := o.Spec.VidRangeStart, o.Spec.VidRangeEnd
	if o.Spec.Vid != 0 {
		vidRangeStart, vidRangeEnd = o.Spec.Vid, o.Spec.Vid
	}

	vlanModel, err = r.NetboxClient.GetAvailableVlanByClaim(
		ctx,
		&models.VlanClaim{
			VidRangeStart: vidRangeStart,
			VidRangeEnd:   vidRangeEnd,
			Metadata: &models.NetboxMetadata{
				Site:   o.Spec.Site,
				Tenant: o.Spec.Tenant,
			},
		},
	)
	if err != nil {
		return nil, cancelLock, ctrl.Result{}, NewDomainError("%w", err)
	}
	logger.V(4).Info(fmt.Sprintf("vlan is not reserved in netbox, assigned new vid: %d", vlanModel.Vid))
	return &vlanModel.Vid, cancelLock, ctrl.Result{}, nil
}
