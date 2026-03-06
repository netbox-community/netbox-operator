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
	"errors"
	"fmt"
	"strings"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
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

const IpRangeClaimFinalizerName = "iprangeclaim.netbox.dev/finalizer"

// IpRangeClaimReconciler reconciles a IpRangeClaim object
type IpRangeClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpRangeClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpRangeClaim{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ipRange := &netboxv1.IpRange{}
	ipRangeName := o.Name
	ipRangeLookupKey := types.NamespacedName{
		Name:      ipRangeName,
		Namespace: o.Namespace,
	}

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		err = r.Get(ctx, ipRangeLookupKey, ipRange)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, IpRangeClaimFinalizerName)
		}

		if err = r.Delete(ctx, ipRange); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// requeue if owned iprange was still found
		return ctrl.Result{Requeue: true}, nil
	}

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, ipRangeLookupKey, reconcileResult, reconcileErr)
	}()

	err = r.Get(ctx, ipRangeLookupKey, ipRange)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		logger.V(4).Info("iprange object matching iprange claim was not found, creating new iprange object")

		ipRangeModel, cancelLock, res, err := r.restoreOrAssignIpRangeAndSetCondition(ctx, o)
		if cancelLock != nil {
			defer cancelLock()
		}
		if ipRangeModel == nil {
			return res, err
		}

		// create the IpRange CR
		ipRangeResource := generateIpRangeFromIpRangeClaim(ctx, o, ipRangeModel.StartAddress, ipRangeModel.EndAddress)
		err = controllerutil.SetControllerReference(o, ipRangeResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = addFinalizer(ctx, r.Client, o, IpRangeClaimFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, ipRangeResource)
		if err != nil {
			return ctrl.Result{}, NewDomainError("failed to create IpRange: %w", err)
		}
	} else {
		// update spec of IpRange object
		logger.V(4).Info("update iprange resource")
		ipRange.Spec = generateIpRangeSpec(o, ipRange.Spec.StartAddress, ipRange.Spec.EndAddress, logger)
		err = controllerutil.SetControllerReference(o, ipRange, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Update(ctx, ipRange)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("reconcile loop finished")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpRangeClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpRangeClaim{}).
		Owns(&netboxv1.IpRange{}).
		Complete(r)
}

// updateStatus updates the IpRangeClaim status based on the current state of the owned IpRange.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *IpRangeClaimReconciler) updateStatus(ctx context.Context, claim *netboxv1.IpRangeClaim, lookupKey types.NamespacedName, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	// Ensure status update is always called, even on early returns
	defer func() {
		updateErr := r.Status().Update(ctx, claim)
		if updateErr != nil {
			err = errors.Join(err, updateErr)
		}
		result, err = IgnoreDomainError(result, err)
	}()

	logger.V(4).Info("updating iprangeclaim status")

	// Fetch the latest IpRange object
	ipRange := &netboxv1.IpRange{}
	err = r.Client.Get(ctx, lookupKey, ipRange)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// IpRange doesn't exist yet
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpRangeAssignedFalse, corev1.EventTypeWarning, reconcileErr)
			// Preserve original result (e.g. RequeueAfter from lock contention)
			if result.IsZero() {
				result = ctrl.Result{RequeueAfter: 1 * time.Second}
			}
			err = nil
			return result, err
		}
		err = fmt.Errorf("failed to get IpRange for status update: %w", err)
		return result, err
	}

	// IpRange exists - report successful assignment if not already reported
	if apismeta.FindStatusCondition(claim.Status.Conditions, netboxv1.ConditionIpRangeAssignedTrue.Type) == nil || apismeta.IsStatusConditionFalse(claim.Status.Conditions, netboxv1.ConditionIpRangeAssignedTrue.Type) {
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpRangeAssignedTrue, corev1.EventTypeNormal,
			nil, fmt.Sprintf(" assigned ip range: %s-%s", ipRange.Spec.StartAddress, ipRange.Spec.EndAddress))
	}
	// Update status based on IpRange readiness
	if apismeta.IsStatusConditionTrue(ipRange.Status.Conditions, netboxv1.ConditionIpRangeReadyTrue.Type) {
		logger.V(4).Info("iprange status ready true")
		var genErr error
		claim.Status, genErr = r.generateIpRangeClaimStatus(claim, ipRange)
		if genErr != nil {
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpRangeClaimReadyFalseStatusGen, corev1.EventTypeWarning, genErr)
			return result, err
		}
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpRangeClaimReadyTrue, corev1.EventTypeNormal, nil)
	} else {
		logger.V(4).Info("iprange status ready false")
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpRangeClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
	}

	return result, err
}

func (r *IpRangeClaimReconciler) tryLockOnParentPrefix(ctx context.Context, o *netboxv1.IpRangeClaim) (*leaselocker.LeaseLocker, context.CancelFunc, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	leaseLockerNSN := types.NamespacedName{
		Name:      convertCIDRToLeaseLockName(o.Spec.ParentPrefix),
		Namespace: r.OperatorNamespace,
	}

	claimNSN := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}

	ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, claimNSN.String())
	if err != nil {
		return nil, nil, ctrl.Result{}, err
	}

	lockCtx, cancel := context.WithCancel(ctx)

	// try to lock lease for parent prefix
	locked := ll.TryLock(lockCtx)
	if !locked {
		cancel()
		// lock for parent prefix was not available, rescheduling
		logger.Info(fmt.Sprintf("failed to lock parent prefix %s", o.Spec.ParentPrefix))
		r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", "failed to lock parent prefix %s",
			o.Spec.ParentPrefix)
		return nil, nil, ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	logger.V(4).Info(fmt.Sprintf("successfully locked parent prefix %s", o.Spec.ParentPrefix))

	return ll, cancel, ctrl.Result{}, nil
}

func (r *IpRangeClaimReconciler) generateIpRangeClaimStatus(o *netboxv1.IpRangeClaim, ipRange *netboxv1.IpRange) (netboxv1.IpRangeClaimStatus, error) {
	startAddressDotDecimal := strings.Split(ipRange.Spec.StartAddress, "/")[0]
	endAddressDotDecimal := strings.Split(ipRange.Spec.EndAddress, "/")[0]

	availableIps, err := r.NetboxClient.GetAvailableIpAddressesByIpRange(ipRange.Status.IpRangeId)
	if err != nil {
		return netboxv1.IpRangeClaimStatus{}, err
	}

	ipAddresses := []string{}
	ipAddressesDotDecimal := []string{}
	for _, ip := range availableIps.Payload {
		ipAddresses = append(ipAddresses, ip.Address)
		ipAddressesDotDecimal = append(ipAddressesDotDecimal, strings.Split(ip.Address, "/")[0])
	}

	return netboxv1.IpRangeClaimStatus{
		IpRange:                fmt.Sprintf("%s-%s", ipRange.Spec.StartAddress, ipRange.Spec.EndAddress),
		IpRangeDotDecimal:      fmt.Sprintf("%s-%s", startAddressDotDecimal, endAddressDotDecimal),
		IpAddresses:            ipAddresses,
		IpAddressesDotDecimal:  ipAddressesDotDecimal,
		StartAddress:           ipRange.Spec.StartAddress,
		StartAddressDotDecimal: startAddressDotDecimal,
		EndAddress:             ipRange.Spec.EndAddress,
		EndAddressDotDecimal:   endAddressDotDecimal,
		IpRangeName:            ipRange.Name,
		Conditions:             o.Status.Conditions,
	}, nil
}

func (r *IpRangeClaimReconciler) restoreOrAssignIpRangeAndSetCondition(ctx context.Context, o *netboxv1.IpRangeClaim) (*models.IpRange, context.CancelFunc, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	ll, cancelLock, res, err := r.tryLockOnParentPrefix(ctx, o)
	if err != nil || ll == nil {
		return nil, nil, res, err
	}

	h := generateIpRangeRestorationHash(o)
	ipRangeModel, err := r.NetboxClient.RestoreExistingIpRangeByHash(h)
	if err != nil {
		return nil, cancelLock, ctrl.Result{Requeue: true}, NewDomainError("failed to restore existing ip range by hash: %w", err)
	}

	if ipRangeModel == nil {
		// ip range cannot be restored from netbox
		// assign new available ip range
		ipRangeModel, err = r.NetboxClient.GetAvailableIpRangeByClaim(
			ctx,
			&models.IpRangeClaim{
				ParentPrefix: o.Spec.ParentPrefix,
				Size:         o.Spec.Size,
				Metadata: &models.NetboxMetadata{
					Tenant: o.Spec.Tenant,
				},
			},
		)
		if err != nil {
			return nil, cancelLock, ctrl.Result{Requeue: true}, NewDomainError("failed to get available ip range: %w", err)
		}
		logger.V(4).Info(fmt.Sprintf("ip range is not reserved in netbox, assigned new ip range: %s-%s", ipRangeModel.StartAddress, ipRangeModel.EndAddress))
	} else {
		// reassign reserved ip range from netbox
		if int(ipRangeModel.Size) != o.Spec.Size {
			return nil, cancelLock, ctrl.Result{Requeue: true}, NewDomainError("ip range size mismatch: expected %d, got %d", o.Spec.Size, ipRangeModel.Size)
		}
		logger.V(4).Info(fmt.Sprintf("reassign reserved ip range from netbox, range: %s-%s", ipRangeModel.StartAddress, ipRangeModel.EndAddress))
	}
	return ipRangeModel, cancelLock, ctrl.Result{}, nil
}
