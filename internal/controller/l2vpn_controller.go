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

const L2VPNFinalizerName = "l2vpn.netbox.dev/finalizer"
const L2VPNManagedCustomFieldsAnnotationName = "l2vpn.netbox.dev/managed-custom-fields"

// L2VPNReconciler reconciles a L2VPN object
type L2VPNReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=l2vpns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=l2vpns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=l2vpns/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *L2VPNReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.L2VPN{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change (L2VPNId, conditions, etc.).
	statusBase := o.DeepCopy()

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.Info("reconcile loop finished")
	}()

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(o, L2VPNFinalizerName) {
			return ctrl.Result{}, nil
		}

		if !o.Spec.PreserveInNetbox {
			if err := r.NetboxClient.DeleteL2VPN(ctx, o.Status.L2VPNId); err != nil {
				return ctrl.Result{}, NewDomainError("failed to delete l2vpn in netbox: %w", err)
			}
		}

		return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, L2VPNFinalizerName)
	}

	// if PreserveInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox {
		err = addFinalizer(ctx, r.Client, o, L2VPNFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. try to lock the shared l2vpn identifier pool if L2VPN status condition
	// is not true, is owned by a L2VPNClaim, and hasn't been created in NetBox
	// yet. This serializes against the same lock the L2VPNClaim controller
	// held while assigning this L2VPN's identifier.
	or := o.OwnerReferences
	var ll *leaselocker.LeaseLocker
	var cancelLock context.CancelFunc
	if len(or) > 0 && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		leaseLockerNSN, owner, identifierDesc, err := r.getLeaseLockerNSNandOwner(ctx, o)
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
			errorMsg := fmt.Sprintf("failed to lock l2vpn identifiers for %s", identifierDesc)
			r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "FailedToLockIdentifierRange", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, NewDomainError("%s", errorMsg)
		}
		logger.V(4).Info(fmt.Sprintf("successfully locked l2vpn identifiers for %s", identifierDesc))
	}

	// 2. reserve or update l2vpn in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	l2vpnModel, err := r.generateNetboxL2VPNModelFromL2VPNSpec(o, req, annotations[L2VPNManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxL2VPNModel, statusUpToDate, err := r.NetboxClient.ReserveOrUpdateL2VPN(ctx, l2vpnModel, o)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.L2VPNId == 0 {
			logger.Info("conflict in claimed l2vpn, deleting l2vpn custom resource", "identifier",
				o.Spec.Identifier, "error", err)
			if deleteErr := r.Delete(ctx, o); deleteErr != nil {
				return ctrl.Result{}, NewDomainError("failed to delete L2VPN CR with conflict: %w", deleteErr)
			}
			// Object deleted - status update in deferred function will be ignored via client.IgnoreNotFound
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, NewDomainError("%w", err)
	}

	// 3. unlock lease of the identifier range
	if ll != nil {
		cancelLock()
		ll.UnlockWithRetry(ctx)
	}

	// 4. if no change, then end loop
	if statusUpToDate {
		return ctrl.Result{}, nil
	}

	// 4.1 update annotation
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[L2VPNManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		return ctrl.Result{}, NewDomainError("failed to generate managed custom fields annotation: %w", err)
	}

	// snapshot before annotation mutation for merge-patch
	patch := client.MergeFrom(o.DeepCopy())

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// patch object to store managed custom fields annotation
	err = r.Patch(ctx, o, patch)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update status fields (set after r.Patch to avoid being overwritten by API response)
	o.Status.L2VPNId = int64(netboxL2VPNModel.GetId())
	o.Status.Slug = netboxL2VPNModel.GetSlug()
	o.Status.L2VPNUrl = config.GetBaseUrl() + "/vpn/l2vpns/" + strconv.FormatInt(int64(netboxL2VPNModel.GetId()), 10)
	if netboxL2VPNModel.LastUpdated.IsSet() {
		o.Status.LastUpdated = metav1.NewTime(*netboxL2VPNModel.LastUpdated.Get())
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *L2VPNReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.L2VPN{}).
		Complete(r)
}

// updateStatus updates the L2VPN status conditions based on the current state of the object.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *L2VPNReconciler) updateStatus(ctx context.Context, o *netboxv1.L2VPN, statusBase *netboxv1.L2VPN, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	if apierrors.IsConflict(err) {
		// Object was modified concurrently — skip status update, will retry on requeue
		return IgnoreDomainError(result, err)
	}

	logger.V(4).Info("updating l2vpn status")

	switch {
	case !o.DeletionTimestamp.IsZero() && reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionL2VPNReadyFalseDeletionFailed, corev1.EventTypeWarning, reconcileErr)
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionL2VPNReadyFalseDeletionInProgress, corev1.EventTypeNormal, nil)
	case o.Status.L2VPNUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionL2VPNReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("identifier: %d", o.Spec.Identifier))
	case reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionL2VPNReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("identifier: %d", o.Spec.Identifier))
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionL2VPNReadyTrue, corev1.EventTypeNormal, nil)
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

func (r *L2VPNReconciler) generateNetboxL2VPNModelFromL2VPNSpec(o *netboxv1.L2VPN, req ctrl.Request, lastL2VPNMetadata string) (*models.L2VPN, error) {
	// unmarshal lastL2VPNMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastL2VPNMetadata != "" {
		if err := json.Unmarshal([]byte(lastL2VPNMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastL2VPNMetadata annotation: %w", err)
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

	// check if created l2vpn contains entire description from spec
	_, found := strings.CutPrefix(description, req.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "L2VPNDescriptionTruncated", "l2vpn was created with truncated description")
	}

	return &models.L2VPN{
		Name:       o.Name,
		Type:       o.Spec.Type,
		Identifier: o.Spec.Identifier,
		Metadata: &models.NetboxMetadata{
			Comments:    o.Spec.Comments,
			Custom:      netboxCustomFields,
			Description: description,
			Tenant:      o.Spec.Tenant,
		},
	}, nil
}

// getLeaseLockerNSNandOwner returns the lease lock identity for the L2VPN's
// owning claim, against the same shared lock (l2vpnIdentifierLockName) the
// L2VPNClaim controller used while assigning this L2VPN's identifier, so
// NetBox reservation stays serialized against concurrent claims for the
// duration of the create.
func (r *L2VPNReconciler) getLeaseLockerNSNandOwner(ctx context.Context, o *netboxv1.L2VPN) (nsn types.NamespacedName, owner string, identifierDesc string, err error) {
	if len(o.OwnerReferences) == 0 {
		return types.NamespacedName{}, "", "", fmt.Errorf("L2VPN %s/%s has no owner references", o.Namespace, o.Name)
	}

	orLookupKey := types.NamespacedName{
		Name:      o.ObjectMeta.OwnerReferences[0].Name,
		Namespace: o.Namespace,
	}

	l2vpnClaim := &netboxv1.L2VPNClaim{}
	err = r.Get(ctx, orLookupKey, l2vpnClaim)
	if err != nil {
		return types.NamespacedName{}, "", "", err
	}

	leaseLockerNSN := types.NamespacedName{
		Name:      l2vpnIdentifierLockName,
		Namespace: r.OperatorNamespace,
	}

	return leaseLockerNSN, orLookupKey.String(), describeL2VPNClaimIdentifier(l2vpnClaim), nil
}
