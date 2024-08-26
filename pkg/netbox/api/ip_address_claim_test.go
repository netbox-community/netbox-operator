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
	"testing"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIPAddressClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIPAddress := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	parentPrefixId := int64(3)
	parentPrefix := "10.114.0.0"
	ipAddress1 := "10.112.140.1/24"
	ipAddress2 := "10.112.140.2/24"
	singleIpAddress2 := "10.112.140.2/32"

	// example of available IP address
	availAddress := func() []*netboxModels.AvailableIP {
		return []*netboxModels.AvailableIP{
			{Address: ipAddress1},
			{Address: ipAddress2},
		}
	}

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

	t.Run("Fetch available IP address's claim by parent prefix.", func(t *testing.T) {

		// ip address mock input
		input := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixId)
		// ip address mock output
		output := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: availAddress(),
		}

		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(input, nil).Return(output, nil)

		// init client
		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		actual, err := client.GetAvailableIpAddressesByParentPrefix(parentPrefixId)

		// assert error return
		AssertNil(t, err)
		assert.Len(t, actual.Payload, 2)
		assert.Equal(t, ipAddress1, actual.Payload[0].Address)
		assert.Equal(t, ipAddress2, actual.Payload[1].Address)
	})

	t.Run("Fetch first available IP address by claim.", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip address mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefix)
		// ip address mock output
		output := &ipam.IpamPrefixesListOK{
			Payload: &ipam.IpamPrefixesListOKBody{
				Results: []*netboxModels.Prefix{
					{
						ID:     parentPrefixId,
						Prefix: &parentPrefix,
					},
				},
			},
		}

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixId)
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{Address: ipAddress2},
			}}

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamPrefixesList(input, nil).Return(output, nil)
		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init client
		client := &NetboxClient{
			Tenancy: mockTenancy,
			Ipam:    mockIPAddress,
		}

		actual, err := client.GetAvailableIpAddressByClaim(&models.IPAddressClaim{
			ParentPrefix: parentPrefix,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		AssertNil(t, err)
		// assert nil output
		assert.Equal(t, singleIpAddress2, actual.IpAddress)
	})

	t.Run("Fetch IP address's claim with incorrect parent prefix.", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip address mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefix)
		// ip address mock output
		output := &ipam.IpamPrefixesListOK{
			Payload: &ipam.IpamPrefixesListOKBody{
				Results: []*netboxModels.Prefix{},
			},
		}

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamPrefixesList(input, nil).Return(output, nil)

		// init client
		client := &NetboxClient{
			Tenancy: mockTenancy,
			Ipam:    mockIPAddress,
		}

		actual, err := client.GetAvailableIpAddressByClaim(&models.IPAddressClaim{
			ParentPrefix: parentPrefix,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		AssertError(t, err, "parent prefix not found")
		// assert nil output
		assert.Nil(t, actual)
	})

	t.Run("Fetch IP address's claim with exhausted parent prefix.", func(t *testing.T) {

		// ip address mock input
		input := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixId)
		// ip address mock output
		output := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{},
		}

		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(input, nil).Return(output, nil)

		// init client
		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		actual, err := client.GetAvailableIpAddressesByParentPrefix(parentPrefixId)

		// assert error
		AssertError(t, err, "parent prefix exhausted")
		// assert nil output
		AssertNil(t, actual)
	})

	t.Run("Reclaim IP Address", func(t *testing.T) {

		ipAddressRestore := "10.111.111.111/32"

		input := "403f19fcb98beaf5a25018536ed5275714a132ff"
		output := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Count:    nil,
				Next:     nil,
				Previous: nil,
				Results: []*netboxModels.IPAddress{
					{
						Address: &ipAddressRestore,
					},
				},
			},
		}

		// init client
		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPAddress.EXPECT().IpamIPAddressesList(ipam.NewIpamIPAddressesListParams(), nil, gomock.Any()).Return(output, nil)

		actual, err := client.RestoreExistingIpByHash(config.GetOperatorConfig().NetboxRestorationHashFieldName, input)

		assert.Nil(t, err)
		assert.Equal(t, ipAddressRestore, actual.IpAddress)
	})
}
