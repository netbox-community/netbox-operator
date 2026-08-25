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
// default values for L2VPN/L2VPNClaim CRs
//
// Tenant is intentionally left unset on every fixture below: setting it would
// route ReserveOrUpdateL2VPN/GetAvailableL2VPNIdentifierByClaim through
// getTenantDetails (clientV3.Tenancy), which these controller-level tests
// don't wire up. Tenant resolution is already covered at the client level by
// pkg/netbox/api/l2vpn_test.go.
// -----------------------------

var l2vpnName = "l2vpn-test"
var l2vpnClaimName = "l2vpnclaim-test"
var l2vpnNamespace = "default"
var l2vpnType = "vxlan-evpn"
var l2vpnIdentifier = int64(5000100)
var l2vpnRestoredIdentifier = int64(5000150)
var l2vpnRangeStart = int64(5000200)
var l2vpnRangeEnd = int64(5000202)
var l2vpnComments = "l2vpn integration test comment"
var l2vpnDescription = "l2vpn integration test"
var l2vpnCustomFields = map[string]string{"example_field": "example value"}
var l2vpnId = int32(42)
var l2vpnSlug = "l2vpn-test-slug"
var l2vpnLastUpdated = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

var l2vpnRestorationHash = "6f6c67651f0b43b2969ba2ae35c74fc91815513a"
var l2vpnCustomFieldsWithHash = map[string]string{"example_field": "example value", "netboxOperatorRestorationHash": l2vpnRestorationHash}
var l2vpnCustomFieldsWithHashMismatchNetboxFmt = map[string]interface{}{"example_field": "example value", "netboxOperatorRestorationHash": "a-different-hash"}

// -----------------------------
// default CRs
// -----------------------------

func defaultL2VPNCR(preserveInNetbox bool) *netboxv1.L2VPN {
	return &netboxv1.L2VPN{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l2vpnName,
			Namespace: l2vpnNamespace,
		},
		Spec: netboxv1.L2VPNSpec{
			Name:             l2vpnName,
			Type:             l2vpnType,
			Identifier:       l2vpnIdentifier,
			CustomFields:     l2vpnCustomFields,
			Comments:         l2vpnComments,
			Description:      l2vpnDescription,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

// defaultL2VPNCreatedByClaim mimics a L2VPN CR that a L2VPNClaim controller
// would have created: its CustomFields carry the restoration hash key, as
// generateL2VPNSpec injects on every child L2VPN CR.
func defaultL2VPNCreatedByClaim(preserveInNetbox bool) *netboxv1.L2VPN {
	return &netboxv1.L2VPN{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l2vpnName,
			Namespace: l2vpnNamespace,
		},
		Spec: netboxv1.L2VPNSpec{
			Name:             l2vpnName,
			Type:             l2vpnType,
			Identifier:       l2vpnIdentifier,
			CustomFields:     l2vpnCustomFieldsWithHash,
			Comments:         l2vpnComments,
			Description:      l2vpnDescription,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

func defaultL2VPNClaimCRWithIdentifier() *netboxv1.L2VPNClaim {
	return &netboxv1.L2VPNClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l2vpnClaimName,
			Namespace: l2vpnNamespace,
		},
		Spec: netboxv1.L2VPNClaimSpec{
			Type:             l2vpnType,
			Identifier:       l2vpnIdentifier,
			CustomFields:     l2vpnCustomFields,
			Comments:         l2vpnComments,
			Description:      l2vpnDescription,
			PreserveInNetbox: false,
		},
	}
}

func defaultL2VPNClaimCRWithRange() *netboxv1.L2VPNClaim {
	return &netboxv1.L2VPNClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l2vpnClaimName,
			Namespace: l2vpnNamespace,
		},
		Spec: netboxv1.L2VPNClaimSpec{
			Type:                 l2vpnType,
			IdentifierRangeStart: l2vpnRangeStart,
			IdentifierRangeEnd:   l2vpnRangeEnd,
			CustomFields:         l2vpnCustomFields,
			Comments:             l2vpnComments,
			Description:          l2vpnDescription,
			PreserveInNetbox:     false,
		},
	}
}

// expectedL2VPNSpecFromClaim mirrors generateL2VPNSpec: the L2VPNClaim
// controller copies the claim's mutable fields onto the child L2VPN CR and
// injects the restoration hash custom field.
func expectedL2VPNSpecFromClaim(claim *netboxv1.L2VPNClaim, identifier int64) netboxv1.L2VPNSpec {
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}
	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateL2VPNRestorationHash(claim)

	return netboxv1.L2VPNSpec{
		Name:             claim.Name,
		Type:             claim.Spec.Type,
		Identifier:       identifier,
		Tenant:           claim.Spec.Tenant,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

// -----------------------------
// netbox mock responses
// -----------------------------

func mockedL2VPNResponse() *v4client.L2VPN {
	lastUpdated := l2vpnLastUpdated
	resp := &v4client.L2VPN{
		Id:   l2vpnId,
		Name: l2vpnName,
		Slug: l2vpnSlug,
	}
	resp.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
	resp.Identifier = *v4client.NewNullableInt64(&l2vpnIdentifier)
	return resp
}

func mockedL2VPNListEmpty() *v4client.PaginatedL2VPNList {
	return &v4client.PaginatedL2VPNList{Count: 0, Results: []v4client.L2VPN{}}
}

// mockedL2VPNListExisting returns a single-page listing containing one L2VPN
// matching l2vpnIdentifier, as found by L2VPNReconciler.getL2VPN (Identifier
// filter). customFields is nil for a plain pre-existing NetBox object, or set
// with the restoration hash key for objects claimed by a L2VPNClaim.
func mockedL2VPNListExisting(customFields map[string]interface{}) *v4client.PaginatedL2VPNList {
	lastUpdated := l2vpnLastUpdated
	existing := v4client.L2VPN{
		Id:           l2vpnId,
		Name:         l2vpnName,
		Slug:         l2vpnSlug,
		CustomFields: customFields,
	}
	existing.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
	existing.Identifier = *v4client.NewNullableInt64(&l2vpnIdentifier)
	return &v4client.PaginatedL2VPNList{Count: 1, Results: []v4client.L2VPN{existing}}
}

// mockedL2VPNListWithHash returns a single-page listing (as scanned by
// forEachL2VPN) containing one L2VPN carrying the given restoration hash, as
// used by RestoreExistingL2VPNByHash.
func mockedL2VPNListWithHash(hash string, identifier int64) *v4client.PaginatedL2VPNList {
	l2vpn := v4client.L2VPN{
		Name: l2vpnName,
		CustomFields: map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: hash,
		},
	}
	l2vpn.Identifier = *v4client.NewNullableInt64(&identifier)
	return &v4client.PaginatedL2VPNList{Count: 1, Results: []v4client.L2VPN{l2vpn}}
}

var ExpectedL2VPNStatus = netboxv1.L2VPNStatus{
	L2VPNId: int64(l2vpnId),
	Slug:    l2vpnSlug,
}
