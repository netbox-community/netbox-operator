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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AsnClaimSpec defines the desired state of AsnClaim
type AsnClaimSpec struct {
	// The name of the NetBox ASN Range from which this ASN should be claimed. Use the
	// `name` value instead of the `slug` value
	// Field is immutable, required
	// Example: "Private ASNs"
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentAsnRange' is immutable"
	ParentAsnRange string `json:"parentAsnRange"`

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
	// resources from NetBox (e.g. Sticky ASNs even if resources are deleted and
	// recreated in Kubernetes)
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// AsnClaimStatus defines the observed state of AsnClaim
type AsnClaimStatus struct {
	// The assigned ASN (Autonomous System Number)
	Asn int64 `json:"asn,omitempty"`

	// The name of the Asn CR created by the AsnClaim Controller
	AsnName string `json:"asnName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="ASN",type=integer,JSONPath=`.status.asn`
//+kubebuilder:printcolumn:name="ASNAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="ASNAssigned")].status`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=asnc

// AsnClaim allows to claim a NetBox ASN from an existing ASN Range.
// The AsnClaim Controller will try to assign an available ASN
// from the ASN Range that is defined in the spec and if successful it will create
// the Asn CR. More info about NetBox ASNs:
// https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/asn.md
type AsnClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AsnClaimSpec   `json:"spec,omitempty"`
	Status AsnClaimStatus `json:"status,omitempty"`
}

func (a *AsnClaim) Conditions() *[]metav1.Condition {
	return &a.Status.Conditions
}

//+kubebuilder:object:root=true

// AsnClaimList contains a list of AsnClaim
type AsnClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AsnClaim `json:"items"`
}

func init() {
	register(&AsnClaim{}, &AsnClaimList{})
}

var ConditionAsnClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "AsnResourceReady",
	Message: "ASN Resource is ready",
}

var ConditionAsnClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "AsnResourceNotReady",
	Message: "ASN Resource is not ready",
}

var ConditionAsnAssignedTrue = metav1.Condition{
	Type:    "ASNAssigned",
	Status:  "True",
	Reason:  "AsnCRCreated",
	Message: "New ASN fetched from NetBox and Asn CR was created",
}

var ConditionAsnAssignedFalse = metav1.Condition{
	Type:    "ASNAssigned",
	Status:  "False",
	Reason:  "AsnCRNotCreated",
	Message: "Failed to fetch new ASN from NetBox",
}
