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

// VLANClaimSpec defines the desired state of VLANClaim
type VLANClaimSpec struct {
	// The unique VLAN ID (VID) for the NetBox VLAN. If not provided, the operator will claim an available VID.
	//+optional
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=4094
	VlanId int `json:"vlanId,omitempty"`

	// The desired name for the VLAN in NetBox. If not provided, the operator will generate one.
	//+optional
	Name string `json:"name,omitempty"`

	// The NetBox Site where this VLAN should exist
	//+kubebuilder:validation:Required
	Site string `json:"site"`

	// The NetBox VLANGroup where this VLAN should be organized. Required if vlanId is not provided.
	//+optional
	VlanGroup string `json:"vlanGroup,omitempty"`

	// Description that should be added to the resource in NetBox
	//+optional
	Description string `json:"description,omitempty"`

	// Comment that should be added to the resource in NetBox
	//+optional
	Comments string `json:"comments,omitempty"`

	// The NetBox Custom Fields that should be added to the resource in NetBox
	//+optional
	CustomFields map[string]string `json:"customFields,omitempty"`

	// Defines whether the Resource should be preserved in NetBox when the
	// Kubernetes Resource is deleted.
	//+optional
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// VLANClaimStatus defines the observed state of VLANClaim
type VLANClaimStatus struct {
	// The assigned VLAN ID (VID)
	//+optional
	VlanId int `json:"vlanId,omitempty"`

	// The name of the Vlan CR created by the VLANClaim Controller
	//+optional
	VlanName string `json:"vlanName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	//+optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="VID",type=integer,JSONPath=`.status.vlanId`
//+kubebuilder:printcolumn:name="Vlan Name",type=string,JSONPath=`.status.vlanName`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=vlc

// VLANClaim is the Schema for the vlanclaims API
type VLANClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VLANClaimSpec   `json:"spec,omitempty"`
	Status VLANClaimStatus `json:"status,omitempty"`
}

func (v *VLANClaim) Conditions() *[]metav1.Condition {
	return &v.Status.Conditions
}

//+kubebuilder:object:root=true

// VLANClaimList contains a list of VLANClaim
type VLANClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VLANClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VLANClaim{}, &VLANClaimList{})
}

var ConditionVlanClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "VlanResourceReady",
	Message: "VLAN Resource is ready",
}

var ConditionVlanClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "VlanResourceNotReady",
	Message: "VLAN Resource is not ready",
}

var ConditionVlanAssignedTrue = metav1.Condition{
	Type:    "VlanAssigned",
	Status:  "True",
	Reason:  "VlanCRCreated",
	Message: "VLAN VID assigned and Vlan CR created",
}

var ConditionVlanAssignedFalse = metav1.Condition{
	Type:    "VlanAssigned",
	Status:  "False",
	Reason:  "VlanCRNotCreated",
	Message: "Failed to assign VID or create Vlan CR",
}
