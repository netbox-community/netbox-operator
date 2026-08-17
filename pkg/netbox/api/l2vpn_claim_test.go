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

func l2vpnWithIdentifier(id int64) v4client.L2VPN {
	l2vpn := v4client.L2VPN{}
	l2vpn.Identifier = *v4client.NewNullableInt64(&id)
	return l2vpn
}

func TestRestoreExistingL2VPNByHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("found", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		identifier := int64(5000123)
		match := l2vpnWithIdentifier(identifier)
		match.Name = "l2vpnclaim-sample"
		match.Slug = "l2vpnclaim-sample"
		match.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "matching-hash"}

		other := l2vpnWithIdentifier(4001)
		other.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "other-hash"}

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count:   2,
			Results: []v4client.L2VPN{other, match},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		result, err := compositeClient.RestoreExistingL2VPNByHash(context.TODO(), "matching-hash")
		AssertNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, identifier, result.Identifier)
		assert.Equal(t, "l2vpnclaim-sample", result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count:   0,
			Results: []v4client.L2VPN{},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		result, err := compositeClient.RestoreExistingL2VPNByHash(context.TODO(), "no-such-hash")
		AssertNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("paginates across multiple pages", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequestPage1 := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)
		mockListRequestPage2 := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		match := l2vpnWithIdentifier(9999)
		match.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "page2-hash"}

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().Offset(int32(0)).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count:   2,
			Results: []v4client.L2VPN{l2vpnWithIdentifier(1)},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().Offset(int32(1)).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count:   2,
			Results: []v4client.L2VPN{match},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		result, err := compositeClient.RestoreExistingL2VPNByHash(context.TODO(), "page2-hash")
		AssertNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(9999), result.Identifier)
	})
}

func TestGetAvailableL2VPNIdentifierByClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("empty range returns start", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count:   0,
			Results: []v4client.L2VPN{},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		result, err := compositeClient.GetAvailableL2VPNIdentifierByClaim(context.TODO(), &models.L2VPNClaim{
			Type:                 "vxlan-evpn",
			IdentifierRangeStart: 4000,
			IdentifierRangeEnd:   16777215,
		})
		AssertNil(t, err)
		assert.Equal(t, int64(4000), result.Identifier)
	})

	t.Run("finds gap after contiguous block from start", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count: 3,
			Results: []v4client.L2VPN{
				l2vpnWithIdentifier(4000),
				l2vpnWithIdentifier(4002),
				l2vpnWithIdentifier(4001),
			},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		result, err := compositeClient.GetAvailableL2VPNIdentifierByClaim(context.TODO(), &models.L2VPNClaim{
			Type:                 "vxlan-evpn",
			IdentifierRangeStart: 4000,
			IdentifierRangeEnd:   16777215,
		})
		AssertNil(t, err)
		assert.Equal(t, int64(4003), result.Identifier)
	})

	t.Run("ignores identifiers outside the requested range", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count: 1,
			Results: []v4client.L2VPN{
				l2vpnWithIdentifier(9999999),
			},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		result, err := compositeClient.GetAvailableL2VPNIdentifierByClaim(context.TODO(), &models.L2VPNClaim{
			Type:                 "vxlan-evpn",
			IdentifierRangeStart: 4000,
			IdentifierRangeEnd:   4001,
		})
		AssertNil(t, err)
		assert.Equal(t, int64(4000), result.Identifier)
	})

	t.Run("returns exhausted error when range is full", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(l2vpnListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{
			Count: 2,
			Results: []v4client.L2VPN{
				l2vpnWithIdentifier(4000),
				l2vpnWithIdentifier(4001),
			},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		result, err := compositeClient.GetAvailableL2VPNIdentifierByClaim(context.TODO(), &models.L2VPNClaim{
			Type:                 "vxlan-evpn",
			IdentifierRangeStart: 4000,
			IdentifierRangeEnd:   4001,
		})
		assert.ErrorIs(t, err, ErrL2VPNRangeExhausted)
		assert.Nil(t, result)
	})

	t.Run("returns error for non-existing tenant without listing l2vpns", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

		tenantName := "nonexistingtenant"
		tenancyListInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(&tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{Results: []*netboxModels.Tenant{}},
		}, nil)

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy},
		}

		result, err := compositeClient.GetAvailableL2VPNIdentifierByClaim(context.TODO(), &models.L2VPNClaim{
			Type:                 "vxlan-evpn",
			IdentifierRangeStart: 4000,
			IdentifierRangeEnd:   4001,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})
		AssertError(t, err, "failed to fetch tenant 'nonexistingtenant': not found")
		assert.Nil(t, result)
	})
}
