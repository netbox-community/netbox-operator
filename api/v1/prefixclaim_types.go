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

// PrefixClaimSpec defines the desired state of PrefixClaim
// TODO: The reason for using a workaround please see https://github.com/netbox-community/netbox-operator/pull/90#issuecomment-2402112475
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.site) || has(self.site)", message="Site is required once set"
// +kubebuilder:validation:XValidation:rule="(!has(self.parentPrefix) && has(self.parentPrefixSelector)) || (has(self.parentPrefix) && !has(self.parentPrefixSelector))"
type PrefixClaimSpec struct {
	// The NetBox Prefix from which this Prefix should be claimed from
	// Field is immutable, required (`parentPrefix` and `parentPrefixSelector` are mutually exclusive)
	// Example: "192.168.0.0/20"
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentPrefix' is immutable"
	ParentPrefix string `json:"parentPrefix,omitempty"`

	// The `parentPrefixSelector` is a key-value map, where all the entries are of data type `<string-string>` The map contains a set of query conditions for selecting a set of prefixes that can be used as the parent prefix The query conditions will be chained by the AND operator, and exact match of the keys and values will be performed The built-in fields `tenant`, `site`, and `family`, along with custom fields, can be used. Note that since the key value pairs in this map are used to generate the URL for the query in NetBox, this also supports non-Text Custom Field types. For more information, please see ParentPrefixSelectorGuide.md
	// Field is immutable, required (`parentPrefix` and `parentPrefixSelector` are mutually exclusive)
	// Example:
	//   customfield1: "Production"
	//   family: "IPv4"
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentPrefixSelector' is immutable"
	//+kubebuilder:validation:XValidation:rule="!has(self.family) || (self.family == 'IPv4' || self.family == 'IPv6')"
	ParentPrefixSelector map[string]string `json:"parentPrefixSelector,omitempty"`

	// The desired prefix length of your Prefix using slash notation. Example: `/24` for an IPv4 Prefix or `/64` for an IPv6 Prefix
	// Field is immutable, required
	// Example: "/24"
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Pattern=`^\/[0-9]|[1-9][0-9]|1[01][0-9]|12[0-8]$`
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'prefixLength' is immutable"
	PrefixLength string `json:"prefixLength"`

	// The NetBox Site to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	// Field is immutable, not required
	// Example: "DM-Buffalo"
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'site' is immutable"
	Site string `json:"site,omitempty"`

	// The NetBox Tenant to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	// Field is immutable, not required
	// Example: "Initech" or "Cyberdyne Systems"
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	// Description that should be added to the resource in NetBox
	// Field is mutable, not required
	Description string `json:"description,omitempty"`

	// Comment that should be added to the resource in NetBox
	// Field is mutable, not required
	Comments string `json:"comments,omitempty"`

	// The NetBox Custom Fields that should be added to the resource in NetBox.
	// Note that currently only Text Type is supported (GitHub #129)
	// More info on NetBox Custom Fields:
	// https://github.com/netbox-community/netbox/blob/main/docs/customization/custom-fields.md
	// Field is mutable, not required
	// Example:
	//   customfield1: "Production"
	//   customfield2: "This is a string"
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Defines whether the Resource should be preserved in NetBox when the
	// Kubernetes Resource is deleted.
	// - When set to true, the resource will not be deleted but preserved in
	//   NetBox upon CR deletion
	// - When set to false, the resource will be cleaned up in NetBox
	//   upon CR deletion
	// Setting preserveInNetbox to true is mandatory if the user wants to restore
	// resources from NetBox (e.g. Sticky CIDRs even if resources are deleted and
	// recreated in Kubernetes)
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// PrefixClaimStatus defines the observed state of PrefixClaim
type PrefixClaimStatus struct {
	// Due to the fact that the parentPrefix can be specified directly in
	//`.spec.parentPrefix` or selected from `.spec.parentPrefixSelector`,
	//we use this field to store exactly which parent prefix we are using
	//for all subsequent reconcile loop calls.
	SelectedParentPrefix string `json:"parentPrefix,omitempty"`

	// The assigned Prefix in CIDR notation
	Prefix string `json:"prefix,omitempty"`

	// The name of the Prefix CR created by the PrefixClaim Controller
	PrefixName string `json:"prefixName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Prefix",type=string,JSONPath=`.status.prefix`
// +kubebuilder:printcolumn:name="PrefixAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="PrefixAssigned")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=pxc

// PrefixClaim allows to claim a NetBox Prefix from an existing Prefix
// (parentPrefix) or a dynamically selected Prefix (parentPrefixSelector).
// The PrefixClaim Controller will try to assign an available Prefix from
// the Prefix that is defined in the spec and if successful it will create
// the Prefix CR. More info about NetBox IP Addresses:
// https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/ipaddress.md
type PrefixClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrefixClaimSpec   `json:"spec,omitempty"`
	Status PrefixClaimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PrefixClaimList contains a list of PrefixClaim
type PrefixClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrefixClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrefixClaim{}, &PrefixClaimList{})
}

var ConditionPrefixClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "PrefixClaimResourceReady",
	Message: "PrefixClaim Resource is ready",
}

var ConditionPrefixClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "PrefixClaimResourceNotReady",
	Message: "PrefixClaim Resource is not ready",
}

var ConditionPrefixAssignedTrue = metav1.Condition{
	Type:    "PrefixAssigned",
	Status:  "True",
	Reason:  "PrefixCRCreated",
	Message: "New prefix fetched from NetBox and prefix CR was created",
}

var ConditionPrefixAssignedFalse = metav1.Condition{
	Type:    "PrefixAssigned",
	Status:  "False",
	Reason:  "PrefixCRNotCreated",
	Message: "Failed to fetch new Prefix from NetBox",
}

var ConditionParentPrefixSelectedTrue = metav1.Condition{
	Type:    "ParentPrefixSelected",
	Status:  "True",
	Reason:  "ParentPrefixSelected",
	Message: "The parent prefix was selected successfully",
}

var ConditionParentPrefixSelectedFalse = metav1.Condition{
	Type:    "ParentPrefixSelected",
	Status:  "False",
	Reason:  "ParentPrefixNotSelected",
	Message: "The parent prefix was not able to be selected",
}
