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

const AsnFinalizerName = "asn.netbox.dev/finalizer"
const AsnManagedCustomFieldsAnnotationName = "asn.netbox.dev/managed-custom-fields"

// AsnReconciler reconciles a Asn object
type AsnReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=asns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=asns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=asns/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AsnReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.Asn{}

	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch
	statusBase := o.DeepCopy()

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, reconcileResult, reconcileErr)
		if reconcileErr == nil && reconcileResult.IsZero() {
			reconcileResult, reconcileErr = scheduler.CalculateNextReconcile(ctx)
		}
		logger.Info("reconcile loop finished")
	}()

	var cancelLock context.CancelFunc

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(o, AsnFinalizerName) {
			return ctrl.Result{}, nil
		}

		if !o.Spec.PreserveInNetbox && o.Status.AsnId != 0 {
			if err = r.NetboxClient.DeleteAsn(ctx, o.Status.AsnId); err != nil {
				return ctrl.Result{}, NewDomainError("failed to delete ASN from netbox: %w", err)
			}
		}

		logger.V(4).Info("removing the finalizer")
		removed := controllerutil.RemoveFinalizer(o, AsnFinalizerName)
		if !removed {
			return ctrl.Result{}, errors.New("failed to remove the finalizer")
		}

		if err = r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// if PreserveInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(o, AsnFinalizerName) {
		logger.V(4).Info("adding the finalizer")
		controllerutil.AddFinalizer(o, AsnFinalizerName)
		if err = r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. try to lock lease of parent ASN range if AsnUrl is not set in status
	// and Asn is owned by an AsnClaim
	or := o.OwnerReferences
	var ll *leaselocker.LeaseLocker
	if len(or) > 0 && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		// get ASN claim
		orLookupKey := types.NamespacedName{
			Name:      or[0].Name,
			Namespace: req.Namespace,
		}
		asnClaim := &netboxv1.AsnClaim{}
		err = r.Get(ctx, orLookupKey, asnClaim)
		if err != nil {
			return ctrl.Result{}, err
		}

		leaseLockerNSN := types.NamespacedName{
			Name:      convertAsnRangeToLeaseLockName(asnClaim.Spec.ParentAsnRange),
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
		locked := ll.TryLock(lockCtx)
		if !locked {
			errorMsg := fmt.Sprintf("failed to lock parent ASN range %s", asnClaim.Spec.ParentAsnRange)
			r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "FailedToLockParentAsnRange", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, NewDomainError("%s", errorMsg)
		}
		logger.V(4).Info("successfully locked parent ASN range", "asnRange", asnClaim.Spec.ParentAsnRange)
	}

	// 2. reserve or update ASN in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	asnModel, err := generateNetboxAsnModelFromAsnSpec(&o.Spec, req, annotations[AsnManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxAsnModel, statusUpToDate, err := r.NetboxClient.ReserveOrUpdateAsn(ctx, asnModel, o)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.AsnId == 0 {
			logger.Info("restoration hash mismatch, deleting ASN custom resource", "asn", o.Spec.Asn)
			if deleteErr := r.Delete(ctx, o); deleteErr != nil {
				return ctrl.Result{}, NewDomainError("failed to delete Asn CR with restoration hash mismatch: %w", deleteErr)
			}
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, NewDomainError("%w", err)
	}

	// 3. unlock lease of parent ASN range
	if ll != nil {
		cancelLock()
		ll.UnlockWithRetry(ctx)
	}

	// 4. if no change in spec generation and NetBox object, skip K8s status update
	if statusUpToDate {
		return ctrl.Result{}, nil
	}

	// 4.1 update annotations
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[AsnManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		return ctrl.Result{}, NewDomainError("failed to generate managed custom fields annotation: %w", err)
	}

	// snapshot before annotation mutation for merge-patch
	patch := client.MergeFrom(o.DeepCopy())

	if err = accessor.SetAnnotations(o, annotations); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Patch(ctx, o, patch); err != nil {
		return ctrl.Result{}, err
	}

	// 4. update status fields
	o.Status.AsnId = int64(netboxAsnModel.Id)
	o.Status.AsnUrl = config.GetBaseUrl() + "/ipam/asns/" + strconv.FormatInt(int64(netboxAsnModel.Id), 10)
	if netboxAsnModel.LastUpdated.Get() != nil {
		o.Status.LastUpdated = metav1.NewTime(*netboxAsnModel.LastUpdated.Get())
	}

	// check if created ASN contains entire description from spec
	if netboxAsnModel.Description != nil {
		_, found := strings.CutPrefix(*netboxAsnModel.Description, req.String()+" // "+o.Spec.Description)
		if !found {
			r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "AsnDescriptionTruncated", "ASN was created with truncated description")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AsnReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.Asn{}).
		Complete(r)
}

// updateStatus updates the Asn status conditions based on the current state of the object.
func (r *AsnReconciler) updateStatus(ctx context.Context, o *netboxv1.Asn, statusBase *netboxv1.Asn, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	if apierrors.IsConflict(err) {
		return IgnoreDomainError(result, err)
	}

	logger.V(4).Info("updating asn status")

	switch {
	case !o.DeletionTimestamp.IsZero() && reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionAsnReadyFalseDeletionFailed, corev1.EventTypeWarning, reconcileErr)
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionAsnReadyFalseDeletionInProgress, corev1.EventTypeNormal, nil)
	case o.Status.AsnUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionAsnReadyFalse, corev1.EventTypeWarning, reconcileErr)
	case reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionAsnReadyFalse, corev1.EventTypeWarning, reconcileErr)
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionAsnReadyTrue, corev1.EventTypeNormal, nil)
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

func generateNetboxAsnModelFromAsnSpec(spec *netboxv1.AsnSpec, req ctrl.Request, lastAsnMetadata string) (*models.ASN, error) {
	// unmarshal lastAsnMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastAsnMetadata != "" {
		if err := json.Unmarshal([]byte(lastAsnMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastAsnMetadata annotation: %w", err)
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

	return &models.ASN{
		Asn: spec.Asn,
		Metadata: &models.NetboxMetadata{
			Comments:    spec.Comments,
			Custom:      netboxCustomFields,
			Description: req.String() + " // " + spec.Description,
			Tenant:      spec.Tenant,
		},
	}, nil
}
