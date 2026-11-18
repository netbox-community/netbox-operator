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

const RouteTargetClaimFinalizerName = "routetargetclaim.netbox.dev/finalizer"

// RouteTargetClaimReconciler reconciles a RouteTargetClaim object
type RouteTargetClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=routetargetclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=routetargetclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=routetargetclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RouteTargetClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("reconcile loop started")

	o := &netboxv1.RouteTargetClaim{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change.
	statusBase := o.DeepCopy()

	routeTarget := &netboxv1.RouteTarget{}
	routeTargetLookupKey := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		err = r.Get(ctx, routeTargetLookupKey, routeTarget)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, RouteTargetClaimFinalizerName)
		}

		if err = r.Delete(ctx, routeTarget); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// requeue if owned route target was still found
		return ctrl.Result{Requeue: true}, nil
	}

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, routeTargetLookupKey, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.V(4).Info("reconcile loop finished")
	}()

	// 2. check if the matching RouteTarget object exists
	err = r.Get(ctx, routeTargetLookupKey, routeTarget)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		logger.V(4).Info("route target object matching route target claim was not found, creating new route target object")

		routeTargetName, cancelLock, res, err := r.restoreOrAssignRouteTargetAndSetCondition(ctx, o)
		if cancelLock != nil {
			defer cancelLock()
		}
		if routeTargetName == nil {
			return res, err
		}

		// 7.a create the RouteTarget object
		routeTargetResource := generateRouteTargetFromRouteTargetClaim(ctx, o, *routeTargetName)
		err = controllerutil.SetControllerReference(o, routeTargetResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = addFinalizer(ctx, r.Client, o, RouteTargetClaimFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, routeTargetResource)
		if err != nil {
			return ctrl.Result{}, NewDomainError("failed to create RouteTarget: %w", err)
		}
	} else {
		// 7.b update fields of the RouteTarget object
		logger.V(4).Info("update route target resource")
		routeTarget.Spec = generateRouteTargetSpec(o, routeTarget.Spec.RouteTarget, logger)
		err = controllerutil.SetControllerReference(o, routeTarget, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Update(ctx, routeTarget)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RouteTargetClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.RouteTargetClaim{}).
		Owns(&netboxv1.RouteTarget{}).
		Complete(r)
}

// updateStatus updates the RouteTargetClaim status based on the current state of the owned RouteTarget.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *RouteTargetClaimReconciler) updateStatus(ctx context.Context, claim *netboxv1.RouteTargetClaim, statusBase *netboxv1.RouteTargetClaim, lookupKey types.NamespacedName, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
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

	logger.V(4).Info("updating routetargetclaim status")

	// Fetch the latest RouteTarget object
	routeTarget := &netboxv1.RouteTarget{}
	err = r.Client.Get(ctx, lookupKey, routeTarget)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// RouteTarget doesn't exist yet
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionRouteTargetAssignedFalse, corev1.EventTypeWarning, reconcileErr)
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionRouteTargetClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
			// Preserve original result (e.g. RequeueAfter from lock contention)
			if result.IsZero() {
				result = ctrl.Result{RequeueAfter: 1 * time.Second}
			}
			err = nil
			return result, err
		}
		err = fmt.Errorf("failed to get RouteTarget for status update: %w", err)
		return result, err
	}

	// RouteTarget exists - report successful assignment if not already reported
	if apismeta.FindStatusCondition(claim.Status.Conditions, netboxv1.ConditionRouteTargetAssignedTrue.Type) == nil || apismeta.IsStatusConditionFalse(claim.Status.Conditions, netboxv1.ConditionRouteTargetAssignedTrue.Type) {
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionRouteTargetAssignedTrue, corev1.EventTypeNormal,
			nil, fmt.Sprintf(" assigned route target: %s", routeTarget.Spec.RouteTarget))
	}
	// Update status based on RouteTarget readiness
	if apismeta.IsStatusConditionTrue(routeTarget.Status.Conditions, netboxv1.ConditionRouteTargetReadyTrue.Type) {
		logger.V(4).Info("route target status ready true")
		claim.Status.RouteTarget = routeTarget.Spec.RouteTarget
		claim.Status.RouteTargetName = routeTarget.Name
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionRouteTargetClaimReadyTrue, corev1.EventTypeNormal, nil)
	} else {
		logger.V(4).Info("route target status ready false")
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionRouteTargetClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
	}

	return result, err
}

// tryLockRouteTargetAsn serializes concurrent claims for the same ASN
// (see routeTargetLockName), so that no two claims can be assigned the same
// route target.
func (r *RouteTargetClaimReconciler) tryLockRouteTargetAsn(ctx context.Context, o *netboxv1.RouteTargetClaim) (ll *leaselocker.LeaseLocker, cleanup context.CancelFunc, res ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	rangeDesc := describeRouteTargetClaimRange(o)
	leaseLockerNSN := types.NamespacedName{
		Name:      routeTargetLockName(o.Spec.Asn),
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
		logger.Info(fmt.Sprintf("failed to lock route targets for %s", rangeDesc))
		r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockRouteTargetRange", "failed to lock route targets for %s",
			rangeDesc)
		return nil, nil, ctrl.Result{RequeueAfter: 2 * time.Second}, NewDomainError("failed to lock route targets for %s", rangeDesc)
	}
	logger.V(4).Info(fmt.Sprintf("successfully locked route targets for %s", rangeDesc))

	cleanup = func() {
		cancel()
		ll.UnlockWithRetry(ctx)
	}
	return ll, cleanup, ctrl.Result{}, nil
}

// describeRouteTargetClaimRange renders the claim's requested route target
// range for log/event messages.
func describeRouteTargetClaimRange(o *netboxv1.RouteTargetClaim) string {
	return fmt.Sprintf("%d:%d-%d", o.Spec.Asn, o.Spec.AssignedNumberRangeStart, o.Spec.AssignedNumberRangeEnd)
}

func (r *RouteTargetClaimReconciler) restoreOrAssignRouteTargetAndSetCondition(ctx context.Context, o *netboxv1.RouteTargetClaim) (*string, context.CancelFunc, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	_, cancelLock, res, err := r.tryLockRouteTargetAsn(ctx, o)
	if err != nil {
		return nil, nil, res, err
	}

	h := generateRouteTargetRestorationHash(o)
	routeTargetModel, err := r.NetboxClient.RestoreExistingRouteTargetByHash(ctx, o.Spec.Asn, h)
	if err != nil {
		return nil, cancelLock, ctrl.Result{}, NewDomainError("%w", err)
	}

	if routeTargetModel != nil {
		logger.V(4).Info(fmt.Sprintf("reassign reserved route target from netbox: %s", routeTargetModel.Name))
		return &routeTargetModel.Name, cancelLock, ctrl.Result{}, nil
	}

	// If route target cannot be restored from netbox, assign a new available
	// route target from the requested range. An exact route target claim is
	// expressed as a range of one (assignedNumberRangeStart ==
	// assignedNumberRangeEnd), which GetAvailableRouteTargetByClaim checks
	// for availability the same way as any other range.
	routeTargetModel, err = r.NetboxClient.GetAvailableRouteTargetByClaim(
		ctx,
		&models.RouteTargetClaim{
			Asn:                      o.Spec.Asn,
			AssignedNumberRangeStart: o.Spec.AssignedNumberRangeStart,
			AssignedNumberRangeEnd:   o.Spec.AssignedNumberRangeEnd,
			Metadata: &models.NetboxMetadata{
				Tenant: o.Spec.Tenant,
			},
		},
	)
	if err != nil {
		return nil, cancelLock, ctrl.Result{}, NewDomainError("%w", err)
	}
	logger.V(4).Info(fmt.Sprintf("route target is not reserved in netbox, assigned new route target: %s", routeTargetModel.Name))
	return &routeTargetModel.Name, cancelLock, ctrl.Result{}, nil
}
