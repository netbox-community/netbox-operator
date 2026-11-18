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

// RouteTargetClaimSpec defines the desired state of RouteTargetClaim
// +kubebuilder:validation:XValidation:rule="self.assignedNumberRangeStart <= self.assignedNumberRangeEnd",message="assignedNumberRangeStart must be less than or equal to assignedNumberRangeEnd"
// +kubebuilder:validation:XValidation:rule="self.asn <= 65535 || self.assignedNumberRangeEnd <= 65535",message="assigned numbers for 4-byte ASNs must be in the range 0-65535"
type RouteTargetClaimSpec struct {
	// The ASN (administrator subfield) of the route target. IP-address-based
	// route targets are not supported.
	// Field is immutable, required, range from 0-4294967295
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=4294967295
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'asn' is immutable"
	Asn int64 `json:"asn"`

	// The lower bound (inclusive) of the range to pick a free assigned number
	// from. Set both this and assignedNumberRangeEnd to the same value to
	// claim an exact route target.
	// Field is immutable, required, range from 0-4294967295 for 2-byte ASNs
	// and 0-65535 for 4-byte ASNs
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=4294967295
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'assignedNumberRangeStart' is immutable"
	AssignedNumberRangeStart int64 `json:"assignedNumberRangeStart"`

	// The upper bound (inclusive) of the range to pick a free assigned number
	// from. Set both this and assignedNumberRangeStart to the same value to
	// claim an exact route target.
	// Field is immutable, required, range from 0-4294967295 for 2-byte ASNs
	// and 0-65535 for 4-byte ASNs
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=4294967295
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="Field 'assignedNumberRangeEnd' is immutable"
	AssignedNumberRangeEnd int64 `json:"assignedNumberRangeEnd"`

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

// RouteTargetClaimStatus defines the observed state of RouteTargetClaim
type RouteTargetClaimStatus struct {
	// The assigned route target value, e.g. "65000:100"
	RouteTarget string `json:"routeTarget,omitempty"`

	// The name of the RouteTarget CR created by the RouteTargetClaim Controller
	RouteTargetName string `json:"routeTargetName,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="RouteTarget",type=string,JSONPath=`.status.routeTarget`
//+kubebuilder:printcolumn:name="RouteTargetAssigned",type=string,JSONPath=`.status.conditions[?(@.type=="RouteTargetAssigned")].status`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=rtc

// RouteTargetClaim allows to claim a NetBox Route Target, either an exact one
// or the next free one for an ASN within a range of assigned numbers. The
// RouteTargetClaim Controller will try to assign an available route target
// and if successful it will create the RouteTarget CR.
// More info about NetBox Route Targets: https://github.com/netbox-community/netbox/blob/main/docs/models/ipam/routetarget.md
type RouteTargetClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RouteTargetClaimSpec   `json:"spec,omitempty"`
	Status RouteTargetClaimStatus `json:"status,omitempty"`
}

func (r *RouteTargetClaim) Conditions() *[]metav1.Condition {
	return &r.Status.Conditions
}

//+kubebuilder:object:root=true

// RouteTargetClaimList contains a list of RouteTargetClaim
type RouteTargetClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RouteTargetClaim `json:"items"`
}

func init() {
	register(&RouteTargetClaim{}, &RouteTargetClaimList{})
}

var ConditionRouteTargetClaimReadyTrue = metav1.Condition{
	Type:    "Ready",
	Status:  "True",
	Reason:  "RouteTargetResourceReady",
	Message: "Route Target Resource is ready",
}

var ConditionRouteTargetClaimReadyFalse = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "RouteTargetResourceNotReady",
	Message: "Route Target Resource is not ready",
}

var ConditionRouteTargetAssignedTrue = metav1.Condition{
	Type:    "RouteTargetAssigned",
	Status:  "True",
	Reason:  "RouteTargetCRCreated",
	Message: "New route target fetched from NetBox and RouteTarget CR was created",
}

var ConditionRouteTargetAssignedFalse = metav1.Condition{
	Type:    "RouteTargetAssigned",
	Status:  "False",
	Reason:  "RouteTargetCRNotCreated",
	Message: "Failed to fetch new route target from NetBox",
}
