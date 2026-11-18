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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const RouteTargetFinalizerName = "routetarget.netbox.dev/finalizer"
const RouteTargetManagedCustomFieldsAnnotationName = "routetarget.netbox.dev/managed-custom-fields"

// RouteTargetReconciler reconciles a RouteTarget object
type RouteTargetReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=routetargets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=routetargets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=routetargets/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RouteTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("reconcile loop started")

	o := &netboxv1.RouteTarget{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Base for the deferred status patch. Re-snapshotted after the status fields are
	// persisted, so from that point on the deferred patch only carries conditions.
	statusBase := o.DeepCopy()

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.V(4).Info("reconcile loop finished")
	}()

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(o, RouteTargetFinalizerName) {
			return ctrl.Result{}, nil
		}

		if !o.Spec.PreserveInNetbox {
			if err := r.NetboxClient.DeleteRouteTarget(ctx, o.Status.RouteTargetId); err != nil {
				return ctrl.Result{}, NewDomainError("failed to delete route target in netbox: %w", err)
			}
		}

		return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, RouteTargetFinalizerName)
	}

	// the finalizer exists only to delete the Route Target from NetBox, so it has to follow PreserveInNetbox
	if o.Spec.PreserveInNetbox {
		err = removeFinalizer(ctx, r.Client, o, RouteTargetFinalizerName)
	} else {
		err = addFinalizer(ctx, r.Client, o, RouteTargetFinalizerName)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// 1. try to lock the lease of the route target ASN if RouteTarget status
	// condition is not true, is owned by a RouteTargetClaim, and hasn't been
	// created in NetBox yet. This serializes against the same lock the
	// RouteTargetClaim controller held while assigning this RouteTarget.
	or := o.OwnerReferences
	var ll *leaselocker.LeaseLocker
	var cancelLock context.CancelFunc
	if len(or) > 0 && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		leaseLockerNSN, owner, rangeDesc, err := r.getLeaseLockerNSNandOwner(ctx, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, owner)
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
			errorMsg := fmt.Sprintf("failed to lock route targets for %s", rangeDesc)
			r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "FailedToLockRouteTargetRange", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, NewDomainError("%s", errorMsg)
		}
		logger.V(4).Info(fmt.Sprintf("successfully locked route targets for %s", rangeDesc))
	}

	// 2. reserve or update route target in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	routeTargetModel, err := r.generateNetboxRouteTargetModelFromRouteTargetSpec(o, req, annotations[RouteTargetManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxRouteTargetModel, statusUpToDate, err := r.NetboxClient.ReserveOrUpdateRouteTarget(ctx, routeTargetModel, o)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.RouteTargetId == 0 && isOwnedByClaim(o, r.Scheme) {
			logger.Info("conflict in claimed route target, deleting route target custom resource", "routeTarget",
				o.Spec.RouteTarget, "error", err)
			if deleteErr := r.Delete(ctx, o); deleteErr != nil {
				return ctrl.Result{}, NewDomainError("failed to delete RouteTarget CR with conflict: %w", deleteErr)
			}
			// Object deleted - status update in deferred function will be ignored via client.IgnoreNotFound
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, NewDomainError("%w", err)
	}

	// 3. unlock the lease of the route target ASN
	if ll != nil {
		cancelLock()
		ll.UnlockWithRetry(ctx)
	}

	// 4. if no change in spec generation and NetBox object, skip K8s status update
	if statusUpToDate {
		return ctrl.Result{}, nil
	}

	// 5. update status fields
	statusFieldsBase := o.DeepCopy()
	o.Status.RouteTargetId = int64(netboxRouteTargetModel.GetId())
	o.Status.RouteTargetUrl = config.GetBaseUrl() + "/ipam/route-targets/" + strconv.FormatInt(int64(netboxRouteTargetModel.GetId()), 10)
	if netboxRouteTargetModel.LastUpdated.IsSet() {
		o.Status.LastUpdated = metav1.NewTime(*netboxRouteTargetModel.LastUpdated.Get())
	}

	// persist the status now so the annotation patch response cannot drop it
	if err := r.Status().Patch(ctx, o, client.MergeFrom(statusFieldsBase)); err != nil {
		return ctrl.Result{}, err
	}
	statusBase = o.DeepCopy()

	// 6. update annotations
	// the status patch response replaced o's annotation map, so re-read it
	annotations, err = accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	patch := client.MergeFrom(o.DeepCopy())

	annotations[RouteTargetManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		return ctrl.Result{}, NewDomainError("failed to generate managed custom fields annotation: %w", err)
	}

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// patch object to store managed custom fields annotation
	err = r.Patch(ctx, o, patch)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RouteTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.RouteTarget{}).
		Complete(r)
}

// updateStatus updates the RouteTarget status conditions based on the current state of the object.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *RouteTargetReconciler) updateStatus(ctx context.Context, o *netboxv1.RouteTarget, statusBase *netboxv1.RouteTarget, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	if apierrors.IsConflict(err) {
		// Object was modified concurrently — skip status update, will retry on requeue
		return IgnoreDomainError(result, err)
	}

	logger.V(4).Info("updating route target status")

	switch {
	case !o.DeletionTimestamp.IsZero() && reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionRouteTargetReadyFalseDeletionFailed, corev1.EventTypeWarning, reconcileErr)
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionRouteTargetReadyFalseDeletionInProgress, corev1.EventTypeNormal, nil)
	case o.Status.RouteTargetUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionRouteTargetReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("route target: %s", o.Spec.RouteTarget))
	case reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionRouteTargetReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("route target: %s", o.Spec.RouteTarget))
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionRouteTargetReadyTrue, corev1.EventTypeNormal, nil)
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

func (r *RouteTargetReconciler) generateNetboxRouteTargetModelFromRouteTargetSpec(o *netboxv1.RouteTarget, req ctrl.Request, lastRouteTargetMetadata string) (*models.RouteTarget, error) {
	// unmarshal lastRouteTargetMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastRouteTargetMetadata != "" {
		if err := json.Unmarshal([]byte(lastRouteTargetMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastRouteTargetMetadata annotation: %w", err)
		}
	}

	netboxCustomFields := make(map[string]string)
	if len(o.Spec.CustomFields) > 0 {
		netboxCustomFields = maps.Clone(o.Spec.CustomFields)
	}

	// if a custom field was removed from the spec, add it with an empty value
	for key := range lastAppliedCustomFields {
		_, ok := netboxCustomFields[key]
		if !ok {
			netboxCustomFields[key] = ""
		}
	}

	description := api.TruncateDescription(req.String() + " // " + o.Spec.Description)

	// check if created route target contains entire description from spec
	_, found := strings.CutPrefix(description, req.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "RouteTargetDescriptionTruncated", "route target was created with truncated description")
	}

	return &models.RouteTarget{
		Name: o.Spec.RouteTarget,
		Metadata: &models.NetboxMetadata{
			Comments:    o.Spec.Comments,
			Custom:      netboxCustomFields,
			Description: description,
			Tenant:      o.Spec.Tenant,
		},
	}, nil
}

// getLeaseLockerNSNandOwner returns the lease lock identity for the
// RouteTarget's owning claim, against the same per-ASN lock
// (routeTargetLockName) the RouteTargetClaim controller used while assigning
// this RouteTarget, so NetBox reservation stays serialized against
// concurrent claims for the duration of the create.
func (r *RouteTargetReconciler) getLeaseLockerNSNandOwner(ctx context.Context, o *netboxv1.RouteTarget) (nsn types.NamespacedName, owner string, rangeDesc string, err error) {
	if len(o.OwnerReferences) == 0 {
		return types.NamespacedName{}, "", "", fmt.Errorf("RouteTarget %s/%s has no owner references", o.Namespace, o.Name)
	}

	orLookupKey := types.NamespacedName{
		Name:      o.ObjectMeta.OwnerReferences[0].Name,
		Namespace: o.Namespace,
	}

	routeTargetClaim := &netboxv1.RouteTargetClaim{}
	err = r.Get(ctx, orLookupKey, routeTargetClaim)
	if err != nil {
		return types.NamespacedName{}, "", "", err
	}

	leaseLockerNSN := types.NamespacedName{
		Name:      routeTargetLockName(routeTargetClaim.Spec.Asn),
		Namespace: r.OperatorNamespace,
	}

	return leaseLockerNSN, orLookupKey.String(), describeRouteTargetClaimRange(routeTargetClaim), nil
}
