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

// IpRangeClaimSpec defines the desired state of IpRangeClaim
type IpRangeClaimSpec struct {
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentPrefix' is immutable"
	ParentPrefix string `json:"parentPrefix"`

	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	CustomFields map[string]string `json:"customFields,omitempty"`

	Comments string `json:"comments,omitempty"`

	Description string `json:"description,omitempty"`

	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// IpRangeClaimStatus defines the observed state of IpRangeClaim
type IpRangeClaimStatus struct {
	IpRange string `json:"ipAddress,omitempty"`

	IpRangeDotDecimal string `json:"ipAddressDotDecimal,omitempty"`

	IpRangeName string `json:"ipAddressName,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="IpRange",type=string,JSONPath=`.status.ipRange`
//+kubebuilder:printcolumn:name="IpAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="IPAssigned")].status`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=iprc

// IpRangeClaim is the Schema for the iprangeclaims API
type IpRangeClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IpRangeClaimSpec   `json:"spec,omitempty"`
	Status IpRangeClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IpRangeClaimList contains a list of IpRangeClaim
type IpRangeClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IpRangeClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IpRangeClaim{}, &IpRangeClaimList{})
}

var ConditionIpRangeClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "IPAddressResourceReady",
	Message: "IPAddress Resource is ready",
}

var ConditionIpRangeClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "IPAddressResourceNotReady",
	Message: "IPAddress Resource is not ready",
}

var ConditionIpRangeAssignedTrue = metav1.Condition{
	Type:    "IPAssigned",
	Status:  "True",
	Reason:  "IPAddressCRCreated",
	Message: "New IP fetched from NetBox and IPAddress CR was created",
}

var ConditionIpRangeAssignedFalse = metav1.Condition{
	Type:    "IPAssigned",
	Status:  "False",
	Reason:  "IPAddressCRNotCreated",
	Message: "Failed to fetch new IP from NetBox",
}
