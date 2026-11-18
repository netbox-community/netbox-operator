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
// default values for RouteTarget/RouteTargetClaim CRs
//
// Tenant is intentionally left unset on every fixture below: setting it would
// route ReserveOrUpdateRouteTarget/GetAvailableRouteTargetByClaim through
// getTenantDetails (clientV3.Tenancy), which these controller-level tests
// don't wire up. Tenant resolution is already covered at the client level by
// pkg/netbox/api/route_target_test.go.
// -----------------------------

var routeTargetName = "routetarget-test"
var routeTargetClaimName = "routetargetclaim-test"
var routeTargetNamespace = "default"
var routeTargetAsn = int64(65000)
var routeTargetValue = "65000:100"
var routeTargetAssignedNumber = int64(100)
var routeTargetRestoredValue = "65000:150"
var routeTargetRangeStart = int64(200)
var routeTargetRangeEnd = int64(202)
var routeTargetComments = "route target integration test comment"
var routeTargetDescription = "route target integration test"
var routeTargetCustomFields = map[string]string{"example_field": "example value"}
var routeTargetId = int32(43)
var routeTargetLastUpdated = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

var routeTargetCustomFieldsWithHash = map[string]string{"example_field": "example value", "netboxOperatorRestorationHash": "some-hash"}
var routeTargetCustomFieldsWithHashMismatchNetboxFmt = map[string]interface{}{"example_field": "example value", "netboxOperatorRestorationHash": "a-different-hash"}

// -----------------------------
// default CRs
// -----------------------------

func defaultRouteTargetCR(preserveInNetbox bool) *netboxv1.RouteTarget {
	return &netboxv1.RouteTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeTargetName,
			Namespace: routeTargetNamespace,
		},
		Spec: netboxv1.RouteTargetSpec{
			RouteTarget:      routeTargetValue,
			CustomFields:     routeTargetCustomFields,
			Comments:         routeTargetComments,
			Description:      routeTargetDescription,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

// defaultRouteTargetCreatedByClaim mimics a RouteTarget CR that a
// RouteTargetClaim controller would have created: its CustomFields carry the
// restoration hash key, as generateRouteTargetSpec injects on every child
// RouteTarget CR.
func defaultRouteTargetCreatedByClaim(preserveInNetbox bool) *netboxv1.RouteTarget {
	cr := defaultRouteTargetCR(preserveInNetbox)
	cr.Spec.CustomFields = routeTargetCustomFieldsWithHash
	return cr
}

// defaultRouteTargetClaimCRExact returns a claim for a single exact route
// target, expressed as a range of one (assignedNumberRangeStart ==
// assignedNumberRangeEnd).
func defaultRouteTargetClaimCRExact() *netboxv1.RouteTargetClaim {
	return &netboxv1.RouteTargetClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeTargetClaimName,
			Namespace: routeTargetNamespace,
		},
		Spec: netboxv1.RouteTargetClaimSpec{
			Asn:                      routeTargetAsn,
			AssignedNumberRangeStart: routeTargetAssignedNumber,
			AssignedNumberRangeEnd:   routeTargetAssignedNumber,
			CustomFields:             routeTargetCustomFields,
			Comments:                 routeTargetComments,
			Description:              routeTargetDescription,
			PreserveInNetbox:         false,
		},
	}
}

func defaultRouteTargetClaimCRWithRange() *netboxv1.RouteTargetClaim {
	return &netboxv1.RouteTargetClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeTargetClaimName,
			Namespace: routeTargetNamespace,
		},
		Spec: netboxv1.RouteTargetClaimSpec{
			Asn:                      routeTargetAsn,
			AssignedNumberRangeStart: routeTargetRangeStart,
			AssignedNumberRangeEnd:   routeTargetRangeEnd,
			CustomFields:             routeTargetCustomFields,
			Comments:                 routeTargetComments,
			Description:              routeTargetDescription,
			PreserveInNetbox:         false,
		},
	}
}

// expectedRouteTargetSpecFromClaim mirrors generateRouteTargetSpec: the
// RouteTargetClaim controller copies the claim's mutable fields onto the
// child RouteTarget CR and injects the restoration hash custom field.
func expectedRouteTargetSpecFromClaim(claim *netboxv1.RouteTargetClaim, routeTarget string) netboxv1.RouteTargetSpec {
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}
	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateRouteTargetRestorationHash(claim)

	return netboxv1.RouteTargetSpec{
		RouteTarget:      routeTarget,
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

func mockedRouteTarget(name string, customFields map[string]interface{}) v4client.RouteTarget {
	lastUpdated := routeTargetLastUpdated
	rt := v4client.RouteTarget{
		Id:           routeTargetId,
		Name:         name,
		CustomFields: customFields,
	}
	rt.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
	return rt
}

func mockedRouteTargetResponse() *v4client.RouteTarget {
	rt := mockedRouteTarget(routeTargetValue, nil)
	return &rt
}

func mockedRouteTargetListEmpty() *v4client.PaginatedRouteTargetList {
	return &v4client.PaginatedRouteTargetList{Count: 0, Results: []v4client.RouteTarget{}}
}

// mockedRouteTargetListExisting returns a single-page listing containing one
// Route Target named routeTargetValue, as found by
// RouteTargetReconciler.getRouteTarget (Name filter). customFields is nil for
// a plain pre-existing NetBox object, or set with the restoration hash key
// for objects claimed by a RouteTargetClaim.
func mockedRouteTargetListExisting(customFields map[string]interface{}) *v4client.PaginatedRouteTargetList {
	return &v4client.PaginatedRouteTargetList{Count: 1, Results: []v4client.RouteTarget{mockedRouteTarget(routeTargetValue, customFields)}}
}

// mockedRouteTargetListWithHash returns a single-page listing (as scanned by
// forEachRouteTargetOfAsn) containing one Route Target carrying the given
// restoration hash, as used by RestoreExistingRouteTargetByHash.
func mockedRouteTargetListWithHash(hash string, name string) *v4client.PaginatedRouteTargetList {
	rt := mockedRouteTarget(name, map[string]interface{}{
		config.GetOperatorConfig().NetboxRestorationHashFieldName: hash,
	})
	return &v4client.PaginatedRouteTargetList{Count: 1, Results: []v4client.RouteTarget{rt}}
}

var ExpectedRouteTargetStatus = netboxv1.RouteTargetStatus{
	RouteTargetId: int64(routeTargetId),
}
