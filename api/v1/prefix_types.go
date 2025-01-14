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

// PrefixSpec defines the desired state of Prefix
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.site) || has(self.site)", message="Site is required once set"
type PrefixSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'prefix' is immutable"
	Prefix string `json:"prefix"`

	//+kubebuilder:validation:XValidation:rule="self == oldSelf || self != ''",message="Field 'site' is required once set"
	Site string `json:"site,omitempty"`

	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	CustomFields map[string]string `json:"customFields,omitempty"`

	Description string `json:"description,omitempty"`

	Comments string `json:"comments,omitempty"`

	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// PrefixStatus defines the observed state of Prefix
type PrefixStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Prefix status: container, active, reserved , deprecated
	PrefixId int64 `json:"id,omitempty"`

	PrefixUrl string `json:"url,omitempty"`

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
// Prefix is the Schema for the prefixes API
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
