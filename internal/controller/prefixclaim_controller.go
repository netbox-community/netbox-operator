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

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/swisscom/leaselocker"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
)

// PrefixClaimReconciler reconciles a PrefixClaim object
type PrefixClaimReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	NetboxClient      *api.NetboxClient
	Recorder          record.EventRecorder
	OperatorNamespace string
	RestConfig        *rest.Config
}

// +kubebuilder:rbac:groups=netbox.dev,resources=prefixclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=netbox.dev,resources=prefixclaims/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=netbox.dev,resources=prefixclaims/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PrefixClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("prefixClaim reconcile loop started")

	/* 0. check if the matching PrefixClaim object exists */
	prefixClaim := &netboxv1.PrefixClaim{}
	if err := r.Client.Get(ctx, req.NamespacedName, prefixClaim); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !prefixClaim.ObjectMeta.DeletionTimestamp.IsZero() {
		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	/* 1. check if the matching Prefix object exists */
	prefix := &netboxv1.Prefix{}
	prefixName := prefixClaim.ObjectMeta.Name
	prefixLookupKey := types.NamespacedName{
		Name:      prefixName,
		Namespace: prefixClaim.Namespace,
	}
	if err := r.Client.Get(ctx, prefixLookupKey, prefix); err != nil { // if not nil (likely the Prefix object is not found)
		/* return error if not a notfound error */
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		debugLogger.Info("the prefix was not found, will create a new prefix object now")

		/* 2. check if the lease for parent prefix is available */
		parentPrefixName := strings.ReplaceAll(prefixClaim.Spec.ParentPrefix, "/", "-")
		leaseLockerNSN := types.NamespacedName{Name: parentPrefixName, Namespace: r.OperatorNamespace}
		ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.Namespace+"/"+prefixName)
		if err != nil {
			return ctrl.Result{}, err
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		/* 3. try to lock the lease for the parent prefix */
		locked := ll.TryLock(lockCtx)
		if !locked {
			// lock for parent prefix was not available, rescheduling
			logger.Info(fmt.Sprintf("failed to lock parent prefix %s", parentPrefixName))
			r.Recorder.Eventf(prefixClaim, corev1.EventTypeWarning, "FailedToLockParentPrefix", "failed to lock parent prefix %s", parentPrefixName)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", parentPrefixName))

		// 4. try to reclaim Prefix using restorationHash
		h := generatePrefixRestorationHash(prefixClaim)
		prefixModel, err := r.NetboxClient.RestoreExistingPrefixByHash(h)
		if err != nil {
			return ctrl.Result{}, err
		}
		// TODO: set condition for each error

		if prefixModel == nil {
			// Prefix cannot be restored from netbox
			// 5.a assign new available Prefix

			// get available Prefix under parent prefix in netbox with equal mask length
			prefixModel, err = r.NetboxClient.GetAvailablePrefixByClaim(
				&models.PrefixClaim{
					ParentPrefix: prefixClaim.Spec.ParentPrefix,
					PrefixLength: prefixClaim.Spec.PrefixLength,
					Metadata: &models.NetboxMetadata{
						Tenant: prefixClaim.Spec.Tenant,
					},
				})
			if err != nil {
				setConditionErr := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, err.Error())
				if setConditionErr != nil {
					return ctrl.Result{}, fmt.Errorf("error updating status: %w, when getting an available Prefix failed: %w", setConditionErr, err)
				}

				return ctrl.Result{Requeue: true}, nil
			}
			debugLogger.Info(fmt.Sprintf("prefix is not reserved in netbox, assignined new prefix: %s", prefixModel.Prefix))
		} else {
			// 5.b reassign reserved Prefix from netbox

			// do nothing, Prefix restored
			debugLogger.Info(fmt.Sprintf("reassign reserved prefix from netbox, prefix: %s", prefixModel.Prefix))
		}

		/* 6-1, create the Prefix object */
		prefixResource := generatePrefixFromPrefixClaim(prefixClaim, prefixModel.Prefix, logger)
		err = controllerutil.SetControllerReference(prefixClaim, prefixResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.Client.Create(ctx, prefixResource)
		if err != nil {
			if setConditionErr := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, ""); setConditionErr != nil {
				return ctrl.Result{}, fmt.Errorf("error updating status: %w, when creation of prefix object failed: %w", setConditionErr, err)
			}
			return ctrl.Result{}, err
		}

		if err = r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixAssignedTrue, corev1.EventTypeNormal, ""); err != nil {
			return ctrl.Result{}, err
		}
	} else { // Prefix object exists
		/* 6-2. update fields of the Prefix object */
		debugLogger.Info("update prefix resource")
		if err := r.Client.Get(ctx, prefixLookupKey, prefix); err != nil {
			return ctrl.Result{}, err
		}

		updatedPrefixSpec := generatePrefixSpec(prefixClaim, prefix.Spec.Prefix, logger)
		if _, err = ctrl.CreateOrUpdate(ctx, r.Client, prefix, func() error {
			// only add the mutable fields here
			prefix.Spec.Site = updatedPrefixSpec.Site
			prefix.Spec.CustomFields = updatedPrefixSpec.CustomFields
			prefix.Spec.Description = updatedPrefixSpec.Description
			prefix.Spec.Comments = updatedPrefixSpec.Comments
			prefix.Spec.PreserveInNetbox = updatedPrefixSpec.PreserveInNetbox
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	debugLogger.Info("update prefixClaim status")
	if apismeta.IsStatusConditionTrue(prefix.Status.Conditions, "Ready") {
		debugLogger.Info("prefix status ready true")

		prefixClaim.Status.Prefix = prefix.Spec.Prefix
		prefixClaim.Status.PrefixName = prefix.Name

		if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixClaimReadyTrue, corev1.EventTypeNormal, ""); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		debugLogger.Info("prefix status ready false")

		if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixClaimReadyFalse, corev1.EventTypeWarning, ""); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("prefixClaim reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrefixClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.PrefixClaim{}).
		Owns(&netboxv1.Prefix{}).
		Complete(r)
}

func (r *PrefixClaimReconciler) GetAvailablePrefix(o *netboxv1.PrefixClaim) (*models.Prefix, error) {
	var availablePrefix *models.Prefix
	var err error
	if availablePrefix, err = r.NetboxClient.GetAvailablePrefixByClaim(
		&models.PrefixClaim{
			ParentPrefix: o.Spec.ParentPrefix,
			PrefixLength: o.Spec.PrefixLength,
			Metadata: &models.NetboxMetadata{
				Tenant: o.Spec.Tenant,
			},
		},
	); err != nil {
		return nil, err
	}

	if _, err = r.NetboxClient.ReserveOrUpdatePrefix(
		&models.Prefix{
			Prefix: availablePrefix.Prefix,
			Metadata: &models.NetboxMetadata{
				Comments:    o.Spec.Comments,
				Custom:      map[string]string{},
				Description: o.Spec.Description,
				Site:        o.Spec.Site,
				Tenant:      o.Spec.Tenant,
			},
		}); err != nil {
		return nil, err
	}

	return availablePrefix, nil
}

// TODO(henrybear327): Duplicated code, consider refactoring this
func (r *PrefixClaimReconciler) SetConditionAndCreateEvent(ctx context.Context, o *netboxv1.PrefixClaim, condition metav1.Condition, eventType string, conditionMessageAppend string) error {
	if len(conditionMessageAppend) > 0 {
		condition.Message = condition.Message + ". " + conditionMessageAppend
	}
	conditionChanged := apismeta.SetStatusCondition(&o.Status.Conditions, condition)
	if conditionChanged {
		r.Recorder.Event(o, eventType, condition.Reason, condition.Message)
	}
	err := r.Client.Status().Update(ctx, o)
	if err != nil {
		return err
	}
	return nil
}
