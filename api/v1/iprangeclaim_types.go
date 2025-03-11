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

// IpRangeClaimSpec defines the desired state of IpRangeClaim
type IpRangeClaimSpec struct {
	// The NetBox Prefix from which this IP Range should be claimed from
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentPrefix' is immutable"
	ParentPrefix string `json:"parentPrefix"`

	// The amount of consecutive IP Addresses you wish to reserve.
	// Currently only sizes up to 50 are supported due to pagination of the
	// NetBox API. In practice, this might be even lower depending on the
	// fragmentation of the parent prefix.
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=2
	//+kubebuilder:validation:Maximum=50
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'size' is immutable"
	Size int `json:"size,omitempty"`

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

// IpRangeClaimStatus defines the observed state of IpRangeClaim
type IpRangeClaimStatus struct {
	// The assigned IP Range in CIDR notation (e.g. 192.168.0.1/32-192.168.0.123/32)
	IpRange string `json:"ipRange,omitempty"`

	// The assigned IP Range in Dot Decimal notation (e.g. 192.168.0.1-192.168.0.123)
	IpRangeDotDecimal string `json:"ipRangeDotDecimal,omitempty"`

	// The full list of IP Addresses in CIDR notation
	IpAddresses []string `json:"ipAddresses,omitempty"`

	// The full list of IP Addresses in Dot Decimal notation
	IpAddressesDotDecimal []string `json:"ipAddressesDotDecimal,omitempty"`

	// The first IP Addresses in CIDR notation
	StartAddress string `json:"startAddress,omitempty"`

	// The first IP Addresses in Dot Decimal notation
	StartAddressDotDecimal string `json:"startAddressDotDecimal,omitempty"`

	// The last IP Addresses in CIDR notation
	EndAddress string `json:"endAddress,omitempty"`

	// The last IP Addresses in Dot Decimal notation
	EndAddressDotDecimal string `json:"endAddressDotDecimal,omitempty"`

	// The name of the IpRange CR created by the IpRangeClaim Controller
	IpRangeName string `json:"ipAddressName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="IpRange",type=string,JSONPath=`.status.ipRange`
//+kubebuilder:printcolumn:name="IpRangeAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="IPRangeAssigned")].status`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=iprc

// IpRangeClaim allows to claim a NetBox IP Range from an existing Prefix.
// The IpRangeClaim Controller will try to assign an available IP Range
// from the Prefix that is defined in the spec and if successful it will create
// the IpRange CR. More info about NetBox IP Ranges:
// https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/iprange.md
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
	Reason:  "IPRangeResourceReady",
	Message: "IP Range Resource is ready",
}

var ConditionIpRangeClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "IPRangeResourceNotReady",
	Message: "IP Range Resource is not ready",
}

var ConditionIpRangeClaimReadyFalseStatusGen = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "IPRangeClaimStatusGenerationFailed",
	Message: "Failed to generate IP Range Status",
}

var ConditionIpRangeAssignedTrue = metav1.Condition{
	Type:    "IPRangeAssigned",
	Status:  "True",
	Reason:  "IPRangeCRCreated",
	Message: "New IP Range fetched from NetBox and IpRange CR was created",
}

var ConditionIpRangeAssignedFalse = metav1.Condition{
	Type:    "IPRangeAssigned",
	Status:  "False",
	Reason:  "IPRangeCRNotCreated",
	Message: "Failed to fetch new IP Range from NetBox",
}

var ConditionIpRangeAssignedFalseSizeMissmatch = metav1.Condition{
	Type:    "IPRangeAssigned",
	Status:  "False",
	Reason:  "IPRangeCRNotCreated",
	Message: "Assigned/Restored IP range has less available IP addresses than requested",
}
