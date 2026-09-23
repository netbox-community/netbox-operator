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

const L2VPNClaimFinalizerName = "l2vpnclaim.netbox.dev/finalizer"

// L2VPNClaimReconciler reconciles a L2VPNClaim object
type L2VPNClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=l2vpnclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=l2vpnclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=l2vpnclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *L2VPNClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.L2VPNClaim{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change.
	statusBase := o.DeepCopy()

	l2vpn := &netboxv1.L2VPN{}
	l2vpnLookupKey := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		err = r.Get(ctx, l2vpnLookupKey, l2vpn)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, L2VPNClaimFinalizerName)
		}

		if err = r.Delete(ctx, l2vpn); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// requeue if owned l2vpn was still found
		return ctrl.Result{Requeue: true}, nil
	}

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, l2vpnLookupKey, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.Info("reconcile loop finished")
	}()

	err = r.Get(ctx, l2vpnLookupKey, l2vpn)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		logger.V(4).Info("l2vpn object matching l2vpn claim was not found, creating new l2vpn object")

		identifier, cancelLock, res, err := r.restoreOrAssignL2VPNAndSetCondition(ctx, o)
		if cancelLock != nil {
			defer cancelLock()
		}
		if identifier == nil {
			return res, err
		}

		// create the L2VPN CR
		l2vpnResource := generateL2VPNFromL2VPNClaim(ctx, o, *identifier)
		err = controllerutil.SetControllerReference(o, l2vpnResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = addFinalizer(ctx, r.Client, o, L2VPNClaimFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, l2vpnResource)
		if err != nil {
			return ctrl.Result{}, NewDomainError("failed to create L2VPN: %w", err)
		}
	} else {
		// update spec of L2VPN object
		logger.V(4).Info("update l2vpn resource")
		l2vpn.Spec = generateL2VPNSpec(o, l2vpn.Spec.Identifier, logger)
		err = controllerutil.SetControllerReference(o, l2vpn, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Update(ctx, l2vpn)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *L2VPNClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.L2VPNClaim{}).
		Owns(&netboxv1.L2VPN{}).
		Complete(r)
}

// updateStatus updates the L2VPNClaim status based on the current state of the owned L2VPN.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *L2VPNClaimReconciler) updateStatus(ctx context.Context, claim *netboxv1.L2VPNClaim, statusBase *netboxv1.L2VPNClaim, lookupKey types.NamespacedName, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
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

	logger.V(4).Info("updating l2vpnclaim status")

	// Fetch the latest L2VPN object
	l2vpn := &netboxv1.L2VPN{}
	err = r.Client.Get(ctx, lookupKey, l2vpn)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// L2VPN doesn't exist yet
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionL2VPNAssignedFalse, corev1.EventTypeWarning, reconcileErr)
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionL2VPNClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
			// Preserve original result (e.g. RequeueAfter from lock contention)
			if result.IsZero() {
				result = ctrl.Result{RequeueAfter: 1 * time.Second}
			}
			err = nil
			return result, err
		}
		err = fmt.Errorf("failed to get L2VPN for status update: %w", err)
		return result, err
	}

	// L2VPN exists - report successful assignment if not already reported
	if apismeta.FindStatusCondition(claim.Status.Conditions, netboxv1.ConditionL2VPNAssignedTrue.Type) == nil || apismeta.IsStatusConditionFalse(claim.Status.Conditions, netboxv1.ConditionL2VPNAssignedTrue.Type) {
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionL2VPNAssignedTrue, corev1.EventTypeNormal,
			nil, fmt.Sprintf(" assigned identifier: %d", l2vpn.Spec.Identifier))
	}
	// Update status based on L2VPN readiness
	if apismeta.IsStatusConditionTrue(l2vpn.Status.Conditions, netboxv1.ConditionL2VPNReadyTrue.Type) {
		logger.V(4).Info("l2vpn status ready true")
		claim.Status.Identifier = l2vpn.Spec.Identifier
		claim.Status.L2VPNName = l2vpn.Name
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionL2VPNClaimReadyTrue, corev1.EventTypeNormal, nil)
	} else {
		logger.V(4).Info("l2vpn status ready false")
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionL2VPNClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
	}

	return result, err
}

// tryLockL2VPNIdentifier serializes concurrent claims — range-based or
// explicit-identifier — against NetBox's shared VNI namespace (see
// l2vpnIdentifierLockName), so that no two claims can be assigned the same
// identifier.
func (r *L2VPNClaimReconciler) tryLockL2VPNIdentifier(ctx context.Context, o *netboxv1.L2VPNClaim) (ll *leaselocker.LeaseLocker, cleanup context.CancelFunc, res ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	identifierDesc := describeL2VPNClaimIdentifier(o)
	leaseLockerNSN := types.NamespacedName{
		Name:      l2vpnIdentifierLockName,
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
		logger.Info(fmt.Sprintf("failed to lock l2vpn identifiers for %s", identifierDesc))
		r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockIdentifierRange", "failed to lock l2vpn identifiers for %s",
			identifierDesc)
		return nil, nil, ctrl.Result{RequeueAfter: 2 * time.Second}, NewDomainError("failed to lock l2vpn identifiers for %s", identifierDesc)
	}
	logger.V(4).Info(fmt.Sprintf("successfully locked l2vpn identifiers for %s", identifierDesc))

	cleanup = func() {
		cancel()
		ll.UnlockWithRetry(ctx)
	}
	return ll, cleanup, ctrl.Result{}, nil
}

// describeL2VPNClaimIdentifier renders the claim's requested identifier range
// for log/event messages.
func describeL2VPNClaimIdentifier(o *netboxv1.L2VPNClaim) string {
	return fmt.Sprintf("%s:%d-%d", o.Spec.Type, o.Spec.IdentifierRangeStart, o.Spec.IdentifierRangeEnd)
}

func (r *L2VPNClaimReconciler) restoreOrAssignL2VPNAndSetCondition(ctx context.Context, o *netboxv1.L2VPNClaim) (*int64, context.CancelFunc, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	_, cancelLock, res, err := r.tryLockL2VPNIdentifier(ctx, o)
	if err != nil {
		return nil, nil, res, err
	}

	h := generateL2VPNRestorationHash(o)
	l2vpnModel, err := r.NetboxClient.RestoreExistingL2VPNByHash(ctx, h)
	if err != nil {
		return nil, cancelLock, ctrl.Result{}, NewDomainError("%w", err)
	}

	if l2vpnModel != nil {
		logger.V(4).Info(fmt.Sprintf("reassign reserved l2vpn identifier from netbox: %d", l2vpnModel.Identifier))
		return &l2vpnModel.Identifier, cancelLock, ctrl.Result{}, nil
	}

	// If l2vpn cannot be restored from netbox, assign a new available
	// identifier from the requested range. An exact VNI claim is expressed
	// as a range of one (identifierRangeStart == identifierRangeEnd), which
	// GetAvailableL2VPNIdentifierByClaim checks for availability the same
	// way as any other range.
	l2vpnModel, err = r.NetboxClient.GetAvailableL2VPNIdentifierByClaim(
		ctx,
		&models.L2VPNClaim{
			Type:                 o.Spec.Type,
			IdentifierRangeStart: o.Spec.IdentifierRangeStart,
			IdentifierRangeEnd:   o.Spec.IdentifierRangeEnd,
			Metadata: &models.NetboxMetadata{
				Tenant: o.Spec.Tenant,
			},
		},
	)
	if err != nil {
		return nil, cancelLock, ctrl.Result{}, NewDomainError("%w", err)
	}
	logger.V(4).Info(fmt.Sprintf("l2vpn is not reserved in netbox, assigned new identifier: %d", l2vpnModel.Identifier))
	return &l2vpnModel.Identifier, cancelLock, ctrl.Result{}, nil
}
