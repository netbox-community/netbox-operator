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

// RouteTargetSpec defines the desired state of RouteTarget
type RouteTargetSpec struct {
	// The route target value in ASN:number notation (RFC 4360). With a 2-byte
	// ASN (0-65535) the assigned number can range from 0-4294967295, with a
	// 4-byte ASN (65536-4294967295) it can range from 0-65535.
	// IP-address-based route targets are not supported.
	// Field is immutable, required
	// Example: "65000:100"
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MaxLength=21
	//+kubebuilder:validation:Pattern=`^[0-9]{1,10}:[0-9]{1,10}$`
	//+kubebuilder:validation:XValidation:rule="!self.matches('^[0-9]{1,10}:[0-9]{1,10}$') || int(self.split(':')[0]) <= 4294967295",message="ASN must be in the range 0-4294967295"
	//+kubebuilder:validation:XValidation:rule="!self.matches('^[0-9]{1,10}:[0-9]{1,10}$') || int(self.split(':')[1]) <= (int(self.split(':')[0]) <= 65535 ? 4294967295 : 65535)",message="assigned number must be in the range 0-4294967295 for 2-byte ASNs and 0-65535 for 4-byte ASNs"
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'routeTarget' is immutable"
	RouteTarget string `json:"routeTarget"`

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
	// resources from NetBox (e.g. Sticky route targets even if resources are
	// deleted and recreated in Kubernetes)
	// Field is mutable, not required
	PreserveInNetbox bool `json:"preserveInNetbox,omitempty"`
}

// RouteTargetStatus defines the observed state of RouteTarget
type RouteTargetStatus struct {
	// The ID of the resource in NetBox
	RouteTargetId int64 `json:"id,omitempty"`

	// Last updated, corresponds to the 'last_updated' returned by NetBox when NetBox Operator updates a resource in NetBox.
	// Format: date-time
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// The URL to the resource in the NetBox UI. Note that the base of this
	// URL depends on the runtime config of NetBox Operator
	RouteTargetUrl string `json:"url,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="RouteTarget",type=string,JSONPath=`.spec.routeTarget`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=rt

// RouteTarget allows to create a NetBox Route Target. The route target value
// (e.g. "65000:100") is set in spec.routeTarget and used as the Route
// Target's name in NetBox. More info about NetBox Route Targets: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/routetarget.md
type RouteTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RouteTargetSpec   `json:"spec,omitempty"`
	Status RouteTargetStatus `json:"status,omitempty"`
}

func (r *RouteTarget) Conditions() *[]metav1.Condition {
	return &r.Status.Conditions
}

//+kubebuilder:object:root=true

// RouteTargetList contains a list of RouteTarget
type RouteTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RouteTarget `json:"items"`
}

func init() {
	register(&RouteTarget{}, &RouteTargetList{})
}

var ConditionRouteTargetReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "RouteTargetReservedInNetbox",
	Message: "Route Target was reserved/updated in NetBox",
}

var ConditionRouteTargetReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToReserveRouteTargetInNetbox",
	Message: "Failed to reserve Route Target in NetBox",
}

var ConditionRouteTargetReadyFalseDeletionInProgress = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "DeletionInProgress",
	Message: "Route Target deletion in progress",
}

var ConditionRouteTargetReadyFalseDeletionFailed = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "FailedToDeleteRouteTargetInNetbox",
	Message: "Failed to delete Route Target in NetBox",
}
