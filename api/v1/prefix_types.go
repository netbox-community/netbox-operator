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

// PrefixSpec defines the desired state of Prefix
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.site) || has(self.site)", message="Site is required once set"
type PrefixSpec struct {
	// The Prefix in CIDR notation that should be reserved in NetBox
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'prefix' is immutable"
	Prefix string `json:"prefix"`

	// The NetBox Site to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	//+kubebuilder:validation:XValidation:rule="self == oldSelf || self != ''",message="Field 'site' is required once set"
	Site string `json:"site,omitempty"`

	// The NetBox Tenant to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	// The NetBox Custom Fields that should be added to the resource in NetBox.
	// Note that currently only Text Type is supported (GitHub #129)
	// More info on NetBox Custom Fields:
	// https://github.com/netbox-community/netbox/blob/main/docs/customization/custom-fields.md
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Description that should be added to the resource in NetBox
	Description string `json:"description,omitempty"`

	// Comment that should be added to the resource in NetBox
	Comments string `json:"comments,omitempty"`

	// Defines whether the Resource should be preserved in NetBox when the
	// Kubernetes Resource is deleted.
	// - When set to true, the resource will not be deleted but preserved in
	//   NetBox upon CR deletion
	// - When set to false, the resource will be cleaned up in NetBox
	//   upon CR deletion
	// Setting preserveInNetbox to true is mandatory if the user wants to restore
	// resources from NetBox (e.g. Sticky CIDRs even if resources are deleted and
	// recreated in Kubernetes)
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// PrefixStatus defines the observed state of Prefix
type PrefixStatus struct {
	// The ID of the resource in NetBox
	PrefixId int64 `json:"id,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	PrefixUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Prefix",type=string,JSONPath=`.spec.prefix`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=px

// Prefix allows to create a NetBox Prefix. More info about NetBox Prefixes: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/prefix.md
type Prefix struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrefixSpec   `json:"spec,omitempty"`
	Status PrefixStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PrefixList contains a list of Prefix
type PrefixList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Prefix `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Prefix{}, &PrefixList{})
}

var ConditionPrefixReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "PrefixReservedInNetbox",
	Message: "Prefix was reserved in NetBox",
}

var ConditionPrefixReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReservePrefixInNetbox",
	Message: "Failed to reserve prefix in NetBox",
}

var ConditionPrefixReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeletePrefixInNetbox",
	Message: "Failed to delete prefix in Netbox",
}
