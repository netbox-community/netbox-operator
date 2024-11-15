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

// IpRangeSpec defines the desired state of IpRange
type IpRangeSpec struct {
	// the startAddress is the first ip address included in the ip range
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'startAddress' is immutable"
	//+kubebuilder:validation:Required
	StartAddress string `json:"startAddress"`

	// the endAddress is the last ip address included in the ip range
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'endAddress' is immutable"
	//+kubebuilder:validation:Required
	EndAddress string `json:"endAddress"`

	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	CustomFields map[string]string `json:"customFields,omitempty"`

	Comments string `json:"comments,omitempty"`

	Description string `json:"description,omitempty"`

	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// IpRangeStatus defines the observed state of IpRange
type IpRangeStatus struct {
	IpRangeId int64 `json:"id,omitempty"`

	IpRangeUrl string `json:"url,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="StartAddress",type=string,JSONPath=`.spec.startAddress`
//+kubebuilder:printcolumn:name="EndAddress",type=string,JSONPath=`.spec.endAddress`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=ir

// IpRange is the Schema for the ipranges API
type IpRange struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IpRangeSpec   `json:"spec,omitempty"`
	Status IpRangeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IpRangeList contains a list of IpRange
type IpRangeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IpRange `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IpRange{}, &IpRangeList{})
}

var ConditionIpRangeReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "IpRangeReservedInNetbox",
	Message: "Ip Range was reserved/updated in NetBox",
}

var ConditionIpRangeReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveIpRangeInNetbox",
	Message: "Failed to reserve Ip Range in NetBox",
}

var ConditionIpRangeReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteIpRangeInNetbox",
	Message: "Failed to delete Ip Range in NetBox",
}
