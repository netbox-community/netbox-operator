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

// AsnSpec defines the desired state of Asn
type AsnSpec struct {
	// The ASN (Autonomous System Number) that should be reserved in NetBox
	// Field is immutable, required
	// Example: 65001
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=4294967295
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'asn' is immutable"
	Asn int64 `json:"asn"`

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

// AsnStatus defines the observed state of Asn
type AsnStatus struct {
	// The ID of the resource in NetBox
	AsnId int64 `json:"id,omitempty"`

	// Last updated, corresponds to the 'last_updated' returned by NetBox when NetBox Operator updates a resource in NetBox.
	// Format: date-time
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	AsnUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="ASN",type=integer,JSONPath=`.spec.asn`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=asn

// Asn allows to create a NetBox ASN (Autonomous System Number). More info about NetBox ASNs: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/asn.md
type Asn struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AsnSpec   `json:"spec,omitempty"`
	Status AsnStatus `json:"status,omitempty"`
}

func (a *Asn) Conditions() *[]metav1.Condition {
	return &a.Status.Conditions
}

//+kubebuilder:object:root=true

// AsnList contains a list of Asn
type AsnList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Asn `json:"items"`
}

func init() {
	register(&Asn{}, &AsnList{})
}

var ConditionAsnReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "AsnReservedInNetbox",
	Message: "ASN was reserved/updated in NetBox",
}

var ConditionAsnReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveAsnInNetbox",
	Message: "Failed to reserve ASN in NetBox",
}

var ConditionAsnReadyFalseDeletionInProgress = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "DeletionInProgress",
	Message: "ASN deletion in progress",
}

var ConditionAsnReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteAsnInNetbox",
	Message: "Failed to delete ASN in NetBox",
}
