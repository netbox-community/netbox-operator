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

// IpRangeSpec defines the desired state of IpRange
type IpRangeSpec struct {
	// The first IP in CIDR notation that should be included in the NetBox IP Range
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'startAddress' is immutable"
	//+kubebuilder:validation:Required
	StartAddress string `json:"startAddress"`

	// The last IP in CIDR notation that should be included in the NetBox IP Range
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'endAddress' is immutable"
	//+kubebuilder:validation:Required
	EndAddress string `json:"endAddress"`

	// The NetBox Tenant to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	// The NetBox Custom Fields that should be added to the resource in NetBox.
	// Note that currently only Text Type is supported (GitHub #129)
	// More info on NetBox Custom Fields:
	// https://github.com/netbox-community/netbox/blob/main/docs/customization/custom-fields.md
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Comment that should be added to the resource in NetBox
	Comments string `json:"comments,omitempty"`

	// Description that should be added to the resource in NetBox
	Description string `json:"description,omitempty"`

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

// IpRangeStatus defines the observed state of IpRange
type IpRangeStatus struct {
	// The ID of the resource in NetBox
	IpRangeId int64 `json:"id,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	IpRangeUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
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
// +kubebuilder:resource:shortName=ipr

// IpRange allows to create a NetBox IP Range. More info about NetBox IP Ranges: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/iprange.md
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
	Reason:  "IPRangeReservedInNetbox",
	Message: "IP Range was reserved/updated in NetBox",
}

var ConditionIpRangeReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveIPRangeInNetbox",
	Message: "Failed to reserve IP Range in NetBox",
}

var ConditionIpRangeReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteIPRangeInNetbox",
	Message: "Failed to delete IP Range in NetBox",
}
