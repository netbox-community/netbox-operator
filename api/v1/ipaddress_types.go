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
	// The actual IP Address that should be reserved in NetBox
	IpAddress string `json:"ipAddress"`

	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	// The NetBox Tenant to be used for creating this resource in Netbox
	Tenant string `json:"tenant,omitempty"`

	// NetBox Custom Fields that should be added to the resource in NetBox. Note that currently only Text Type is supported (GitHub #129)
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Comment that should be added to the resource in NetBox
	Comments string `json:"comments,omitempty"`

	// Description that should be added to the resource in NetBox
	Description string `json:"description,omitempty"`

	// preserveInNetbox defines whether or not the Resource should stay in NetBox when the Kubernetes Resource is deleted
	// When set to true, the resource will not be deleted in NetBox upon CR deletion
	// When set to false, the resource will be cleaned up in NetBox upon CR deletion
	// If you want to restore resources from NetBox (e.g. recreation of an entire cluster), preserveInNetbox set to true is a prerequisite.
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// IpAddressStatus defines the observed state of IpAddress
type IpAddressStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The ID of the resource in NetBox
	IpAddressId int64 `json:"id,omitempty"`

	// The URL to the NetBox UI to display this resource. Note that the base depends on the runtime config of NetBox Operator
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
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
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

var ConditionIpaddressReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteIpInNetbox",
	Message: "Failed to delete IP in NetBox",
}
