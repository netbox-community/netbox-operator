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

// VlanSpec defines the desired state of Vlan
type VlanSpec struct {
	// The unique VLAN ID (VID)
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=4094
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'vlanId' is immutable"
	VlanId int `json:"vlanId"`

	// The desired name for the VLAN in NetBox
	//+kubebuilder:validation:Required
	Name string `json:"name"`

	// The NetBox Site where this VLAN should exist
	//+kubebuilder:validation:Required
	Site string `json:"site"`

	// The NetBox VLANGroup where this VLAN should be organized
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

// VlanStatus defines the observed state of Vlan
type VlanStatus struct {
	// The NetBox internal database ID of the created/managed VLAN
	//+optional
	VlanId int64 `json:"id,omitempty"`

	// The URL to the VLAN object in the NetBox UI
	//+optional
	VlanUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
	//+optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="VLAN ID",type=integer,JSONPath=`.spec.vlanId`
//+kubebuilder:printcolumn:name="NetBox ID",type=integer,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=vl

// Vlan is the Schema for the vlans API
type Vlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VlanSpec   `json:"spec,omitempty"`
	Status VlanStatus `json:"status,omitempty"`
}

func (v *Vlan) Conditions() *[]metav1.Condition {
	return &v.Status.Conditions
}

//+kubebuilder:object:root=true

// VlanList contains a list of Vlan
type VlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Vlan{}, &VlanList{})
}

var ConditionVlanReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "VlanSynchronized",
	Message: "VLAN was synchronized with NetBox",
}

var ConditionVlanReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "VlanSyncFailed",
	Message: "Failed to synchronize VLAN with NetBox",
}
