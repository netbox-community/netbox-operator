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
	"errors"
	"strconv"
	"testing"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
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
	parentPrefixId := int64(3)
	availablePrefixListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
	//prefix mock output
	childPrefix1 := "10.112.140.0/24"
	childPrefix2 := "10.120.180.0/24"
	availablePrefixListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{Prefix: childPrefix1},
			{Prefix: childPrefix2},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(availablePrefixListInput, nil).Return(availablePrefixListOutput, nil)

	netboxClient := &NetboxClient{
		Ipam: mockPrefixIpam,
	}

	actual, err := netboxClient.GetAvailablePrefixesByParentPrefix(parentPrefixId)
	assert.Nil(t, err)
	assert.Equal(t, childPrefix1, actual.Payload[0].Prefix)
	assert.Equal(t, childPrefix2, actual.Payload[1].Prefix)
}

func TestPrefixClaim_GetNoAvailablePrefixesByParentPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	//prefix mock input
	parentPrefixId := int64(3)
	availablePrefixListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
	//prefix mock output
	availablePrefixListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(availablePrefixListInput, nil).Return(availablePrefixListOutput, nil)

	netboxClient := &NetboxClient{
		Ipam: mockPrefixIpam,
	}

	actual, err := netboxClient.GetAvailablePrefixesByParentPrefix(parentPrefixId)
	assert.Nil(t, actual)
	assert.EqualError(t, err, "parent prefix exhausted")
}

func TestPrefixClaim_GetAvailablePrefixByClaim_WithWrongParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
	prefixListInput := ipam.
		NewIpamPrefixesListParams().
		WithPrefix(&prefix)
	prefixListOutput := &ipam.IpamPrefixesListOK{
		Payload: &ipam.IpamPrefixesListOKBody{
			Results: []*netboxModels.Prefix{},
		},
	}

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockPrefixIpam.EXPECT().IpamPrefixesList(prefixListInput, nil).Return(prefixListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	netboxClient := &NetboxClient{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}

	actual, err := netboxClient.GetAvailablePrefixByClaim(&models.PrefixClaim{
		ParentPrefix: prefix,
		PrefixLength: "/28",
	})
	assert.Nil(t, actual)
	assert.EqualError(t, err, "parent prefix not found")
}

func TestPrefixClaim_GetBestFitPrefixByClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
	parentPrefixId := int64(1)
	prefix := "10.112.140.14/28"
	prefixListInput := ipam.
		NewIpamPrefixesListParams().
		WithPrefix(&parentPrefix)
	prefixListOutput := &ipam.IpamPrefixesListOK{
		Payload: &ipam.IpamPrefixesListOKBody{
			Results: []*netboxModels.Prefix{
				{
					ID:     parentPrefixId,
					Prefix: &parentPrefix,
				},
			},
		},
	}

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix,
			},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesList(prefixListInput, nil).Return(prefixListOutput, nil).AnyTimes()
	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	netboxClient := &NetboxClient{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}

	actual, err := netboxClient.GetAvailablePrefixByClaim(&models.PrefixClaim{
		ParentPrefix: parentPrefix,
		PrefixLength: "/28",
	})

	assert.Nil(t, err)
	assert.Equal(t, prefix, actual.Prefix)
}

func TestPrefixClaim_GetBestFitPrefixByClaimNoAvailablePrefixMatchesSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
	parentPrefixId := int64(1)
	prefix := "10.112.140.0/23"
	prefix1 := "10.112.142.32/27"
	prefix2 := "10.112.142.64/26"
	prefixListInput := ipam.
		NewIpamPrefixesListParams().
		WithPrefix(&parentPrefix)
	prefixListOutput := &ipam.IpamPrefixesListOK{
		Payload: &ipam.IpamPrefixesListOKBody{
			Results: []*netboxModels.Prefix{
				{
					ID:     parentPrefixId,
					Prefix: &parentPrefix,
				},
			},
		},
	}

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
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

	mockPrefixIpam.EXPECT().IpamPrefixesList(prefixListInput, nil).Return(prefixListOutput, nil).AnyTimes()
	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	netboxClient := &NetboxClient{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}

	actual, err := netboxClient.GetAvailablePrefixByClaim(&models.PrefixClaim{
		ParentPrefix: parentPrefix,
		PrefixLength: "/28",
	})

	assert.Nil(t, err)
	assert.Equal(t, "10.112.142.32/28", actual.Prefix)
}

func TestPrefixClaim_GetBestFitPrefixByClaimNoAvailablePrefixMatchesSizeCriteria(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
	parentPrefixId := int64(1)
	prefix := "10.112.140.14/30"
	prefixListInput := ipam.
		NewIpamPrefixesListParams().
		WithPrefix(&parentPrefix)
	prefixListOutput := &ipam.IpamPrefixesListOK{
		Payload: &ipam.IpamPrefixesListOKBody{
			Results: []*netboxModels.Prefix{
				{
					ID:     parentPrefixId,
					Prefix: &parentPrefix,
				},
			},
		},
	}

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix,
			},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesList(prefixListInput, nil).Return(prefixListOutput, nil).AnyTimes()
	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	netboxClient := &NetboxClient{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}

	_, err := netboxClient.GetAvailablePrefixByClaim(&models.PrefixClaim{
		ParentPrefix: parentPrefix,
		PrefixLength: "/28",
	})

	assert.True(t, errors.Is(err, ErrNoPrefixMatchsSizeCriteria))
}

func TestPrefixClaim_GetBestFitPrefixByClaimInvalidFormatFromNetbox(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
	parentPrefixId := int64(1)
	prefix1 := "10.112.140.0"
	prefix2 := "10.112.142.32/27"
	prefix3 := "10.112.142.64/26."
	prefixListInput := ipam.
		NewIpamPrefixesListParams().
		WithPrefix(&parentPrefix)
	prefixListOutput := &ipam.IpamPrefixesListOK{
		Payload: &ipam.IpamPrefixesListOKBody{
			Results: []*netboxModels.Prefix{
				{
					ID:     parentPrefixId,
					Prefix: &parentPrefix,
				},
			},
		},
	}

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
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

	mockPrefixIpam.EXPECT().IpamPrefixesList(prefixListInput, nil).Return(prefixListOutput, nil).AnyTimes()
	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	netboxClient := &NetboxClient{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}

	actual, err := netboxClient.GetAvailablePrefixByClaim(&models.PrefixClaim{
		ParentPrefix: parentPrefix,
		PrefixLength: "/28",
	})

	assert.Nil(t, err)
	assert.Equal(t, "10.112.142.32/28", actual.Prefix)
}

func TestPrefixClaim_GetBestFitPrefixByClaimInvalidPrefixClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
	parentPrefixId := int64(1)
	prefix1 := "10.112.140.14/25"
	prefixListInput := ipam.
		NewIpamPrefixesListParams().
		WithPrefix(&parentPrefix)
	prefixListOutput := &ipam.IpamPrefixesListOK{
		Payload: &ipam.IpamPrefixesListOKBody{
			Results: []*netboxModels.Prefix{
				{
					ID:     parentPrefixId,
					Prefix: &parentPrefix,
				},
			},
		},
	}

	inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	prefixAvailableListInput := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
	prefixAvailableListOutput := &ipam.IpamPrefixesAvailablePrefixesListOK{
		Payload: []*netboxModels.AvailablePrefix{
			{
				Prefix: prefix1,
			},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesList(prefixListInput, nil).Return(prefixListOutput, nil).AnyTimes()
	mockPrefixIpam.EXPECT().IpamPrefixesAvailablePrefixesList(prefixAvailableListInput, nil).Return(prefixAvailableListOutput, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	netboxClient := &NetboxClient{
		Ipam:    mockPrefixIpam,
		Tenancy: mockTenancy,
	}

	_, err := netboxClient.GetAvailablePrefixByClaim(&models.PrefixClaim{
		ParentPrefix: parentPrefix,
		PrefixLength: "/28.",
	})

	assert.True(t, errors.Is(err, strconv.ErrSyntax))
}
