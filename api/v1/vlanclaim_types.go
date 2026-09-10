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

// VlanClaimSpec defines the desired state of VlanClaim
// +kubebuilder:validation:XValidation:rule="(has(self.vid) && !has(self.vidRangeStart) && !has(self.vidRangeEnd)) || (!has(self.vid) && has(self.vidRangeStart) && has(self.vidRangeEnd))",message="Exactly one of 'vid' or ('vidRangeStart' and 'vidRangeEnd') must be set"
type VlanClaimSpec struct {
	// The exact VID to claim. Mutually exclusive with vidRangeStart/vidRangeEnd.
	// Field is immutable, not required, range from 1-4094
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=4094
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'vid' is immutable"
	Vid int32 `json:"vid,omitempty"`

	// The lower bound (inclusive) of the range to pick a free VID from.
	// Mutually exclusive with vid, required together with vidRangeEnd.
	// Field is immutable
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'vidRangeStart' is immutable"
	VidRangeStart int32 `json:"vidRangeStart,omitempty"`

	// The upper bound (inclusive) of the range to pick a free VID from.
	// Mutually exclusive with vid, required together with vidRangeStart.
	// Field is immutable
	//+kubebuilder:validation:Maximum=4094
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'vidRangeEnd' is immutable"
	VidRangeEnd int32 `json:"vidRangeEnd,omitempty"`

	// The NetBox Site to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value.
	// Also scopes VID allocation: a free VID is looked up among VLANs assigned to this Site.
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

// VlanClaimStatus defines the observed state of VlanClaim
type VlanClaimStatus struct {
	// The assigned VID
	Vid int32 `json:"vid,omitempty"`

	// The name of the Vlan CR created by the VlanClaim Controller
	VlanName string `json:"vlanName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.site`
//+kubebuilder:printcolumn:name="Vid",type=integer,JSONPath=`.status.vid`
//+kubebuilder:printcolumn:name="VlanAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="VlanAssigned")].status`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=vlnc

// VlanClaim allows to claim a VID for a NetBox VLAN, either an exact one or
// the next free one in a range. The VlanClaim Controller will try to assign
// an available VID and if successful it will create the Vlan CR.
// More info about NetBox VLANs: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/vlan.md
type VlanClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VlanClaimSpec   `json:"spec,omitempty"`
	Status VlanClaimStatus `json:"status,omitempty"`
}

func (v *VlanClaim) Conditions() *[]metav1.Condition {
	return &v.Status.Conditions
}

//+kubebuilder:object:root=true

// VlanClaimList contains a list of VlanClaim
type VlanClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VlanClaim `json:"items"`
}

func init() {
	register(&VlanClaim{}, &VlanClaimList{})
}

var ConditionVlanClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "VlanResourceReady",
	Message: "VLAN Resource is ready",
}

var ConditionVlanClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "VlanResourceNotReady",
	Message: "VLAN Resource is not ready",
}

var ConditionVlanAssignedTrue = metav1.Condition{
	Type:    "VlanAssigned",
	Status:  "True",
	Reason:  "VlanCRCreated",
	Message: "New VID fetched from NetBox and Vlan CR was created",
}

var ConditionVlanAssignedFalse = metav1.Condition{
	Type:    "VlanAssigned",
	Status:  "False",
	Reason:  "VlanCRNotCreated",
	Message: "Failed to fetch new VID from NetBox",
}

var ConditionVlanAssignedFalseRangeExhausted = metav1.Condition{
	Type:    "VlanAssigned",
	Status:  "False",
	Reason:  "VlanCRNotCreated",
	Message: "No free VID available in the requested range",
}
