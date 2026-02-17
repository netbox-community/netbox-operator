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

package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPrefixClaim_GetAvailablePrefixesByParentPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	//prefix mock input
	parentPrefixId := int32(3)
	availablePrefixListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))

	//prefix mock output
	childPrefix1 := "10.112.140.0/24"
	childPrefix2 := "10.120.180.0/24"
	availablePrefixListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: childPrefix1,
				Family: int64(IPv4Family),
			},
			{
				Prefix: childPrefix2,
				Family: int64(IPv4Family),
			},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(availablePrefixListInput, nil).Return(availablePrefixListOutput, nil)

	clientV3 := &NetboxClientV3{
		Ipam: mockPrefixIpam,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
	}

	actual, err := compositeClient.GetAvailablePrefixesByParentPrefix(parentPrefixId)
	assert.Nil(t, err)
	assert.Equal(t, childPrefix1, actual.Payload[0].Prefix)
	assert.Equal(t, childPrefix2, actual.Payload[1].Prefix)
}

func TestPrefixClaim_GetNoAvailablePrefixesByParentPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	//prefix mock input
	parentPrefixId := int32(3)
	availablePrefixListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	//prefix mock output
	availablePrefixListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(availablePrefixListInput, nil).Return(availablePrefixListOutput, nil)

	clientV3 := &NetboxClientV3{
		Ipam: mockPrefixIpam,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
	}

	actual, err := compositeClient.GetAvailablePrefixesByParentPrefix(parentPrefixId)
	assert.Nil(t, actual)
	assert.ErrorIs(t, err, ErrParentPrefixExhausted)
}

func TestPrefixClaim_GetAvailablePrefixByClaim_WithWrongParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	prefix := "10.112.140.0/24"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{prefix}).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: prefix,
			PrefixLength: "/28",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})
	assert.Nil(t, actual)
	assert.EqualError(t, err, "parent prefix not found")
}

func TestPrefixClaim_GetBestFitPrefixByClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}
	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	// example of site
	siteId := int64(3)
	siteName := "Site1"
	siteOutputSlug := "site1"
	expectedSite := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{
				{
					ID:   siteId,
					Name: &siteName,
					Slug: &siteOutputSlug,
				},
			},
		},
	}
	inputSite := dcim.NewDcimSitesListParams().WithName(&siteName)

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)
	prefix := "10.112.140.14/28"

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix,
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	expectedPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expectedPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
	mockDcim.EXPECT().DcimSitesList(inputSite, nil).Return(expectedSite, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Tenancy: mockTenancy,
		Dcim:    mockDcim,
		Ipam:    mockPrefixIpam,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
				Site:   siteName,
			},
		})

	assert.Nil(t, err)
	assert.Equal(t, prefix, actual.Prefix)
}

func TestPrefixClaim_InvalidIPv4PrefixLength(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/33",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	var expectedPrefix *models.Prefix

	assert.Error(t, err)
	assert.Equal(t, expectedPrefix, actual)
}

func TestPrefixClaim_FailWhenRequestingEntirePrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/24",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	var expectedPrefix *models.Prefix

	assert.Error(t, err)
	assert.Equal(t, expectedPrefix, actual)
}

func TestPrefixClaim_FailWhenPrefixLargerThanParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/20",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	var expectedPrefix *models.Prefix

	assert.Error(t, err)
	assert.Equal(t, expectedPrefix, actual)
}

func TestPrefixClaim_ValidIPv6PrefixLength(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "2001:db8:85a3:8d3::/30"
	parentPrefixId := int32(1)
	prefix := "2001:db8:85a3:8d3::/33"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix,
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv6Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/33",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	assert.Nil(t, err)
	assert.Equal(t, prefix, actual.Prefix)
}

func TestPrefixClaim_GetBestFitPrefixByClaimNoAvailablePrefixMatchesSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/22"
	parentPrefixId := int32(1)
	prefix := "10.112.140.0/23"
	prefix1 := "10.112.142.32/27"
	prefix2 := "10.112.142.64/26"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix,
			},
			{
				Prefix: prefix1,
			},
			{
				Prefix: prefix2,
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	assert.Nil(t, err)
	assert.Equal(t, "10.112.142.32/28", actual.Prefix)
}

func TestPrefixClaim_GetBestFitPrefixByClaimNoAvailablePrefixMatchesSizeCriteria(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)
	prefix := "10.112.140.14/30"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix,
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	_, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	assert.True(t, errors.Is(err, ErrNoPrefixMatchsSizeCriteria))
}

func TestPrefixClaim_GetBestFitPrefixByClaimInvalidFormatFromNetbox(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/22"
	parentPrefixId := int32(1)
	prefix1 := "10.112.140.0"
	prefix2 := "10.112.142.32/27"
	prefix3 := "10.112.142.64/26."

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix1,
			},
			{
				Prefix: prefix2,
			},
			{
				Prefix: prefix3,
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	assert.Nil(t, err)
	assert.Equal(t, "10.112.142.32/28", actual.Prefix)
}

func TestPrefixClaim_GetBestFitPrefixByClaimInvalidPrefixClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// example of tenant
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)
	prefix1 := "10.112.140.14/25"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix1,
			},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	_, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28.",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	assert.True(t, errors.Is(err, strconv.ErrSyntax))
}

func TestPrefixClaim_GetNoAvailablePrefixesWithNonExistingTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// non-existing tenant
	tenantName := "non-existing-tenant"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	// expected error
	expectedErrorMsg := "failed to fetch tenant 'non-existing-tenant': not found"

	// empty tenant list
	emptyTenantList := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{},
		},
	}

	parentPrefix := "10.112.140.0/24"

	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(emptyTenantList, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	prefix, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28.",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	assert.Equal(t, prefix, (*models.Prefix)(nil))
}

func TestPrefixClaim_GetNoAvailablePrefixesWithErrorWhenGettingTenantList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// non-existing tenant
	tenantName := "non-existing tenant"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	// expected errors
	getTenantDetailsErrorMsg := "failed to fetch Tenant details"
	tenancyTenantsListErrorMsg := "cannot get the list" // testcase defined error

	parentPrefix := "10.112.140.0/24"

	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(nil, errors.New(tenancyTenantsListErrorMsg)).AnyTimes()

	clientV3 := &NetboxClientV3{
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	prefix, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28.",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

	// assert 1st level error - GetTenantDetails()
	assert.Containsf(t, err.Error(), getTenantDetailsErrorMsg, "expected error containing %q, got %s", getTenantDetailsErrorMsg, err)

	// assert 2nd level error - TenanyTenantsList()
	assert.Containsf(t, err.Error(), tenancyTenantsListErrorMsg, "expected error containing %q, got %s", tenancyTenantsListErrorMsg, err)

	// assert nil output
	assert.Equal(t, prefix, (*models.Prefix)(nil))
}

func TestPrefixClaim_GetNoAvailablePrefixesWithNonExistingSite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// tenant
	tenantName := "tenant"
	tenantId := int64(2)
	tenantOutputSlug := "tenant1"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	// non-existing site
	siteName := "non-existing-site"
	inputSite := dcim.NewDcimSitesListParams().WithName(&siteName)
	// empty site list
	emptySiteList := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{},
		},
	}

	mockDcim.EXPECT().DcimSitesList(inputSite, nil).Return(emptySiteList, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	// expected error
	expectedErrorMsg := "failed to fetch site 'non-existing-site': not found"

	parentPrefix := "10.112.140.0/24"

	clientV3 := &NetboxClientV3{
		Dcim:    mockDcim,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	prefix, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28.",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
				Site:   siteName,
			},
		})

	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	assert.Equal(t, prefix, (*models.Prefix)(nil))
}

func TestPrefixClaim_GetAvailablePrefixIfNoSiteInSpec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	// tenant
	tenantName := "tenant"
	tenantId := int64(2)
	tenantOutputSlug := "tenant1"

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)
	prefix := "10.112.140.14/28"

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix,
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixByClaim(
		context.TODO(),
		&models.PrefixClaim{
			ParentPrefix: parentPrefix,
			PrefixLength: "/28",
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
				Site:   "",
			},
		})

	assert.Nil(t, err)
	assert.Equal(t, prefix, actual.Prefix)
}

func TestPrefixClaim_GetAvailablePrefixByParentPrefixSelector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)

	pxcSpec := netboxv1.PrefixClaimSpec{
		ParentPrefixSelector: map[string]string{
			"environment": "dev",
			"family":      "IPv4",
			"tenant":      "tenant",
			"site":        "Site1",
		},
		PrefixLength: "/32",
	}

	// example of site
	siteId := int64(3)
	siteName := "Site1"
	siteOutputSlug := "site1"
	expectedSite := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{
				{
					ID:   siteId,
					Name: &siteName,
					Slug: &siteOutputSlug,
				},
			},
		},
	}
	inputSite := dcim.NewDcimSitesListParams().WithName(&siteName)

	// tenant
	tenantName := "tenant"
	tenantId := int64(2)
	tenantOutputSlug := "tenant1"

	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	parentPrefix := "10.112.140.0/24"
	parentPrefixId := int32(1)

	// get prefix to check if it's a candidate
	expectedCustomFieldName := "environment"
	expectedCustomFields := &extras.ExtrasCustomFieldsListOK{
		Payload: &extras.ExtrasCustomFieldsListOKBody{
			Results: []*netboxModels.CustomField{
				{
					Name: &expectedCustomFieldName,
				},
			},
		},
	}

	prefixFamily := int64(IPv4Family)
	prefixFamilyLabel := netboxModels.PrefixFamilyLabelIPV4
	prefixListInputWithParam := ipam.NewIpamPrefixesListParams()
	prefixListOutputWithParam := &ipam.IpamPrefixesListOK{
		Payload: &ipam.IpamPrefixesListOKBody{
			Results: []*netboxModels.Prefix{
				{
					Prefix: &parentPrefix,
					ID:     int64(parentPrefixId),
					Family: &netboxModels.PrefixFamily{Label: &prefixFamilyLabel, Value: &prefixFamily},
				},
			},
		},
	}
	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(int64(parentPrefixId))
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Family: prefixFamily,
				Prefix: parentPrefix,
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{parentPrefix}).
		Return(mockListRequest)

	aggregateFamily := v4client.NewAggregateFamily()
	aggregateFamily.SetValue(v4client.AggregateFamilyValue(IPv4Family))

	outputPrefix := v4client.Prefix{
		Id:     parentPrefixId,
		Prefix: parentPrefix,
		Family: *aggregateFamily,
	}

	mockListRequest.EXPECT().
		Execute().
		Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{outputPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	mockPrefixIpam.EXPECT().IpamPrefixesList(prefixListInputWithParam, nil, gomock.Any()).Return(prefixListOutputWithParam, nil).Times(1)
	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(gomock.Any(), nil).Return(expectedTenant, nil).AnyTimes()
	mockDcim.EXPECT().DcimSitesList(inputSite, nil).Return(expectedSite, nil).AnyTimes()
	mockExtras.EXPECT().ExtrasCustomFieldsList(extras.NewExtrasCustomFieldsListParams(), gomock.Any(), gomock.Any()).Return(expectedCustomFields, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
		Extras:  mockExtras,
		Dcim:    mockDcim,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixesByParentPrefixSelector(context.TODO(), &pxcSpec)

	assert.Nil(t, err)
	assert.Equal(t, parentPrefix, actual[0].Prefix)
}

func TestPrefixClaim_GetAvailablePrefixByParentPrefixSelectorFailIfNonExistingFieldInParentPrefixSelector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)

	pxcSpec := netboxv1.PrefixClaimSpec{
		ParentPrefixSelector: map[string]string{
			"non-existing": "non-existing",
			"environment":  "dev",
			"family":       "IPv4",
			"tenant":       "tenant",
			"site":         "Site1",
		},
		PrefixLength: "/32",
	}

	// example of site
	siteId := int64(3)
	siteName := "Site1"
	siteOutputSlug := "site1"
	expectedSite := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{
				{
					ID:   siteId,
					Name: &siteName,
					Slug: &siteOutputSlug,
				},
			},
		},
	}
	inputSite := dcim.NewDcimSitesListParams().WithName(&siteName)

	// tenant
	tenantName := "tenant"
	tenantId := int64(2)
	tenantOutputSlug := "tenant1"

	expectedTenant := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	// get prefix to check if it's a candidate
	expectedCustomFieldName := "environment"
	expectedCustomFields := &extras.ExtrasCustomFieldsListOK{
		Payload: &extras.ExtrasCustomFieldsListOKBody{
			Results: []*netboxModels.CustomField{
				{
					Name: &expectedCustomFieldName,
				},
			},
		},
	}

	mockTenancy.EXPECT().TenancyTenantsList(gomock.Any(), nil).Return(expectedTenant, nil).AnyTimes()
	mockDcim.EXPECT().DcimSitesList(inputSite, nil).Return(expectedSite, nil).AnyTimes()
	mockExtras.EXPECT().ExtrasCustomFieldsList(extras.NewExtrasCustomFieldsListParams(), gomock.Any(), gomock.Any()).Return(expectedCustomFields, nil).AnyTimes()

	clientV3 := &NetboxClientV3{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
		Extras:  mockExtras,
		Dcim:    mockDcim,
	}
	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}

	actual, err := compositeClient.GetAvailablePrefixesByParentPrefixSelector(context.TODO(), &pxcSpec)

	assert.Nil(t, actual)
	AssertError(t, err, "invalid parentPrefixSelector, netbox custom fields non-existing do not exist")
}
