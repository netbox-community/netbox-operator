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

// L2VPNClaimSpec defines the desired state of L2VPNClaim
// +kubebuilder:validation:XValidation:rule="(has(self.identifier) && !has(self.identifierRangeStart) && !has(self.identifierRangeEnd)) || (!has(self.identifier) && has(self.identifierRangeStart) && has(self.identifierRangeEnd))",message="Exactly one of 'identifier' or ('identifierRangeStart' and 'identifierRangeEnd') must be set"
type L2VPNClaimSpec struct {
	// The NetBox L2VPN type. Only VXLAN-based types are supported, since those
	// are the ones that carry a VNI in their identifier.
	// Field is immutable, required
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Enum=vxlan;vxlan-evpn
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'type' is immutable"
	Type string `json:"type"`

	// The exact VNI to claim. Mutually exclusive with identifierRangeStart/identifierRangeEnd.
	// Field is immutable, not required, range from 4000-16777215
	//+kubebuilder:validation:Minimum=4000
	//+kubebuilder:validation:Maximum=16777215
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'identifier' is immutable"
	Identifier int64 `json:"identifier,omitempty"`

	// The lower bound (inclusive) of the range to pick a free VNI from.
	// Mutually exclusive with identifier, required together with identifierRangeEnd.
	// Field is immutable
	//+kubebuilder:validation:Minimum=4000
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'identifierRangeStart' is immutable"
	IdentifierRangeStart int64 `json:"identifierRangeStart,omitempty"`

	// The upper bound (inclusive) of the range to pick a free VNI from.
	// Mutually exclusive with identifier, required together with identifierRangeStart.
	// Field is immutable
	//+kubebuilder:validation:Maximum=16777215
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'identifierRangeEnd' is immutable"
	IdentifierRangeEnd int64 `json:"identifierRangeEnd,omitempty"`

	// The NetBox Tenant to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	// Field is immutable, not required
	// Example: "Initech" or "Cyberdyne Systems"
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
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
	// resources from NetBox (e.g. Sticky VNIs even if resources are deleted and
	// recreated in Kubernetes)
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// L2VPNClaimStatus defines the observed state of L2VPNClaim
type L2VPNClaimStatus struct {
	// The assigned VNI
	Identifier int64 `json:"identifier,omitempty"`

	// The name of the L2VPN CR created by the L2VPNClaim Controller
	L2VPNName string `json:"l2VPNName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Identifier",type=integer,JSONPath=`.status.identifier`
//+kubebuilder:printcolumn:name="L2VPNAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="L2VPNAssigned")].status`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=l2vc

// L2VPNClaim allows to claim a VNI for a NetBox L2VPN, either an exact one or
// the next free one in a range. The L2VPNClaim Controller will try to assign
// an available identifier and if successful it will create the L2VPN CR.
// More info about NetBox L2VPNs: https://github.com/netbox-community/netbox/blob/main/docs/models/vpn/l2vpn.md
type L2VPNClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   L2VPNClaimSpec   `json:"spec,omitempty"`
	Status L2VPNClaimStatus `json:"status,omitempty"`
}

func (l *L2VPNClaim) Conditions() *[]metav1.Condition {
	return &l.Status.Conditions
}

//+kubebuilder:object:root=true

// L2VPNClaimList contains a list of L2VPNClaim
type L2VPNClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []L2VPNClaim `json:"items"`
}

func init() {
	register(&L2VPNClaim{}, &L2VPNClaimList{})
}

var ConditionL2VPNClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "L2VPNResourceReady",
	Message: "L2VPN Resource is ready",
}

var ConditionL2VPNClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "L2VPNResourceNotReady",
	Message: "L2VPN Resource is not ready",
}

var ConditionL2VPNAssignedTrue = metav1.Condition{
	Type:    "L2VPNAssigned",
	Status:  "True",
	Reason:  "L2VPNCRCreated",
	Message: "New identifier fetched from NetBox and L2VPN CR was created",
}

var ConditionL2VPNAssignedFalse = metav1.Condition{
	Type:    "L2VPNAssigned",
	Status:  "False",
	Reason:  "L2VPNCRNotCreated",
	Message: "Failed to fetch new identifier from NetBox",
}

var ConditionL2VPNAssignedFalseRangeExhausted = metav1.Condition{
	Type:    "L2VPNAssigned",
	Status:  "False",
	Reason:  "L2VPNCRNotCreated",
	Message: "No free identifier available in the requested range",
}
