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

	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func vlanWithVid(vid int32) v4client.VLAN {
	return v4client.VLAN{Vid: vid}
}

// vlanRestorationClient returns a composite client whose v3 VLAN list returns
// the given results. The custom field filter is passed as an opaque function
// that gomock cannot compare, hence gomock.Any() for the params and options.
func vlanRestorationClient(ctrl *gomock.Controller, results []*netboxModels.VLAN) *NetboxCompositeClient {
	mockIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockIpam.EXPECT().IpamVlansList(gomock.Any(), nil, gomock.Any()).Return(&ipam.IpamVlansListOK{
		Payload: &ipam.IpamVlansListOKBody{Results: results},
	}, nil)
	return &NetboxCompositeClient{clientV3: &NetboxClientV3{Ipam: mockIpam}}
}

func TestRestoreExistingVlanByHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("found", func(t *testing.T) {
		name := "vlanclaim-sample"
		vid := int64(100)
		compositeClient := vlanRestorationClient(ctrl, []*netboxModels.VLAN{
			{ID: 7, Name: &name, Vid: &vid},
		})

		result, err := compositeClient.RestoreExistingVlanByHash(context.TODO(), "matching-hash")
		AssertNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(100), result.Vid)
		assert.Equal(t, "vlanclaim-sample", result.Name)
		assert.Equal(t, int64(7), result.Id)
	})

	t.Run("not found", func(t *testing.T) {
		compositeClient := vlanRestorationClient(ctrl, nil)

		result, err := compositeClient.RestoreExistingVlanByHash(context.TODO(), "no-such-hash")
		AssertNil(t, err)
		assert.Nil(t, result)
	})

	t.Run("multiple results", func(t *testing.T) {
		name := "vlanclaim-sample"
		vid1, vid2 := int64(100), int64(101)
		compositeClient := vlanRestorationClient(ctrl, []*netboxModels.VLAN{
			{ID: 7, Name: &name, Vid: &vid1},
			{ID: 8, Name: &name, Vid: &vid2},
		})

		result, err := compositeClient.RestoreExistingVlanByHash(context.TODO(), "duplicate-hash")
		AssertError(t, err, "incorrect number of restoration results, number of results: 2")
		assert.Nil(t, result)
	})
}

func TestGetAvailableVlanByClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("empty range returns start", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidGte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidLte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(vlanListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{
			Count:   0,
			Results: []v4client.VLAN{},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableVlanByClaim(context.TODO(), &models.VlanClaim{
			VidRangeStart: 200,
			VidRangeEnd:   299,
		})
		AssertNil(t, err)
		assert.Equal(t, int32(200), result.Vid)
	})

	t.Run("finds gap after contiguous block from start", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidGte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidLte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(vlanListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{
			Count: 3,
			Results: []v4client.VLAN{
				vlanWithVid(200),
				vlanWithVid(202),
				vlanWithVid(201),
			},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableVlanByClaim(context.TODO(), &models.VlanClaim{
			VidRangeStart: 200,
			VidRangeEnd:   299,
		})
		AssertNil(t, err)
		assert.Equal(t, int32(203), result.Vid)
	})

	t.Run("only lists vids within the requested range", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidGte([]int32{200}).Return(mockListRequest)
		mockListRequest.EXPECT().VidLte([]int32{201}).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(vlanListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{
			Count:   1,
			Results: []v4client.VLAN{vlanWithVid(200)},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableVlanByClaim(context.TODO(), &models.VlanClaim{
			VidRangeStart: 200,
			VidRangeEnd:   201,
		})
		AssertNil(t, err)
		assert.Equal(t, int32(201), result.Vid)
	})

	t.Run("paginates across multiple pages", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequestPage1 := mock_interfaces.NewMockIpamVlansListRequest(ctrl)
		mockListRequestPage2 := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().VidGte(gomock.Any()).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().VidLte(gomock.Any()).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().Limit(vlanListPageSize).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().Offset(int32(0)).Return(mockListRequestPage1)
		mockListRequestPage1.EXPECT().Execute().Return(&v4client.PaginatedVLANList{
			Count:   2,
			Results: []v4client.VLAN{vlanWithVid(200)},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().VidGte(gomock.Any()).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().VidLte(gomock.Any()).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().Limit(vlanListPageSize).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().Offset(int32(1)).Return(mockListRequestPage2)
		mockListRequestPage2.EXPECT().Execute().Return(&v4client.PaginatedVLANList{
			Count:   2,
			Results: []v4client.VLAN{vlanWithVid(201)},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableVlanByClaim(context.TODO(), &models.VlanClaim{
			VidRangeStart: 200,
			VidRangeEnd:   299,
		})
		AssertNil(t, err)
		assert.Equal(t, int32(202), result.Vid)
	})

	t.Run("returns exhausted error when range is full", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidGte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidLte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(vlanListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{
			Count: 2,
			Results: []v4client.VLAN{
				vlanWithVid(200),
				vlanWithVid(201),
			},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableVlanByClaim(context.TODO(), &models.VlanClaim{
			VidRangeStart: 200,
			VidRangeEnd:   201,
		})
		assert.ErrorIs(t, err, ErrVlanRangeExhausted)
		assert.Nil(t, result)
	})

	t.Run("scopes the scan to the claim's site", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		siteName := "Site1"
		siteId := int64(3)
		dcimListInput := dcim.NewDcimSitesListParams().WithName(&siteName)
		dcimListOutput := &dcim.DcimSitesListOK{
			Payload: &dcim.DcimSitesListOKBody{
				Results: []*netboxModels.Site{
					{ID: siteId, Name: &siteName, Slug: &siteName},
				},
			},
		}
		mockDcim.EXPECT().DcimSitesList(dcimListInput, nil).Return(dcimListOutput, nil)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidGte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().VidLte(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Site([]string{siteName}).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(vlanListPageSize).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{
			Count:   0,
			Results: []v4client.VLAN{},
		}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI},
			clientV3: &NetboxClientV3{Dcim: mockDcim},
		}

		result, err := compositeClient.GetAvailableVlanByClaim(context.TODO(), &models.VlanClaim{
			VidRangeStart: 200,
			VidRangeEnd:   299,
			Metadata: &models.NetboxMetadata{
				Site: siteName,
			},
		})
		AssertNil(t, err)
		assert.Equal(t, int32(200), result.Vid)
	})

	t.Run("returns error for non-existing tenant without listing vlans", func(t *testing.T) {
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

		result, err := compositeClient.GetAvailableVlanByClaim(context.TODO(), &models.VlanClaim{
			VidRangeStart: 200,
			VidRangeEnd:   201,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})
		AssertError(t, err, "failed to fetch tenant 'nonexistingtenant': not found")
		assert.Nil(t, result)
	})
}
