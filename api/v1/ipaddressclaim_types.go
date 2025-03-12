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

// IpAddressClaimSpec defines the desired state of IpAddressClaim
type IpAddressClaimSpec struct {
	// The NetBox Prefix from which this IP Address should be claimed from
	// Field is immutable, required
	// Example: "192.168.0.0/20"
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'parentPrefix' is immutable"
	ParentPrefix string `json:"parentPrefix"`

	// The NetBox Tenant to be assigned to this resource in NetBox. Use the `name` value instead of the `slug` value
	// Field is immutable, not required
	// Example: "Initech" or "Cyberdyne Systems"
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'tenant' is immutable"
	Tenant string `json:"tenant,omitempty"`

	// The NetBox Custom Fields that should be added to the resource in NetBox.
	// Note that currently only Text Type is supported (GitHub #129)
	// More info on NetBox Custom Fields:
	// https://github.com/netbox-community/netbox/blob/main/docs/customization/custom-fields.md
	// Field is mutable, not required
	// Example:
	//   customfield1: "Production"
	//   customfield2: "This is a string"
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Comment that should be added to the resource in NetBox
	// Field is mutable, not required
	Comments string `json:"comments,omitempty"`

	// Description that should be added to the resource in NetBox
	// Field is mutable, not required
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
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// IpAddressClaimStatus defines the observed state of IpAddressClaim
type IpAddressClaimStatus struct {
	// The assigned IP Address in CIDR notation
	IpAddress string `json:"ipAddress,omitempty"`

	// The assigned IP Address in Dot Decimal notation
	IpAddressDotDecimal string `json:"ipAddressDotDecimal,omitempty"`

	// The name of the IpAddress CR created by the IpAddressClaim Controller
	IpAddressName string `json:"ipAddressName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="IpAddress",type=string,JSONPath=`.status.ipAddress`
//+kubebuilder:printcolumn:name="IpAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="IPAssigned")].status`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=ipc

// IpAddressClaim allows to claim a NetBox IP Address from an existing Prefix.
// The IpAddressClaim Controller will try to assign an available IP Address
// from the Prefix that is defined in the spec and if successful it will create
// the IpAddress CR. More info about NetBox IP Addresses:
// https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/ipaddress.md
type IpAddressClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IpAddressClaimSpec   `json:"spec,omitempty"`
	Status IpAddressClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IpAddressClaimList contains a list of IpAddressClaim
type IpAddressClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IpAddressClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IpAddressClaim{}, &IpAddressClaimList{})
}

var ConditionIpClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "IPAddressResourceReady",
	Message: "IPAddress Resource is ready",
}

var ConditionIpClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "IPAddressResourceNotReady",
	Message: "IPAddress Resource is not ready",
}

var ConditionIpAssignedTrue = metav1.Condition{
	Type:    "IPAssigned",
	Status:  "True",
	Reason:  "IPAddressCRCreated",
	Message: "New IP fetched from NetBox and IPAddress CR was created",
}

var ConditionIpAssignedFalse = metav1.Condition{
	Type:    "IPAssigned",
	Status:  "False",
	Reason:  "IPAddressCRNotCreated",
	Message: "Failed to fetch new IP from NetBox",
}
