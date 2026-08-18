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
	"math"
	"strconv"
	"strings"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/scheduler"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const VlanGroupFinalizerName = "vlangroup.netbox.dev/finalizer"
const VlanGroupManagedCustomFieldsAnnotationName = "vlangroup.netbox.dev/managed-custom-fields"

// VlanGroupReconciler reconciles a VlanGroup object
type VlanGroupReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=vlangroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=vlangroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=vlangroups/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VlanGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.VlanGroup{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change (VlanGroupId, conditions, etc.).
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
		if !controllerutil.ContainsFinalizer(o, VlanGroupFinalizerName) {
			return ctrl.Result{}, nil
		}

		if !o.Spec.PreserveInNetbox {
			if o.Status.VlanGroupId > math.MaxInt32 {
				return ctrl.Result{}, fmt.Errorf("reconciliation of vlan groups with id's larger than 2147483647 is not supported")
			}
			if err := r.NetboxClient.DeleteVlanGroup(ctx, int32(o.Status.VlanGroupId)); err != nil {
				return ctrl.Result{}, NewDomainError("failed to delete vlan group in netbox: %w", err)
			}
		}

		return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, VlanGroupFinalizerName)
	}

	// if PreserveInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox {
		err = addFinalizer(ctx, r.Client, o, VlanGroupFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// reserve or update vlan group in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	vlanGroupModel, err := r.generateNetboxVlanGroupModelFromVlanGroupSpec(o, req, annotations[VlanGroupManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxVlanGroupModel, statusUpToDate, err := r.NetboxClient.ReserveOrUpdateVlanGroup(ctx, vlanGroupModel, o)
	if err != nil {
		return ctrl.Result{}, NewDomainError("%w", err)
	}

	// if no change, then end loop
	if statusUpToDate {
		return ctrl.Result{}, nil
	}

	// update annotation
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[VlanGroupManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
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
	o.Status.VlanGroupId = int64(netboxVlanGroupModel.GetId())
	o.Status.Slug = netboxVlanGroupModel.GetSlug()
	o.Status.VlanGroupUrl = config.GetBaseUrl() + "/ipam/vlan-groups/" + strconv.FormatInt(int64(netboxVlanGroupModel.GetId()), 10)
	if netboxVlanGroupModel.LastUpdated.IsSet() {
		o.Status.LastUpdated = metav1.NewTime(*netboxVlanGroupModel.LastUpdated.Get())
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VlanGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.VlanGroup{}).
		Complete(r)
}

// updateStatus updates the VlanGroup status conditions based on the current state of the object.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
func (r *VlanGroupReconciler) updateStatus(ctx context.Context, o *netboxv1.VlanGroup, statusBase *netboxv1.VlanGroup, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	if apierrors.IsConflict(err) {
		// Object was modified concurrently — skip status update, will retry on requeue
		return IgnoreDomainError(result, err)
	}

	logger.V(4).Info("updating vlan group status")

	switch {
	case !o.DeletionTimestamp.IsZero() && reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanGroupReadyFalseDeletionFailed, corev1.EventTypeWarning, reconcileErr)
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanGroupReadyFalseDeletionInProgress, corev1.EventTypeNormal, nil)
	case o.Status.VlanGroupUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanGroupReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("name: %s", o.Spec.Name))
	case reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanGroupReadyFalse, corev1.EventTypeWarning, reconcileErr,
			fmt.Sprintf("name: %s", o.Spec.Name))
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionVlanGroupReadyTrue, corev1.EventTypeNormal, nil)
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

func (r *VlanGroupReconciler) generateNetboxVlanGroupModelFromVlanGroupSpec(o *netboxv1.VlanGroup, req ctrl.Request, lastVlanGroupMetadata string) (*models.VlanGroup, error) {
	// unmarshal lastVlanGroupMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastVlanGroupMetadata != "" {
		if err := json.Unmarshal([]byte(lastVlanGroupMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastVlanGroupMetadata annotation: %w", err)
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

	// check if created vlan group contains entire description from spec
	_, found := strings.CutPrefix(description, req.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "VlanGroupDescriptionTruncated", "vlan group was created with truncated description")
	}

	return &models.VlanGroup{
		Name:          o.Spec.Name,
		VidRangeStart: o.Spec.VidRangeStart,
		VidRangeEnd:   o.Spec.VidRangeEnd,
		Metadata: &models.NetboxMetadata{
			Custom:      netboxCustomFields,
			Description: description,
			Site:        o.Spec.Site,
			Tenant:      o.Spec.Tenant,
		},
	}, nil
}
