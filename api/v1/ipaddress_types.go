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

// IpAddressSpec defines the desired state of IpAddress
type IpAddressSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'ipAddress' is immutable"
	//+kubebuilder:validation:Required
	IpAddress string `json:"ipAddress"`

	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	CustomFields map[string]string `json:"customFields,omitempty"`

	Comments string `json:"comments,omitempty"`

	Description string `json:"description,omitempty"`

	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// IpAddressStatus defines the observed state of IpAddress
type IpAddressStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	IpAddressId int64 `json:"id,omitempty"`

	IpAddressUrl string `json:"url,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="IpAddress",type=string,JSONPath=`.spec.ipAddress`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
// +kubebuilder:resource:shortName=ip

// IpAddress is the Schema for the ipaddresses API
type IpAddress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IpAddressSpec   `json:"spec,omitempty"`
	Status IpAddressStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IpAddressList contains a list of IpAddress
type IpAddressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IpAddress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IpAddress{}, &IpAddressList{})
}

var ConditionIpaddressReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "IpReservedInNetbox",
	Message: "IP was reserved/updated in NetBox",
}

var ConditionIpaddressReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveIpInNetbox",
	Message: "Failed to reserve IP in NetBox",
}
