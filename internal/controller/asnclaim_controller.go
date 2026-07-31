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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AsnClaimReconciler reconciles a AsnClaim object
type AsnClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=asnclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=asnclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=asnclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AsnClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	/* 0. check if the matching AsnClaim object exists */
	o := &netboxv1.AsnClaim{}
	if err := r.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch
	statusBase := o.DeepCopy()

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, req.NamespacedName, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.Info("reconcile loop finished")
	}()

	// 1. check if matching Asn object already exists
	asn := &netboxv1.Asn{}
	asnName := o.Name
	asnLookupKey := types.NamespacedName{
		Name:      asnName,
		Namespace: o.Namespace,
	}

	err := r.Get(ctx, asnLookupKey, asn)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to get Asn: %w", err)
		}

		logger.V(4).Info("asn object matching asn claim was not found, creating new asn object")

		// 2. check if lease for parent ASN range is available
		leaseLockerNSN := types.NamespacedName{
			Name:      convertAsnRangeToLeaseLockName(o.Spec.ParentAsnRange),
			Namespace: r.OperatorNamespace,
		}
		ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.Namespace+"/"+asnName)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create lease locker: %w", err)
		}

		// 3. try to lock lease for parent ASN range
		lockCtx, cancelLock := context.WithTimeout(ctx, lockAcquireTimeout)
		defer cancelLock()
		locked := ll.TryLock(lockCtx)
		if !locked {
			errorMsg := fmt.Sprintf("failed to lock parent ASN range %s", o.Spec.ParentAsnRange)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, NewDomainError("%s", errorMsg)
		}
		logger.V(4).Info("successfully locked parent ASN range", "asnRange", o.Spec.ParentAsnRange)

		// 4. try to reclaim ASN
		h := generateAsnRestorationHash(o)
		asnModel, err := r.NetboxClient.RestoreExistingAsnByHash(ctx, h)
		if err != nil {
			return ctrl.Result{}, NewDomainError("%w", err)
		}

		if asnModel == nil {
			// ASN cannot be restored from netbox
			// 5.a assign new available ASN
			// NetBox creates the ASN as part of the available-asns request, so the
			// restoration hash has to be part of that request. Otherwise a crash before
			// the Asn resource is reconciled would leave an unidentifiable ASN behind.
			asnModel, err = r.NetboxClient.GetAvailableAsnByClaim(
				ctx,
				&models.ASNClaim{
					ParentAsnRange: o.Spec.ParentAsnRange,
					Metadata: &models.NetboxMetadata{
						Tenant:      o.Spec.Tenant,
						Description: o.Spec.Description,
						Custom: map[string]string{
							config.GetOperatorConfig().NetboxRestorationHashFieldName: h,
						},
					},
				})
			if err != nil {
				return ctrl.Result{}, NewDomainError("%w", err)
			}
			logger.V(4).Info("ASN is not reserved in netbox, assigned new ASN", "asn", asnModel.Asn)
		} else {
			// 5.b reassign reserved ASN from netbox
			logger.V(4).Info("reassign reserved ASN from netbox", "asn", asnModel.Asn)
		}

		// 6.a create the Asn object
		asnResource := generateAsnFromAsnClaim(o, asnModel.Asn, logger)
		if err := controllerutil.SetControllerReference(o, asnResource, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set controller reference: %w", err)
		}

		if err := r.Create(ctx, asnResource); err != nil {
			return ctrl.Result{}, NewDomainError("failed to create Asn: %w", err)
		}

		logger.V(4).Info("successfully created Asn resource")

	} else {
		// 6.b update fields of Asn object
		logger.V(4).Info("update asn resource")
		updatedAsnSpec := generateAsnSpec(o, asn.Spec.Asn, logger)
		_, err := ctrl.CreateOrUpdate(ctx, r.Client, asn, func() error {
			// only add the mutable fields here
			asn.Spec.CustomFields = updatedAsnSpec.CustomFields
			asn.Spec.Comments = updatedAsnSpec.Comments
			asn.Spec.Description = updatedAsnSpec.Description
			asn.Spec.PreserveInNetbox = updatedAsnSpec.PreserveInNetbox
			if err := controllerutil.SetControllerReference(o, asn, r.Scheme); err != nil {
				return fmt.Errorf("failed to set controller reference: %w", err)
			}
			return nil
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update Asn: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AsnClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.AsnClaim{}).
		Owns(&netboxv1.Asn{}).
		Complete(r)
}

// updateStatus updates the AsnClaim status based on the current state of the owned Asn.
func (r *AsnClaimReconciler) updateStatus(ctx context.Context, claim *netboxv1.AsnClaim, statusBase *netboxv1.AsnClaim, lookupKey types.NamespacedName, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	// Ensure status update is always called, even on early returns
	defer func() {
		if apierrors.IsConflict(err) {
			result, err = IgnoreDomainError(result, err)
			return
		}
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

	logger.V(4).Info("updating asnclaim status")

	// Fetch the latest Asn object
	asn := &netboxv1.Asn{}
	err = r.Client.Get(ctx, lookupKey, asn)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Asn doesn't exist yet
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionAsnAssignedFalse, corev1.EventTypeWarning, reconcileErr)
			r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionAsnClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
			if result.IsZero() {
				result = ctrl.Result{RequeueAfter: 1 * time.Second}
			}
			err = nil
			return result, err
		}
		err = fmt.Errorf("failed to get Asn for status update: %w", err)
		return result, err
	}

	// Asn exists - report successful ASN assignment if not already reported
	if apismeta.FindStatusCondition(claim.Status.Conditions, netboxv1.ConditionAsnAssignedTrue.Type) == nil || apismeta.IsStatusConditionFalse(claim.Status.Conditions, netboxv1.ConditionAsnAssignedTrue.Type) {
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionAsnAssignedTrue, corev1.EventTypeNormal, nil)
	}
	// Update status based on Asn readiness
	if apismeta.IsStatusConditionTrue(asn.Status.Conditions, netboxv1.ConditionAsnReadyTrue.Type) {
		logger.V(4).Info("asn status ready true")
		claim.Status.Asn = asn.Spec.Asn
		claim.Status.AsnName = asn.Name
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionAsnClaimReadyTrue, corev1.EventTypeNormal, nil)
	} else {
		logger.V(4).Info("asn status ready false")
		r.EventStatusRecorder.Report(ctx, claim, netboxv1.ConditionAsnClaimReadyFalse, corev1.EventTypeWarning, reconcileErr)
	}

	return result, err
}
