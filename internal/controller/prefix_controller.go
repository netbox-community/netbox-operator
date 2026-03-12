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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
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
	"github.com/swisscom/leaselocker"
)

const PrefixFinalizerName = "prefix.netbox.dev/finalizer"
const PXManagedCustomFieldsAnnotationName = "prefix.netbox.dev/managed-custom-fields"

// PrefixReconciler reconciles a Prefix object
type PrefixReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=prefixes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=prefixes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=prefixes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PrefixReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	/* 0. check if the matching Prefix object exists */
	o := &netboxv1.Prefix{}
	if err := r.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change (PrefixId, conditions, etc.).
	statusBase := o.DeepCopy()

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(o, PrefixFinalizerName) {
			if !o.Spec.PreserveInNetbox {
				if o.Status.PrefixId > math.MaxInt32 {
					return ctrl.Result{}, fmt.Errorf("reconciliation of prefixes with id's larger than 2147483647 is not supported")
				}
				if err := r.NetboxClient.DeletePrefix(ctx, int32(o.Status.PrefixId)); err != nil {
					r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixReadyFalseDeletionFailed, corev1.EventTypeWarning, err)
					return ctrl.Result{Requeue: true}, nil
				}
			}

			logger.V(4).Info("removing the finalizer")
			if removed := controllerutil.RemoveFinalizer(o, PrefixFinalizerName); !removed {
				return ctrl.Result{}, errors.New("failed to remove the finalizer")
			}

			if err := r.Update(ctx, o); err != nil {
				return ctrl.Result{}, err
			}
		}

		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, reconcileResult, reconcileErr)
	}()

	// register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(o, PrefixFinalizerName) {
		logger.V(4).Info("adding the finalizer")
		controllerutil.AddFinalizer(o, PrefixFinalizerName)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	/*
		1. try to lock the lease of the parent prefix if all of the following conditions are met
			- the prefix is owned by at least 1 prefixClaim
			- the prefix status condition is not ready
	*/
	ownerReferences := o.OwnerReferences
	var ll *leaselocker.LeaseLocker
	var cancelLock context.CancelFunc
	var err error
	if len(ownerReferences) > 0 /* len(nil array) = 0 */ && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		// get prefixClaim
		ownerReferencesLookupKey := types.NamespacedName{
			Name:      ownerReferences[0].Name, // TODO(henrybear327): Under what condition would we have more than 1 ownerReferences? What should we do with it?
			Namespace: req.Namespace,
		}
		prefixClaim := &netboxv1.PrefixClaim{}
		if err := r.Get(ctx, ownerReferencesLookupKey, prefixClaim); err != nil {
			return ctrl.Result{}, err
		}

		if prefixClaim.Status.SelectedParentPrefix == "" {
			// the parent prefix is not selected
			return ctrl.Result{
				Requeue: true,
			}, NewDomainError("the parent prefix is not selected")
		}

		if prefixClaim.Status.SelectedParentPrefix != msgCanNotInferParentPrefix {
			// we can't restore from the restoration hash

			// get the name of the parent prefix
			leaseLockerNSN := types.NamespacedName{
				Name:      convertCIDRToLeaseLockName(prefixClaim.Status.SelectedParentPrefix),
				Namespace: r.OperatorNamespace,
			}
			ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.String())
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
				errorMsg := fmt.Sprintf("failed to lock parent prefix %s", prefixClaim.Status.SelectedParentPrefix)
				r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
				return ctrl.Result{
					RequeueAfter: 2 * time.Second,
				}, NewDomainError("%s", errorMsg)
			}
			logger.V(4).Info("successfully locked parent prefix", "prefix", prefixClaim.Status.SelectedParentPrefix)
		}
	}

	/* 2. reserve or update Prefix in netbox */
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	prefixModel, err := generateNetboxPrefixModelFromPrefixSpec(&o.Spec, req, annotations[PXManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxPrefixModel, err := r.NetboxClient.ReserveOrUpdatePrefix(ctx, prefixModel)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.PrefixId == 0 {
			logger.Info("restoration hash mismatch, deleting prefix custom resource", "prefix", o.Spec.Prefix)
			if deleteErr := r.Delete(ctx, o); deleteErr != nil {
				return ctrl.Result{Requeue: true}, NewDomainError("failed to delete prefix CR with restoration hash mismatch: %w", deleteErr)
			}
			// Object deleted - status update in deferred function will be ignored via client.IgnoreNotFound
			return ctrl.Result{}, nil
		}

		return ctrl.Result{Requeue: true}, NewDomainError("failed to reserve or update prefix in netbox: %w", err)
	}

	/* 3. unlock lease of parent prefix */
	if ll != nil {
		cancelLock()
		ll.UnlockWithRetry(ctx)
	}

	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[PXManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		return ctrl.Result{Requeue: true}, NewDomainError("failed to generate managed custom fields annotation: %w", err)
	}

	// snapshot before annotation mutation for merge-patch
	patch := client.MergeFrom(o.DeepCopy())

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// patch object to store lastPrefixMetadata annotation
	if err := r.Patch(ctx, o, patch); err != nil {
		return ctrl.Result{}, err
	}

	// update status fields (set after r.Patch to avoid being overwritten by API response)
	o.Status.PrefixId = int64(netboxPrefixModel.Id)
	o.Status.PrefixUrl = config.GetBaseUrl() + "/ipam/prefixes/" + strconv.FormatInt(int64(netboxPrefixModel.Id), 10)

	// check if the created prefix contains the entire description from spec
	if netboxPrefixModel.Description == nil {
		return ctrl.Result{Requeue: true}, NewDomainError("prefix in netbox is missing a description")
	}
	if _, found := strings.CutPrefix(*netboxPrefixModel.Description, req.String()+" // "+o.Spec.Description); !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "PrefixDescriptionTruncated", "prefix was created with truncated description")
	}

	logger.V(4).Info(fmt.Sprintf("reserved prefix in netbox, prefix: %s", o.Spec.Prefix))

	logger.Info("reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrefixReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.Prefix{}).
		Complete(r)
}

// updateStatus updates the Prefix status conditions based on the current state of the object.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *PrefixReconciler) updateStatus(ctx context.Context, o *netboxv1.Prefix, statusBase *netboxv1.Prefix, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	logger.V(4).Info("updating prefix status")

	switch {
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionPrefixReadyFalse, corev1.EventTypeNormal, reconcileErr)
	case o.Status.PrefixUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionPrefixReadyFalse, corev1.EventTypeWarning, reconcileErr)
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionPrefixReadyTrue, corev1.EventTypeNormal, nil)
	}

	if apierrors.IsConflict(err) {
		// Object was modified concurrently — skip status update, will retry on requeue
		return IgnoreDomainError(result, err)
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

func generateNetboxPrefixModelFromPrefixSpec(spec *netboxv1.PrefixSpec, req ctrl.Request, lastPrefixMetadata string) (*models.Prefix, error) {
	// unmarshal lastPrefixMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastPrefixMetadata != "" {
		if err := json.Unmarshal([]byte(lastPrefixMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastPrefixMetadata annotation: %w", err)
		}
	}

	netboxCustomFields := make(map[string]string)
	if len(spec.CustomFields) > 0 {
		netboxCustomFields = maps.Clone(spec.CustomFields)
	}

	// if a custom field was removed from the spec, add it with an empty value
	for key := range lastAppliedCustomFields {
		_, ok := netboxCustomFields[key]
		if !ok {
			netboxCustomFields[key] = ""
		}
	}

	return &models.Prefix{
		Prefix: spec.Prefix,
		Metadata: &models.NetboxMetadata{
			Comments:    spec.Comments,
			Custom:      netboxCustomFields,
			Description: req.String() + " // " + spec.Description,
			Site:        spec.Site,
			Tenant:      spec.Tenant,
		},
	}, nil
}
