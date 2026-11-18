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

package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func routeTargetWithName(name string) v4client.RouteTarget {
	return v4client.RouteTarget{Name: name}
}

// expectRouteTargetPage sets up the IpamRouteTargetsList call chain issued by
// forEachRouteTargetOfAsn for a single page at offset.
func expectRouteTargetPage(ctrl *gomock.Controller, mockIpamAPI *mock_interfaces.MockIpamAPI, asnPrefix string, offset int32, results []v4client.RouteTarget) {
	mockListRequest := mock_interfaces.NewMockIpamRouteTargetsListRequest(ctrl)
	mockIpamAPI.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockListRequest)
	mockListRequest.EXPECT().NameIsw([]string{asnPrefix}).Return(mockListRequest)
	mockListRequest.EXPECT().Limit(routeTargetListPageSize).Return(mockListRequest)
	mockListRequest.EXPECT().Offset(offset).Return(mockListRequest)
	mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedRouteTargetList{
		Count:   int32(len(results)),
		Results: results,
	}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)
}

func TestRouteTargetName(t *testing.T) {
	assert.Equal(t, "65000:100", RouteTargetName(65000, 100))
	assert.Equal(t, "4200000000:65535", RouteTargetName(4200000000, 65535))
}

func TestRestoreExistingRouteTargetByHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("found", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		match := routeTargetWithName("65000:123")
		match.Id = 5
		match.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "matching-hash"}

		other := routeTargetWithName("65000:1")
		other.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "other-hash"}

		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", 0, []v4client.RouteTarget{other, match})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingRouteTargetByHash(context.TODO(), 65000, "matching-hash")
		AssertNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "65000:123", result.Name)
		assert.Equal(t, int64(5), result.Id)
	})

	t.Run("not found", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", 0, []v4client.RouteTarget{})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingRouteTargetByHash(context.TODO(), 65000, "no-such-hash")
		AssertNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("paginates across multiple pages", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		page1 := make([]v4client.RouteTarget, routeTargetListPageSize)
		for i := range page1 {
			page1[i] = routeTargetWithName(RouteTargetName(65000, int64(i)))
		}

		match := routeTargetWithName("65000:9999")
		match.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "page2-hash"}

		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", 0, page1)
		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", routeTargetListPageSize, []v4client.RouteTarget{match})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingRouteTargetByHash(context.TODO(), 65000, "page2-hash")
		AssertNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "65000:9999", result.Name)
	})

	t.Run("list error is returned", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamRouteTargetsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().NameIsw(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(nil, &http.Response{StatusCode: 500, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingRouteTargetByHash(context.TODO(), 65000, "any-hash")
		AssertError(t, err, "failed to list route targets: status 500")
		assert.Nil(t, result)
	})
}

func TestGetAvailableRouteTargetByClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("empty range returns start", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", 0, []v4client.RouteTarget{})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableRouteTargetByClaim(context.TODO(), &models.RouteTargetClaim{
			Asn:                      65000,
			AssignedNumberRangeStart: 100,
			AssignedNumberRangeEnd:   199,
		})
		AssertNil(t, err)
		assert.Equal(t, "65000:100", result.Name)
	})

	t.Run("finds gap after contiguous block from start", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", 0, []v4client.RouteTarget{
			routeTargetWithName("65000:100"),
			routeTargetWithName("65000:102"),
			routeTargetWithName("65000:101"),
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableRouteTargetByClaim(context.TODO(), &models.RouteTargetClaim{
			Asn:                      65000,
			AssignedNumberRangeStart: 100,
			AssignedNumberRangeEnd:   199,
		})
		AssertNil(t, err)
		assert.Equal(t, "65000:103", result.Name)
	})

	t.Run("ignores assigned numbers outside the requested range and malformed names", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", 0, []v4client.RouteTarget{
			routeTargetWithName("65000:99"),
			routeTargetWithName("65000:200"),
			routeTargetWithName("65000:abc"),
			routeTargetWithName("65000:100:1"),
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableRouteTargetByClaim(context.TODO(), &models.RouteTargetClaim{
			Asn:                      65000,
			AssignedNumberRangeStart: 100,
			AssignedNumberRangeEnd:   199,
		})
		AssertNil(t, err)
		assert.Equal(t, "65000:100", result.Name)
	})

	t.Run("exact route target available", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectRouteTargetPage(ctrl, mockIpamAPI, "4200000000:", 0, []v4client.RouteTarget{
			routeTargetWithName("4200000000:1"),
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableRouteTargetByClaim(context.TODO(), &models.RouteTargetClaim{
			Asn:                      4200000000,
			AssignedNumberRangeStart: 2,
			AssignedNumberRangeEnd:   2,
		})
		AssertNil(t, err)
		assert.Equal(t, "4200000000:2", result.Name)
	})

	t.Run("returns exhausted error when range is full", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectRouteTargetPage(ctrl, mockIpamAPI, "65000:", 0, []v4client.RouteTarget{
			routeTargetWithName("65000:100"),
			routeTargetWithName("65000:101"),
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableRouteTargetByClaim(context.TODO(), &models.RouteTargetClaim{
			Asn:                      65000,
			AssignedNumberRangeStart: 100,
			AssignedNumberRangeEnd:   101,
		})
		assert.ErrorIs(t, err, ErrRouteTargetRangeExhausted)
		assert.Nil(t, result)
	})

	t.Run("returns error for non-existing tenant without listing route targets", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

		tenantName := "nonexistingtenant"
		tenancyListInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(&tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{Results: []*netboxModels.Tenant{}},
		}, nil)

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy},
		}

		result, err := compositeClient.GetAvailableRouteTargetByClaim(context.TODO(), &models.RouteTargetClaim{
			Asn:                      65000,
			AssignedNumberRangeStart: 100,
			AssignedNumberRangeEnd:   101,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})
		AssertError(t, err, "failed to fetch tenant 'nonexistingtenant': not found")
		assert.Nil(t, result)
	})
}
