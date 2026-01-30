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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
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

// IpAddressClaimReconciler reconciles a IpAddressClaim object
type IpAddressClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddressclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddressclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddressclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpAddressClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	/* 0. check if the matching IpAddressClaim object exists */
	o := &netboxv1.IpAddressClaim{}
	if err := r.Client.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// Defer status update to ensure it happens regardless of how we exit
	// This follows Kubernetes controller best practices
	// The deferred function captures the return values to include error context in status
	defer func() {
		if statusErr := r.updateStatus(ctx, o, req.NamespacedName, err, debugLogger); statusErr != nil {
			logger.Error(statusErr, "failed to update IpAddressClaim status in deferred call")
		}

		// If err is a StatusError, we've reported it in status conditions, so return nil to controller-runtime
		// This prevents exponential backoff for user-facing errors that are already visible in status
		// Regular errors are still returned to trigger retry with backoff
		if err != nil && IsStatusError(err) {
			err = nil
		}
	}()

	// 1. check if matching IpAddress object already exists
	ipAddress := &netboxv1.IpAddress{}
	ipAddressName := o.ObjectMeta.Name
	ipAddressLookupKey := types.NamespacedName{
		Name:      ipAddressName,
		Namespace: o.Namespace,
	}

	err = r.Client.Get(ctx, ipAddressLookupKey, ipAddress)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to get IpAddress: %w", err)
		}

		debugLogger.Info("ipaddress object matching ipaddress claim was not found, creating new ipaddress object")

		// 2. check if lease for parent prefix is available
		leaseLockerNSN := types.NamespacedName{
			Name:      convertCIDRToLeaseLockName(o.Spec.ParentPrefix),
			Namespace: r.OperatorNamespace,
		}
		ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.Namespace+"/"+ipAddressName)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create lease locker: %w", err)
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// 3. try to lock lease for parent prefix
		locked := ll.TryLock(lockCtx)
		if !locked {
			// lock for parent prefix was not available, rescheduling
			errorMsg := fmt.Sprintf("failed to lock parent prefix %s", o.Spec.ParentPrefix)
			r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", o.Spec.ParentPrefix))

		// 4. try to reclaim ip address
		h := generateIpAddressRestorationHash(o)
		ipAddressModel, err := r.NetboxClient.RestoreExistingIpByHash(h)
		if err != nil {
			return ctrl.Result{Requeue: true}, NewStatusError("failed to restore existing IP by hash: %w", err)
		}

		if ipAddressModel == nil {
			// ip address cannot be restored from netbox
			// 5.a assign new available ip address
			ipAddressModel, err = r.NetboxClient.GetAvailableIpAddressByClaim(
				&models.IPAddressClaim{
					ParentPrefix: o.Spec.ParentPrefix,
					Metadata: &models.NetboxMetadata{
						Tenant: o.Spec.Tenant,
					},
				})
			if err != nil {
				return ctrl.Result{Requeue: true}, NewStatusError("failed to get available IP address from NetBox: %w", err)
			}
			debugLogger.Info(fmt.Sprintf("ip address is not reserved in netbox, assigned new ip address: %s", ipAddressModel.IpAddress))
		} else {
			// 5.b reassign reserved ip address from netbox
			// do nothing, ip address restored
			debugLogger.Info(fmt.Sprintf("reassign reserved ip address from netbox, ip: %s", ipAddressModel.IpAddress))
		}

		// 6.a create the IPAddress object
		ipAddressResource := generateIpAddressFromIpAddressClaim(o, ipAddressModel.IpAddress, logger)
		if err := controllerutil.SetControllerReference(o, ipAddressResource, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set controller reference: %w", err)
		}

		if err := r.Client.Create(ctx, ipAddressResource); err != nil {
			return ctrl.Result{}, NewStatusError("failed to create IpAddress: %w", err)
		}

		debugLogger.Info("successfully created IpAddress resource")

	} else {
		// 6.b update fields of IPAddress object
		debugLogger.Info("update ipaddress resource")
		if err := r.Client.Get(ctx, ipAddressLookupKey, ipAddress); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get IpAddress for update: %w", err)
		}

		updatedIpAddressSpec := generateIpAddressSpec(o, ipAddress.Spec.IpAddress, logger)
		_, err := ctrl.CreateOrUpdate(ctx, r.Client, ipAddress, func() error {
			// only add the mutable fields here
			ipAddress.Spec.CustomFields = updatedIpAddressSpec.CustomFields
			ipAddress.Spec.Comments = updatedIpAddressSpec.Comments
			ipAddress.Spec.Description = updatedIpAddressSpec.Description
			ipAddress.Spec.PreserveInNetbox = updatedIpAddressSpec.PreserveInNetbox
			if err := controllerutil.SetControllerReference(o, ipAddress, r.Scheme); err != nil {
				return fmt.Errorf("failed to set controller reference: %w", err)
			}
			return nil
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update IpAddress: %w", err)
		}
	}

	logger.Info("reconcile loop finished")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpAddressClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpAddressClaim{}).
		Owns(&netboxv1.IpAddress{}).
		Complete(r)
}

// Status updates the IpAddressClaim status based on the current state of the owned IpAddress.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
// It captures any reconcile errors to include them in the status condition message.
func (r *IpAddressClaimReconciler) updateStatus(ctx context.Context, claim *netboxv1.IpAddressClaim, lookupKey types.NamespacedName, reconcileErr error, debugLogger logr.Logger) error {
	debugLogger.Info("updating ipaddressclaim status")

	// Initialize status conditions if this is a new resource
	if apismeta.FindStatusCondition(claim.Status.Conditions, netboxv1.ConditionReadyFalseNewResource.Type) == nil {
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionReadyFalseNewResource, corev1.EventTypeNormal, nil)
	}

	// Fetch the latest IpAddress object
	ipAddress := &netboxv1.IpAddress{}
	if err := r.Client.Get(ctx, lookupKey, ipAddress); err != nil {
		if apierrors.IsNotFound(err) {
			// IpAddress doesn't exist yet
			if reconcileErr != nil {
				// If we have an error and no IpAddress, it means IP assignment failed
				debugLogger.Info("ipaddress not found with reconcile error, reporting IpAssigned false")
				r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpAssignedFalse, corev1.EventTypeWarning, reconcileErr)
			} else {
				// No error and no IpAddress - this shouldn't happen in normal flow, skip update
				debugLogger.Info("ipaddress not found without error, skipping status update")
			}
			return nil
		}
		return fmt.Errorf("failed to get IpAddress for status update: %w", err)
	}

	// IpAddress exists - report successful IP assignment if not already reported
	if apismeta.FindStatusCondition(claim.Status.Conditions, netboxv1.ConditionIpAssignedTrue.Type) == nil {
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpAssignedTrue, corev1.EventTypeNormal, nil)
	}
	// Update status based on IpAddress readiness
	if apismeta.IsStatusConditionTrue(ipAddress.Status.Conditions, "Ready") {
		debugLogger.Info("ipaddress status ready true")
		claim.Status.IpAddress = ipAddress.Spec.IpAddress
		claim.Status.IpAddressDotDecimal = strings.Split(ipAddress.Spec.IpAddress, "/")[0]
		claim.Status.IpAddressName = ipAddress.Name
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpClaimReadyTrue, corev1.EventTypeNormal, nil)
	} else {
		debugLogger.Info("ipaddress status ready false")
		// Pass any reconcile error to the status condition
		// StatusErrors are user-facing, regular errors indicate transient/system issues
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionIpClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
	}

	err := r.Status().Update(ctx, claim)
	if err != nil {
		return err
	}

	return nil
}
