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

// VlanGroupSpec defines the desired state of VlanGroup
// +kubebuilder:validation:XValidation:rule="(has(self.vidRangeStart) && has(self.vidRangeEnd)) || (!has(self.vidRangeStart) && !has(self.vidRangeEnd))",message="Fields 'vidRangeStart' and 'vidRangeEnd' must both be set, or neither"
type VlanGroupSpec struct {
	// The name of the VLAN Group in NetBox
	// Field is immutable, required
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'name' is immutable"
	Name string `json:"name"`

	// The NetBox Site this VLAN Group is scoped to. Use the `name` value instead of the `slug` value
	// Field is mutable, not required
	Site string `json:"site,omitempty"`

	// The lower bound (inclusive) of the VID range allowed in this VLAN Group.
	// Not required, must be set together with vidRangeEnd
	// Field is mutable
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=4094
	VidRangeStart int32 `json:"vidRangeStart,omitempty"`

	// The upper bound (inclusive) of the VID range allowed in this VLAN Group.
	// Not required, must be set together with vidRangeStart
	// Field is mutable
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=4094
	VidRangeEnd int32 `json:"vidRangeEnd,omitempty"`

	// The NetBox Tenant to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	// Field is mutable, not required
	// Example: "Initech" or "Cyberdyne Systems"
	Tenant string `json:"tenant,omitempty"`

	// The NetBox Custom Fields that should be added to the resource in NetBox.
	// Note that currently only Text Type is supported (GitHub #129)
	// More info on NetBox Custom Fields:
	// https://github.com/netbox-community/netbox/blob/main/docs/customization/custom-fields.md
	// Field is mutable, not required
	// Example:
	//   customfield1: "Production"
	//   customfield2: "This is a string"
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Description that should be added to the resource in NetBox
	// Field is mutable, not required
	Description string `json:"description,omitempty"`

	// Defines whether the Resource should be preserved in NetBox when the
	// Kubernetes Resource is deleted.
	// - When set to true, the resource will not be deleted but preserved in
	//   NetBox upon CR deletion
	// - When set to false, the resource will be cleaned up in NetBox
	//   upon CR deletion
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// VlanGroupStatus defines the observed state of VlanGroup
type VlanGroupStatus struct {
	// The ID of the resource in NetBox
	VlanGroupId int64 `json:"id,omitempty"`

	// The slug generated for the resource in NetBox
	Slug string `json:"slug,omitempty"`

	// Last updated, corresponds to the 'last_updated' returned by NetBox when NetBox Operator updates a resource in NetBox.
	// Format: date-time
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	VlanGroupUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
//+kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.site`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=vlng

// VlanGroup allows to create a NetBox VLAN Group. More info about NetBox VLAN Groups: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/vlangroup.md
type VlanGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VlanGroupSpec   `json:"spec,omitempty"`
	Status VlanGroupStatus `json:"status,omitempty"`
}

func (v *VlanGroup) Conditions() *[]metav1.Condition {
	return &v.Status.Conditions
}

//+kubebuilder:object:root=true

// VlanGroupList contains a list of VlanGroup
type VlanGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VlanGroup `json:"items"`
}

func init() {
	register(&VlanGroup{}, &VlanGroupList{})
}

var ConditionVlanGroupReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "VLANGroupReservedInNetbox",
	Message: "VLAN Group was reserved/updated in NetBox",
}

var ConditionVlanGroupReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveVLANGroupInNetbox",
	Message: "Failed to reserve VLAN Group in NetBox",
}

var ConditionVlanGroupReadyFalseDeletionInProgress = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "DeletionInProgress",
	Message: "VLAN Group deletion in progress",
}

var ConditionVlanGroupReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteVLANGroupInNetbox",
	Message: "Failed to delete VLAN Group in NetBox",
}
