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

// IpAddressSpec defines the desired state of IpAddress
type IpAddressSpec struct {
	// The IP Address in CIDR notation that should be reserved in NetBox
	// Field is immutable, required
	// Example: "192.168.0.1/32"
	//+kubebuilder:validation:Format=cidr
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'ipAddress' is immutable"
	//+kubebuilder:validation:Required
	IpAddress string `json:"ipAddress"`

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

// IpAddressStatus defines the observed state of IpAddress
type IpAddressStatus struct {
	// The ID of the resource in NetBox
	IpAddressId int64 `json:"id,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	IpAddressUrl string `json:"url,omitempty"`

	// Indicates if Sync with the backend was successful
	// If connection to the backend failed but the spec did not change it is set to unknmown
	SyncState SyncState `json:"syncState,omitempty"`

	// Generation observed during the last reconciliation
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions represent the latest available observations of an object's state
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
//+kubebuilder:resource:shortName=ipa

// IpAddress allows to create a NetBox IP Address. More info about NetBox IP Addresses: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/ipaddress.md
type IpAddress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IpAddressSpec   `json:"spec,omitempty"`
	Status IpAddressStatus `json:"status,omitempty"`
}

func (i *IpAddress) Conditions() *[]metav1.Condition {
	return &i.Status.Conditions
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

var ConditionIpaddressReadyFalseUpdateFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToUpdateIpInNetbox",
	Message: "Failed to update IP in NetBox",
}

var ConditionIpaddressReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteIpInNetbox",
	Message: "Failed to delete IP in NetBox",
}

type SyncState string

const (
	SyncStateUnknown   SyncState = "Unknown"
	SyncStateSucceeded SyncState = "Succeeded"
	SyncStateFailed    SyncState = "Failed"
)
