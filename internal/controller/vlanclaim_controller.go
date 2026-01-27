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
	"crypto/sha256"
	"fmt"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
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

// VLANClaimReconciler reconciles a VLANClaim object
type VLANClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=vlanclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=vlanclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=vlanclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=netbox.dev,resources=vlans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *VLANClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	/* 0. check if the matching VLANClaim object exists */
	o := &netboxv1.VLANClaim{}
	if err := r.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Set ready to false initially
	if apismeta.FindStatusCondition(o.Status.Conditions, "Ready") == nil {
		if err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionVlanClaimReadyFalse, corev1.EventTypeNormal, nil); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. check if matching Vlan object already exists
	vlan := &netboxv1.Vlan{}
	vlanName := o.Name
	vlanLookupKey := types.NamespacedName{
		Name:      vlanName,
		Namespace: o.Namespace,
	}

	err := r.Get(ctx, vlanLookupKey, vlan)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		debugLogger.Info("vlan object matching vlan claim was not found, creating new vlan object")

		// 2. check if lease for vlan group is available
		leaseLockerNSN := types.NamespacedName{
			Name:      convertVlanGroupToLeaseLockName(o.Spec.VlanGroup),
			Namespace: r.OperatorNamespace,
		}
		ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.Namespace+"/"+vlanName)
		if err != nil {
			return ctrl.Result{}, err
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// 3. try to lock lease for vlan group
		locked := ll.TryLock(lockCtx)
		if !locked {
			errorMsg := fmt.Sprintf("failed to lock vlan group %s", o.Spec.VlanGroup)
			r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockVlanGroup", errorMsg)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		debugLogger.Info(fmt.Sprintf("successfully locked vlan group %s", o.Spec.VlanGroup))

		// 4. try to reclaim vlan
		h := generateVlanRestorationHash(o)
		vlanModel, err := r.NetboxClient.RestoreExistingVlanByHash(h)
		if err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionVlanAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{Requeue: true}, nil
		}

		if vlanModel == nil {
			// vlan cannot be restored from netbox, assign new one
			if o.Spec.VlanId != 0 {
				vlanModel = &models.Vlan{VlanId: o.Spec.VlanId}
			} else {
				vlanModel, err = r.NetboxClient.GetAvailableVlanByClaim(&models.VLANClaim{
					VlanGroup: o.Spec.VlanGroup,
					Metadata: &models.NetboxMetadata{
						Site: o.Spec.Site,
					},
				})
				if err != nil {
					if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionVlanAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
						return ctrl.Result{}, errReport
					}
					return ctrl.Result{Requeue: true}, nil
				}
			}
			debugLogger.Info(fmt.Sprintf("assigned vlan vid: %d", vlanModel.VlanId))
		}

		// 6.a create the Vlan object
		vlanResource := generateVlanFromVlanClaim(o, vlanModel.VlanId)
		if err = controllerutil.SetControllerReference(o, vlanResource, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err = r.Create(ctx, vlanResource); err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionVlanAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{}, err
		}

		if err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionVlanAssignedTrue, corev1.EventTypeNormal, nil); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// 6.b update fields of Vlan object
		debugLogger.Info("update vlan resource")
		updatedVlanSpec := generateVlanSpec(o, vlan.Spec.VlanId)
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, vlan, func() error {
			vlan.Spec.Name = updatedVlanSpec.Name
			vlan.Spec.Site = updatedVlanSpec.Site
			vlan.Spec.VlanGroup = updatedVlanSpec.VlanGroup
			vlan.Spec.CustomFields = updatedVlanSpec.CustomFields
			vlan.Spec.Comments = updatedVlanSpec.Comments
			vlan.Spec.Description = updatedVlanSpec.Description
			vlan.Spec.PreserveInNetbox = updatedVlanSpec.PreserveInNetbox
			return controllerutil.SetControllerReference(o, vlan, r.Scheme)
		})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// 7. update VLANClaim Ready status
	if apismeta.IsStatusConditionTrue(vlan.Status.Conditions, "Ready") {
		o.Status.VlanId = vlan.Spec.VlanId
		o.Status.VlanName = vlan.Name
		if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionVlanClaimReadyTrue, corev1.EventTypeNormal, nil); errReport != nil {
			return ctrl.Result{}, errReport
		}
	} else {
		if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionVlanClaimReadyFalse, corev1.EventTypeWarning, nil); errReport != nil {
			return ctrl.Result{}, errReport
		}
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("reconcile loop finished")
	return ctrl.Result{}, nil
}

func generateVlanRestorationHash(o *netboxv1.VLANClaim) string {
	h := sha256.New()
	h.Write([]byte(o.Namespace))
	h.Write([]byte(o.Name))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func generateVlanFromVlanClaim(o *netboxv1.VLANClaim, vid int) *netboxv1.Vlan {
	return &netboxv1.Vlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.Namespace,
		},
		Spec: generateVlanSpec(o, vid),
	}
}

func generateVlanSpec(o *netboxv1.VLANClaim, vid int) netboxv1.VlanSpec {
	name := o.Spec.Name
	if name == "" {
		name = o.Name
	}
	return netboxv1.VlanSpec{
		VlanId:           vid,
		Name:             name,
		Site:             o.Spec.Site,
		VlanGroup:        o.Spec.VlanGroup,
		Description:      o.Spec.Description,
		Comments:         o.Spec.Comments,
		CustomFields:     o.Spec.CustomFields,
		PreserveInNetbox: o.Spec.PreserveInNetbox,
	}
}

func (r *VLANClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.VLANClaim{}).
		Owns(&netboxv1.Vlan{}).
		Complete(r)
}
