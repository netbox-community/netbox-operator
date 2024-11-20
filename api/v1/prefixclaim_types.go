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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PrefixClaimSpec defines the desired state of PrefixClaim
// TODO: The reason for using a workaround please see https://github.com/netbox-community/netbox-operator/pull/90#issuecomment-2402112475
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.site) || has(self.site)", message="Site is required once set"
// +kubebuilder:validation:XValidation:rule="(!has(self.parentPrefix) && has(self.parentPrefixSelector)) || (has(self.parentPrefix) && !has(self.parentPrefixSelector))"
type PrefixClaimSpec struct {

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentPrefix' is immutable"
	ParentPrefix string `json:"parentPrefix,omitempty"`

	// The `parentPrefixSelector` is a key-value map, where all the entries are of data type `<string-string>`
	// The map contains a set of query conditions for selecting a set of prefixes that can be used as the parent prefix
	// The query conditions will be chained by the AND operator, and exact match of the keys and values will be performed
	// The built-in fields `tenant`, `site`, and `family`, along with custom fields, can be used
	// For more information, please see ParentPrefixSelectorGuide.md
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentPrefixSelector' is immutable"
	//+kubebuilder:validation:XValidation:rule="!has(self.family) || (self.family == 'IPv4' || self.family == 'IPv6')"
	ParentPrefixSelector map[string]string `json:"parentPrefixSelector,omitempty"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Pattern=`^\/[0-9]|[1-9][0-9]|1[01][0-9]|12[0-8]$`
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'prefixLength' is immutable"
	PrefixLength string `json:"prefixLength"`

	// Use the `name` value instead of the `slug` value
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'site' is immutable"
	Site string `json:"site,omitempty"`

	// Use the `name` value instead of the `slug` value
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	Description string `json:"description,omitempty"`

	Comments string `json:"comments,omitempty"`

	CustomFields map[string]string `json:"customFields,omitempty"`

	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// PrefixClaimStatus defines the observed state of PrefixClaim
type PrefixClaimStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Prefix status: container, active, reserved, deprecated

	// Due to the fact that the parent prefix can be specified directly in `ParentPrefix` or selected from `ParentPrefixSelector`,
	// we use this field to store exactly which parent prefix we are using for all subsequent reconcile loop calls,
	// as Spec.ParentPrefix is an immutable field, we can't overwrite it
	SelectedParentPrefix string             `json:"parentPrefix,omitempty"`
	Prefix               string             `json:"prefix,omitempty"`
	PrefixName           string             `json:"prefixName,omitempty"`
	Conditions           []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Prefix",type=string,JSONPath=`.status.prefix`
// +kubebuilder:printcolumn:name="PrefixAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="PrefixAssigned")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=pxc
// PrefixClaim is the Schema for the prefixclaims API
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
