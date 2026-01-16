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
	"strconv"

	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"

	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const VlanFinalizerName = "vlan.netbox.dev/finalizer"
const ManagedByNetboxOperatorValue = "netbox-operator"

// VlanReconciler reconciles a Vlan object
type VlanReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
}

//+kubebuilder:rbac:groups=netbox.dev,resources=vlans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=vlans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=vlans/finalizers,verbs=update

func (r *VlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	o := &netboxv1.Vlan{}
	if err := r.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle Finalizer
	if o.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(o, VlanFinalizerName) {
			controllerutil.AddFinalizer(o, VlanFinalizerName)
			if err := r.Update(ctx, o); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(o, VlanFinalizerName) {
			if !o.Spec.PreserveInNetbox && o.Status.VlanId != 0 {
				if err := r.NetboxClient.DeleteVlan(o.Status.VlanId); err != nil {
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(o, VlanFinalizerName)
			if err := r.Update(ctx, o); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// NetBox Operation
	vlanModel := &models.Vlan{
		VlanId: o.Spec.VlanId,
		Name:   o.Spec.Name,
		Metadata: &models.NetboxMetadata{
			Site:        o.Spec.Site,
			Description: o.Spec.Description,
			Comments:    o.Spec.Comments,
			Custom:      o.Spec.CustomFields,
		},
	}
	if vlanModel.Metadata.Custom == nil {
		vlanModel.Metadata.Custom = make(map[string]string)
	}
	vlanModel.Metadata.Custom["managed_by"] = ManagedByNetboxOperatorValue

	existingVlans, err := r.NetboxClient.GetVlan(vlanModel)
	if err != nil {
		return ctrl.Result{}, err
	}

	var nbVlan *netboxModels.VLAN
	vid := int64(o.Spec.VlanId)
	writableVlan := &netboxModels.WritableVLAN{
		Vid:         &vid,
		Name:        &o.Spec.Name,
		Description: o.Spec.Description,
		Comments:    o.Spec.Comments,
	}

	// Handle Site
	site, err := r.NetboxClient.GetSiteDetails(o.Spec.Site)
	if err != nil {
		return ctrl.Result{}, err
	}
	writableVlan.Site = &site.Id

	// Handle VlanGroup
	if o.Spec.VlanGroup != "" {
		vlanGroup, err := r.NetboxClient.GetVlanGroupDetails(o.Spec.VlanGroup)
		if err != nil {
			return ctrl.Result{}, err
		}
		writableVlan.Group = &vlanGroup.Id
	}

	// Handle Custom Fields
	cf := make(map[string]interface{})
	for k, v := range vlanModel.Metadata.Custom {
		cf[k] = v
	}
	writableVlan.CustomFields = cf

	if len(existingVlans.Payload.Results) == 0 {
		nbVlan, err = r.NetboxClient.CreateVlan(writableVlan)
	} else {
		existing := existingVlans.Payload.Results[0]
		// Check ownership or update if managed
		managedBy, ok := existing.CustomFields.(map[string]interface{})["managed_by"]
		if !ok || managedBy != ManagedByNetboxOperatorValue {
			logger.Info("taking ownership of unmanaged VLAN", "vlan", o.Spec.VlanId)
		}
		nbVlan, err = r.NetboxClient.UpdateVlan(existing.ID, writableVlan)
	}

	if err != nil {
		r.updateStatus(ctx, o, false, "NetBoxError", err.Error())
		return ctrl.Result{}, err
	}

	// Update Status
	o.Status.VlanId = nbVlan.ID
	o.Status.VlanUrl = config.GetBaseUrl() + "/ipam/vlans/" + strconv.FormatInt(nbVlan.ID, 10)

	apismeta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "VLAN successfully synchronized to NetBox",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VlanReconciler) updateStatus(ctx context.Context, o *netboxv1.Vlan, ready bool, reason, message string) {
	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}
	apismeta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
	_ = r.Status().Update(ctx, o)
}

func (r *VlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.Vlan{}).
		Complete(r)
}
