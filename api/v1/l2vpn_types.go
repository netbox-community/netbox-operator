/*
Copyright 2026 Swisscom (Schweiz) AG.

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

// L2VPNSpec defines the desired state of L2VPN
type L2VPNSpec struct {
	// The name of the L2VPN in NetBox
	// Field is immutable, required
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'name' is immutable"
	Name string `json:"name"`

	// The NetBox L2VPN type. Only VXLAN-based types are supported, since those
	// are the ones that carry a VNI in their identifier.
	// Field is immutable, required
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Enum=vxlan;vxlan-evpn
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'type' is immutable"
	Type string `json:"type"`

	// The VNI to be assigned to this L2VPN in NetBox
	// Field is immutable, required, range from 4000-16777215
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=4000
	//+kubebuilder:validation:Maximum=16777215
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'identifier' is immutable"
	Identifier int64 `json:"identifier"`

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
	// resources from NetBox (e.g. Sticky VNIs even if resources are deleted and
	// recreated in Kubernetes)
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// L2VPNStatus defines the observed state of L2VPN
type L2VPNStatus struct {
	// The ID of the resource in NetBox
	L2VPNId int64 `json:"id,omitempty"`

	// The slug generated for the resource in NetBox
	Slug string `json:"slug,omitempty"`

	// Last updated, corresponds to the 'last_updated' returned by NetBox when NetBox Operator updates a resource in NetBox.
	// Format: date-time
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	L2VPNUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Identifier",type=integer,JSONPath=`.spec.identifier`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=l2v

// L2VPN allows to create a NetBox L2VPN, e.g. to track a VXLAN VNI. More info about NetBox L2VPNs: https://github.com/netbox-community/netbox/blob/main/docs/models/vpn/l2vpn.md
type L2VPN struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   L2VPNSpec   `json:"spec,omitempty"`
	Status L2VPNStatus `json:"status,omitempty"`
}

func (l *L2VPN) Conditions() *[]metav1.Condition {
	return &l.Status.Conditions
}

//+kubebuilder:object:root=true

// L2VPNList contains a list of L2VPN
type L2VPNList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []L2VPN `json:"items"`
}

func init() {
	register(&L2VPN{}, &L2VPNList{})
}

var ConditionL2VPNReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "L2VPNReservedInNetbox",
	Message: "L2VPN was reserved/updated in NetBox",
}

var ConditionL2VPNReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveL2VPNInNetbox",
	Message: "Failed to reserve L2VPN in NetBox",
}

var ConditionL2VPNReadyFalseDeletionInProgress = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "DeletionInProgress",
	Message: "L2VPN deletion in progress",
}

var ConditionL2VPNReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteL2VPNInNetbox",
	Message: "Failed to delete L2VPN in NetBox",
}
