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

const (
	msgCanNotInferParentPrefix = "Prefix restored from hash, cannot infer the parent prefix"
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

	/* 1. compute and assign the parent prefix if required */
	// The current design will use prefixClaim.Status.ParentPrefix for storing the selected parent prefix,
	// and as the source of truth for future parent prefix references
	if prefixClaim.Status.SelectedParentPrefix == "" /* parent prefix not yet selected/assigned */ {
		if prefixClaim.Spec.ParentPrefix != "" {
			prefixClaim.Status.SelectedParentPrefix = prefixClaim.Spec.ParentPrefix

			// set status, and condition field
			msg := fmt.Sprintf("parentPrefix is provided in CR: %v", prefixClaim.Status.SelectedParentPrefix)
			if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionParentPrefixSelectedTrue, corev1.EventTypeNormal, msg); err != nil {
				return ctrl.Result{}, err
			}
		} else if len(prefixClaim.Spec.ParentPrefixSelector) > 0 {
			// we first check if a prefix can be restored from the netbox

			// since the parent prefix is not part of the restoration hash computation
			// we can quickly check to see if the prefix with the restoration hash is matched in NetBox
			h := generatePrefixRestorationHash(prefixClaim)
			canBeRestored, err := r.NetboxClient.RestoreExistingPrefixByHash(h)
			if err != nil {
				msg := fmt.Sprintf("failed to look up prefix by hash: %v", err.Error())
				if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionParentPrefixSelectedFalse, corev1.EventTypeWarning, msg); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{Requeue: true}, nil
			}

			if canBeRestored != nil {
				// Yes, so we will claim the prefix directly
				/*
					Because the parent prefix isn't part of the restoration hash,
					we can't be 100% certain what was the original parent prefix when we performed the initial allocation when restoring.

					Consider the following case:

					1) A prefix (P1) is allocated from (P), with preserveInNetBox set to true.
					|------------------------|	Parent prefix (P)
					|-----|						Allocated prefix (P1)

					2) Prefix (P1) is deleted using the NetBox operator (but still visible in NetBox because of preserveInNetBox being true)
					|------------------------|	Parent prefix (P)
					|-----|						Allocated prefix (P1)

					3) In NetBox, another prefix (P2) is created manually
					|------------------------|	Parent prefix (P)
					|---------------| 			Manually added prefix (P2)
					|-----|						Allocated prefix (P1)

					4) Perform prefix restoration

					Now we won't know if P or P2 is the parent prefix.
					But this doesn't matter since we are certain which was the original prefix we allocated and we can recover exactly that one.
				*/

				// since we can't infer the parent prefix
				// we write a special string in the ParentPrefix status field indicating the situation
				prefixClaim.Status.SelectedParentPrefix = msgCanNotInferParentPrefix

				if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionParentPrefixSelectedTrue, corev1.EventTypeNormal, msgCanNotInferParentPrefix); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				// No, so we need to select one parent prefix from prefix candidates

				// The main idea is that we select one of the available parent prefixes as the ParentPrefix for all subsequent computation
				// The existing algorithm for prefix allocation within a ParentPrefix remains unchanged

				// fetch available prefixes from netbox
				parentPrefixCandidates, err := r.NetboxClient.GetAvailablePrefixByParentPrefixSelector(&prefixClaim.Spec)
				if err != nil || len(parentPrefixCandidates) == 0 {
					errorMsg := fmt.Sprintf("no parent prefix can be obtained with the query conditions set in ParentPrefixSelector, err = %v, number of candidates = %v", err, len(parentPrefixCandidates))
					if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionParentPrefixSelectedFalse, corev1.EventTypeWarning, errorMsg); err != nil {
						return ctrl.Result{}, err
					}

					// we requeue as this might be a temporary prefix exhausation
					return ctrl.Result{Requeue: true}, nil
				}

				// TODO(henrybear327): use best-fit algorithm to pick a parent prefix
				parentPrefixCandidate := parentPrefixCandidates[0]
				prefixClaim.Status.SelectedParentPrefix = parentPrefixCandidate.Prefix

				// set status, and condition field
				msg := fmt.Sprintf("parentPrefix is selected: %v", prefixClaim.Status.SelectedParentPrefix)
				if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionParentPrefixSelectedTrue, corev1.EventTypeNormal, msg); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			// this case should not be triggered anymore, as we have validation rules put in place on the CR
			if err := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionParentPrefixSelectedFalse, corev1.EventTypeWarning, "either ParentPrefixSelector or ParentPrefix needs to be set"); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	/* 2. check if the matching Prefix object exists */
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

		if prefixClaim.Status.SelectedParentPrefix != msgCanNotInferParentPrefix {
			// we can't restore from the restoration hash

			/* 3. check if the lease for parent prefix is available */
			leaseLockerNSN := types.NamespacedName{
				Name:      convertCIDRToLeaseLockName(prefixClaim.Status.SelectedParentPrefix),
				Namespace: r.OperatorNamespace,
			}
			ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.Namespace+"/"+prefixName)
			if err != nil {
				return ctrl.Result{}, err
			}

			lockCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			/* 4. try to lock the lease for the parent prefix */
			locked := ll.TryLock(lockCtx)
			if !locked {
				// lock for parent prefix was not available, rescheduling
				errorMsg := fmt.Sprintf("failed to lock parent prefix %s", prefixClaim.Status.SelectedParentPrefix)
				r.Recorder.Eventf(prefixClaim, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
				return ctrl.Result{
					RequeueAfter: 2 * time.Second,
				}, nil
			}
			debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", prefixClaim.Status.SelectedParentPrefix))
		} // else {
		// we can restore from the restoration hash
		// we skip directly to try to reclaim Prefix using restorationHash
		// }

		// 5. try to reclaim Prefix using restorationHash
		h := generatePrefixRestorationHash(prefixClaim)
		prefixModel, err := r.NetboxClient.RestoreExistingPrefixByHash(h)
		if err != nil {
			if setConditionErr := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, err.Error()); setConditionErr != nil {
				return ctrl.Result{}, fmt.Errorf("error updating status: %w, when look up of prefix by hash failed: %w", setConditionErr, err)
			}
			return ctrl.Result{Requeue: true}, nil
		}

		if prefixModel == nil {
			// Prefix cannot be restored from netbox
			// 6.a assign new available Prefix

			// get available Prefix under parent prefix in netbox with equal mask length
			prefixModel, err = r.NetboxClient.GetAvailablePrefixByClaim(
				&models.PrefixClaim{
					ParentPrefix: prefixClaim.Status.SelectedParentPrefix,
					PrefixLength: prefixClaim.Spec.PrefixLength,
					Metadata: &models.NetboxMetadata{
						Tenant: prefixClaim.Spec.Tenant,
						Site:   prefixClaim.Spec.Site,
					},
				})
			if err != nil {
				if errors.Is(err, api.ErrParentPrefixExhausted) {
					msg := fmt.Sprintf("%v, will restart the parent prefix selection process", err.Error())
					if setConditionErr := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, msg); setConditionErr != nil {
						return ctrl.Result{}, fmt.Errorf("error updating status: %w, when failed to get matching prefix: %w", setConditionErr, err)
					}

					// we reset the selected parent prefix, since this one is already exhausted
					prefixClaim.Status.SelectedParentPrefix = ""

					return ctrl.Result{Requeue: true}, nil
				}

				if setConditionErr := r.SetConditionAndCreateEvent(ctx, prefixClaim, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, err.Error()); setConditionErr != nil {
					return ctrl.Result{}, fmt.Errorf("error updating status: %w, when failed to get matching prefix: %w", setConditionErr, err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
			debugLogger.Info(fmt.Sprintf("prefix is not reserved in netbox, assignined new prefix: %s", prefixModel.Prefix))
		} else {
			// 6.b reassign reserved Prefix from netbox

			// do nothing, Prefix restored
			debugLogger.Info(fmt.Sprintf("reassign reserved prefix from netbox, prefix: %s", prefixModel.Prefix))
		}

		/* 7.a create the Prefix object */
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
		/* 7.b update fields of the Prefix object */
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
