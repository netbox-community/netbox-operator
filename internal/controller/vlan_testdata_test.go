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
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// -----------------------------
// default values for Vlan/VlanClaim CRs
//
// Tenant and Site are intentionally left unset on every fixture below:
// setting them would route ReserveOrUpdateVlan/GetAvailableVlanByClaim
// through getTenantDetails/getSiteDetails (clientV3.Tenancy/Dcim), which
// these controller-level tests don't wire up. Tenant/Site resolution is
// already covered at the client level by pkg/netbox/api/vlan_test.go.
// -----------------------------

var vlanName = "vlan-test"
var vlanClaimName = "vlanclaim-test"
var vlanNamespace = "default"
var vlanVid = int32(100)
var vlanRestoredVid = int32(150)
var vlanRangeStart = int32(200)
var vlanRangeEnd = int32(202)
var vlanComments = "vlan integration test comment"
var vlanDescription = "vlan integration test"
var vlanCustomFields = map[string]string{"example_field": "example value"}
var vlanId = int32(42)
var vlanLastUpdated = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

var vlanRestorationHash = "b3e6e6c1a7d4e2f9c1a1b2c3d4e5f6a7b8c9d0e1"
var vlanCustomFieldsWithHash = map[string]string{"example_field": "example value", "netboxOperatorRestorationHash": vlanRestorationHash}
var vlanCustomFieldsWithHashMismatchNetboxFmt = map[string]interface{}{"example_field": "example value", "netboxOperatorRestorationHash": "a-different-hash"}

// -----------------------------
// default CRs
// -----------------------------

func defaultVlanCR(preserveInNetbox bool) *netboxv1.Vlan {
	return &netboxv1.Vlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vlanName,
			Namespace: vlanNamespace,
		},
		Spec: netboxv1.VlanSpec{
			Vid:              vlanVid,
			CustomFields:     vlanCustomFields,
			Comments:         vlanComments,
			Description:      vlanDescription,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

// defaultVlanCreatedByClaim mimics a Vlan CR that a VlanClaim controller
// would have created: its CustomFields carry the restoration hash key, as
// generateVlanSpec injects on every child Vlan CR.
func defaultVlanCreatedByClaim(preserveInNetbox bool) *netboxv1.Vlan {
	return &netboxv1.Vlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vlanName,
			Namespace: vlanNamespace,
		},
		Spec: netboxv1.VlanSpec{
			Vid:              vlanVid,
			CustomFields:     vlanCustomFieldsWithHash,
			Comments:         vlanComments,
			Description:      vlanDescription,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

func defaultVlanClaimCRWithVid() *netboxv1.VlanClaim {
	return &netboxv1.VlanClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vlanClaimName,
			Namespace: vlanNamespace,
		},
		Spec: netboxv1.VlanClaimSpec{
			Vid:              vlanVid,
			CustomFields:     vlanCustomFields,
			Comments:         vlanComments,
			Description:      vlanDescription,
			PreserveInNetbox: false,
		},
	}
}

func defaultVlanClaimCRWithRange() *netboxv1.VlanClaim {
	return &netboxv1.VlanClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vlanClaimName,
			Namespace: vlanNamespace,
		},
		Spec: netboxv1.VlanClaimSpec{
			VidRangeStart:    vlanRangeStart,
			VidRangeEnd:      vlanRangeEnd,
			CustomFields:     vlanCustomFields,
			Comments:         vlanComments,
			Description:      vlanDescription,
			PreserveInNetbox: false,
		},
	}
}

// expectedVlanSpecFromClaim mirrors generateVlanSpec: the VlanClaim
// controller copies the claim's mutable fields onto the child Vlan CR and
// injects the restoration hash custom field.
func expectedVlanSpecFromClaim(claim *netboxv1.VlanClaim, vid int32) netboxv1.VlanSpec {
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}
	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateVlanRestorationHash(claim)

	return netboxv1.VlanSpec{
		Vid:              vid,
		Site:             claim.Spec.Site,
		Tenant:           claim.Spec.Tenant,
		Status:           claim.Spec.Status,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

// -----------------------------
// netbox mock responses
// -----------------------------

func mockedVlanResponse() *v4client.VLAN {
	lastUpdated := vlanLastUpdated
	resp := &v4client.VLAN{
		Id:   vlanId,
		Name: vlanName,
		Vid:  vlanVid,
	}
	resp.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
	return resp
}

func mockedVlanListEmpty() *v4client.PaginatedVLANList {
	return &v4client.PaginatedVLANList{Count: 0, Results: []v4client.VLAN{}}
}

// mockedVlanListExisting returns a single-page listing containing one Vlan
// matching vlanVid, as found by VlanReconciler.getVlan (Vid filter).
// customFields is nil for a plain pre-existing NetBox object, or set with the
// restoration hash key for objects claimed by a VlanClaim.
func mockedVlanListExisting(customFields map[string]interface{}) *v4client.PaginatedVLANList {
	lastUpdated := vlanLastUpdated
	existing := v4client.VLAN{
		Id:           vlanId,
		Name:         vlanName,
		Vid:          vlanVid,
		CustomFields: customFields,
	}
	existing.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
	return &v4client.PaginatedVLANList{Count: 1, Results: []v4client.VLAN{existing}}
}

// mockedVlanListWithHash returns a single-page listing (as scanned by
// forEachVlan) containing one Vlan carrying the given restoration hash, as
// used by RestoreExistingVlanByHash.
func mockedVlanListWithHash(hash string, vid int32) *v4client.PaginatedVLANList {
	vlan := v4client.VLAN{
		Name: vlanName,
		Vid:  vid,
		CustomFields: map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: hash,
		},
	}
	return &v4client.PaginatedVLANList{Count: 1, Results: []v4client.VLAN{vlan}}
}

var ExpectedVlanStatus = netboxv1.VlanStatus{
	VlanId: int64(vlanId),
}
