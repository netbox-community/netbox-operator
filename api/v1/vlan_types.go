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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VlanSpec defines the desired state of Vlan
type VlanSpec struct {
	// The name of the VLAN in NetBox
	// Field is immutable, required
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'name' is immutable"
	Name string `json:"name"`

	// The VLAN ID (VID) to be assigned to this VLAN in NetBox
	// Field is immutable, required, range from 1-4094
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=4094
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'vid' is immutable"
	Vid int32 `json:"vid"`

	// The NetBox Site to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	// Field is immutable, not required
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'site' is immutable"
	Site string `json:"site,omitempty"`

	// The NetBox Tenant to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	// Field is immutable, not required
	// Example: "Initech" or "Cyberdyne Systems"
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	// The operational status of the VLAN in NetBox
	// Field is mutable, not required, defaults to "active"
	//+kubebuilder:validation:Enum=active;reserved;deprecated
	Status string `json:"status,omitempty"`

	// The NetBox Custom Fields that should be added to the resource in NetBox.
	// Note that currently only Text Type is supported (GitHub #129)
	// More info on NetBox Custom Fields:
	// https://github.com/netbox-community/netbox/blob/main/docs/customization/custom-fields.md
	// Field is mutable, not required
	// Example:
	//   customfield1: "Production"
	//   customfield2: "This is a string"
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Comment that should be added to the resource in NetBox
	// Field is mutable, not required
	Comments string `json:"comments,omitempty"`

	// Description that should be added to the resource in NetBox
	// Field is mutable, not required
	Description string `json:"description,omitempty"`

	// Defines whether the Resource should be preserved in NetBox when the
	// Kubernetes Resource is deleted.
	// - When set to true, the resource will not be deleted but preserved in
	//   NetBox upon CR deletion
	// - When set to false, the resource will be cleaned up in NetBox
	//   upon CR deletion
	// Setting preserveInNetbox to true is mandatory if the user wants to restore
	// resources from NetBox (e.g. Sticky VIDs even if resources are deleted and
	// recreated in Kubernetes)
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// VlanStatus defines the observed state of Vlan
type VlanStatus struct {
	// The ID of the resource in NetBox
	VlanId int64 `json:"id,omitempty"`

	// Last updated, corresponds to the 'last_updated' returned by NetBox when NetBox Operator updates a resource in NetBox.
	// Format: date-time
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	VlanUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
//+kubebuilder:printcolumn:name="Vid",type=integer,JSONPath=`.spec.vid`
//+kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.site`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=vln

// Vlan allows to create a NetBox VLAN. More info about NetBox VLANs: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/vlan.md
type Vlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VlanSpec   `json:"spec,omitempty"`
	Status VlanStatus `json:"status,omitempty"`
}

func (v *Vlan) Conditions() *[]metav1.Condition {
	return &v.Status.Conditions
}

//+kubebuilder:object:root=true

// VlanList contains a list of Vlan
type VlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vlan `json:"items"`
}

func init() {
	register(&Vlan{}, &VlanList{})
}

var ConditionVlanReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "VLANReservedInNetbox",
	Message: "VLAN was reserved/updated in NetBox",
}

var ConditionVlanReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveVLANInNetbox",
	Message: "Failed to reserve VLAN in NetBox",
}

var ConditionVlanReadyFalseDeletionInProgress = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "DeletionInProgress",
	Message: "VLAN deletion in progress",
}

var ConditionVlanReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteVLANInNetbox",
	Message: "Failed to delete VLAN in NetBox",
}
