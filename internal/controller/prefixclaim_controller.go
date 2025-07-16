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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
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
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=prefixclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=prefixclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=prefixclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PrefixClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	/* 0. check if the matching PrefixClaim object exists */
	o := &netboxv1.PrefixClaim{}
	if err := r.Client.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// Set ready to false initially
	if apismeta.FindStatusCondition(o.Status.Conditions, netboxv1.ConditionReadyFalseNewResource.Type) == nil {
		err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionReadyFalseNewResource, corev1.EventTypeNormal, nil)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to initialise Ready condition: %w, ", err)
		}
	}

	/* 1. compute and assign the parent prefix if required */
	// The current design will use prefixClaim.Status.ParentPrefix for storing the selected parent prefix,
	// and as the source of truth for future parent prefix references
	if o.Status.SelectedParentPrefix == "" /* parent prefix not yet selected/assigned */ {
		if o.Spec.ParentPrefix != "" {
			o.Status.SelectedParentPrefix = o.Spec.ParentPrefix

			// set status, and condition field
			msg := fmt.Sprintf("parentPrefix is provided in CR: %v", o.Status.SelectedParentPrefix)
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionParentPrefixSelectedTrue, corev1.EventTypeNormal, nil, msg); errReport != nil {
				return ctrl.Result{}, errReport
			}
		} else if len(o.Spec.ParentPrefixSelector) > 0 {
			// we first check if a prefix can be restored from the netbox

			// since the parent prefix is not part of the restoration hash computation
			// we can quickly check to see if the prefix with the restoration hash is matched in NetBox
			h := generatePrefixRestorationHash(o)
			canBeRestored, err := r.NetboxClient.RestoreExistingPrefixByHash(h, o.GetPrefixLengthAsInt())
			if err != nil {
				if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionParentPrefixSelectedFalse, corev1.EventTypeWarning, fmt.Errorf("failed to look up prefix by hash: %w", err)); errReport != nil {
					return ctrl.Result{}, errReport
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
				o.Status.SelectedParentPrefix = msgCanNotInferParentPrefix

				if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionParentPrefixSelectedTrue, corev1.EventTypeNormal, nil, msgCanNotInferParentPrefix); errReport != nil {
					return ctrl.Result{}, errReport
				}
			} else {
				// No, so we need to select one parent prefix from prefix candidates

				// The main idea is that we select one of the available parent prefixes as the ParentPrefix for all subsequent computation
				// The existing algorithm for prefix allocation within a ParentPrefix remains unchanged

				// fetch available prefixes from netbox
				parentPrefixCandidates, err := r.NetboxClient.GetAvailablePrefixesByParentPrefixSelector(&o.Spec)
				if err != nil {
					r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, netboxv1.ConditionPrefixAssignedFalse.Reason, netboxv1.ConditionPrefixAssignedFalse.Message+": "+err.Error())
					if err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, err); err != nil {
						return ctrl.Result{}, err
					}
					return ctrl.Result{Requeue: true}, nil
				}
				if len(parentPrefixCandidates) == 0 {
					message := "no parent prefix found matching the parentPrefixSelector"
					r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, netboxv1.ConditionPrefixAssignedFalse.Reason, netboxv1.ConditionPrefixAssignedFalse.Message+": "+message)
					if err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, errors.New(message)); err != nil {
						return ctrl.Result{}, err
					}
					// we requeue as this might be a temporary prefix exhausation
					return ctrl.Result{Requeue: true}, nil
				}

				// TODO(henrybear327): use best-fit algorithm to pick a parent prefix
				parentPrefixCandidate := parentPrefixCandidates[0]
				o.Status.SelectedParentPrefix = parentPrefixCandidate.Prefix

				// set status, and condition field
				msg := fmt.Sprintf("parentPrefix is selected: %v", o.Status.SelectedParentPrefix)
				if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionParentPrefixSelectedTrue, corev1.EventTypeNormal, nil, msg); errReport != nil {
					return ctrl.Result{}, errReport
				}
			}
		} else {
			// this case should not be triggered anymore, as we have validation rules put in place on the CR
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionParentPrefixSelectedFalse, corev1.EventTypeWarning, fmt.Errorf("%s", "either ParentPrefixSelector or ParentPrefix needs to be set")); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{}, nil
		}
	}

	/* 2. check if the matching Prefix object exists */
	prefix := &netboxv1.Prefix{}
	prefixName := o.ObjectMeta.Name
	prefixLookupKey := types.NamespacedName{
		Name:      prefixName,
		Namespace: o.Namespace,
	}
	if err := r.Client.Get(ctx, prefixLookupKey, prefix); err != nil { // if not nil (likely the Prefix object is not found)
		/* return error if not a notfound error */
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		debugLogger.Info("the prefix was not found, will create a new prefix object now")

		if o.Status.SelectedParentPrefix != msgCanNotInferParentPrefix {
			// we can't restore from the restoration hash

			/* 3. check if the lease for parent prefix is available */
			leaseLockerNSN := types.NamespacedName{
				Name:      convertCIDRToLeaseLockName(o.Status.SelectedParentPrefix),
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
				errorMsg := fmt.Sprintf("failed to lock parent prefix %s", o.Status.SelectedParentPrefix)
				r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
				return ctrl.Result{
					RequeueAfter: 2 * time.Second,
				}, nil
			}
			debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", o.Status.SelectedParentPrefix))
		} // else {
		// we can restore from the restoration hash
		// we skip directly to try to reclaim Prefix using restorationHash
		// }

		// 5. try to reclaim Prefix using restorationHash
		h := generatePrefixRestorationHash(o)
		prefixModel, err := r.NetboxClient.RestoreExistingPrefixByHash(h, o.GetPrefixLengthAsInt())
		if err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{Requeue: true}, nil
		}

		if prefixModel == nil {
			// Prefix cannot be restored from netbox
			// 6.a assign new available Prefix

			// get available Prefix under parent prefix in netbox with equal mask length
			prefixModel, err = r.NetboxClient.GetAvailablePrefixByClaim(
				&models.PrefixClaim{
					ParentPrefix: o.Status.SelectedParentPrefix,
					PrefixLength: o.Spec.PrefixLength,
					Metadata: &models.NetboxMetadata{
						Tenant: o.Spec.Tenant,
						Site:   o.Spec.Site,
					},
				})
			if err != nil {
				if errors.Is(err, api.ErrParentPrefixExhausted) {
					if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, fmt.Errorf("%w, will restart the parent prefix selection process", err)); errReport != nil {
						return ctrl.Result{}, errReport
					}

					// we reset the selected parent prefix, since this one is already exhausted
					o.Status.SelectedParentPrefix = ""

					return ctrl.Result{Requeue: true}, nil
				}

				if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
					return ctrl.Result{}, errReport
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
		prefixResource := generatePrefixFromPrefixClaim(o, prefixModel.Prefix, logger)
		err = controllerutil.SetControllerReference(o, prefixResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.Client.Create(ctx, prefixResource)
		if err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{}, err
		}

		if err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixAssignedTrue, corev1.EventTypeNormal, nil); err != nil {
			return ctrl.Result{}, err
		}
	} else { // Prefix object exists
		/* 7.b update fields of the Prefix object */
		debugLogger.Info("update prefix resource")
		if err := r.Client.Get(ctx, prefixLookupKey, prefix); err != nil {
			return ctrl.Result{}, err
		}

		updatedPrefixSpec := generatePrefixSpec(o, prefix.Spec.Prefix, logger)
		if _, err = ctrl.CreateOrUpdate(ctx, r.Client, prefix, func() error {
			// only add the mutable fields here
			prefix.Spec.Site = updatedPrefixSpec.Site
			prefix.Spec.CustomFields = updatedPrefixSpec.CustomFields
			prefix.Spec.Description = updatedPrefixSpec.Description
			prefix.Spec.Comments = updatedPrefixSpec.Comments
			prefix.Spec.PreserveInNetbox = updatedPrefixSpec.PreserveInNetbox
			err = controllerutil.SetControllerReference(o, prefix, r.Scheme)
			if err != nil {
				return err
			}
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	debugLogger.Info("update prefixClaim status")
	if apismeta.IsStatusConditionTrue(prefix.Status.Conditions, "Ready") {
		debugLogger.Info("prefix status ready true")

		o.Status.Prefix = prefix.Spec.Prefix
		o.Status.PrefixName = prefix.Name

		if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixClaimReadyTrue, corev1.EventTypeNormal, nil); errReport != nil {
			return ctrl.Result{}, errReport
		}
	} else {
		debugLogger.Info("prefix status ready false")

		if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixClaimReadyFalse, corev1.EventTypeWarning, nil); errReport != nil {
			return ctrl.Result{}, errReport
		}
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrefixClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.PrefixClaim{}).
		Owns(&netboxv1.Prefix{}).
		Complete(r)
}
