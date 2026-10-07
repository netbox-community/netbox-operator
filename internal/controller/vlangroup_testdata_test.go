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

package controller

import (
	"time"

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// -----------------------------
// default values for VlanGroup CRs
//
// Tenant and Site are intentionally left unset on every fixture below:
// setting them would route ReserveOrUpdateVlanGroup through
// getTenantDetails/getSiteDetails (clientV3.Tenancy/Dcim), which these
// controller-level tests don't wire up. Tenant/Site resolution is already
// covered at the client level by pkg/netbox/api/vlan_group_test.go.
// -----------------------------

var vlanGroupName = "vlangroup-test"
var vlanGroupNamespace = "default"
var vlanGroupDescription = "vlan group integration test"
var vlanGroupCustomFields = map[string]string{"example_field": "example value"}
var vlanGroupId = int32(77)
var vlanGroupSlug = "vlangroup-test"
var vlanGroupLastUpdated = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// -----------------------------
// default CRs
// -----------------------------

func defaultVlanGroupCR(preserveInNetbox bool) *netboxv1.VlanGroup {
	return &netboxv1.VlanGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vlanGroupName,
			Namespace: vlanGroupNamespace,
		},
		Spec: netboxv1.VlanGroupSpec{
			CustomFields:     vlanGroupCustomFields,
			Description:      vlanGroupDescription,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

// -----------------------------
// netbox mock responses
// -----------------------------

func mockedVlanGroupResponse() *v4client.VLANGroup {
	lastUpdated := vlanGroupLastUpdated
	resp := &v4client.VLANGroup{
		Id:   vlanGroupId,
		Name: vlanGroupName,
		Slug: vlanGroupSlug,
	}
	resp.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
	return resp
}

func mockedVlanGroupListEmpty() *v4client.PaginatedVLANGroupList {
	return &v4client.PaginatedVLANGroupList{Count: 0, Results: []v4client.VLANGroup{}}
}

// mockedVlanGroupListExisting returns a single-page listing containing one
// VlanGroup matching vlanGroupName, as found by
// VlanGroupReconciler.getVlanGroup (Name filter).
func mockedVlanGroupListExisting() *v4client.PaginatedVLANGroupList {
	lastUpdated := vlanGroupLastUpdated
	existing := v4client.VLANGroup{
		Id:   vlanGroupId,
		Name: vlanGroupName,
		Slug: vlanGroupSlug,
	}
	existing.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
	return &v4client.PaginatedVLANGroupList{Count: 1, Results: []v4client.VLANGroup{existing}}
}

var ExpectedVlanGroupStatus = netboxv1.VlanGroupStatus{
	VlanGroupId: int64(vlanGroupId),
	Slug:        vlanGroupSlug,
}
