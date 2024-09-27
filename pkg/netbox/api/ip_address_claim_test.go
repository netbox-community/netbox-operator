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

	// test data for IPv4 ip address claim
	parentPrefixIdV4 := int64(3)
	parentPrefixV4 := "10.114.0.0"
	ipAddressV4_1 := "10.112.140.1/24"
	ipAddressV4_2 := "10.112.140.2/24"
	singleIpAddressV4_2 := "10.112.140.2/32"

	// example of available IPv4 IP addresses
	availAddressesIPv4 := func() []*netboxModels.AvailableIP {
		return []*netboxModels.AvailableIP{
			{
				Address: ipAddressV4_1,
				Family:  int64(IPv4Family),
			},
			{
				Address: ipAddressV4_2,
				Family:  int64(IPv4Family),
			},
		}
	}

	// test data for IPv6 ip address claim
	parentPrefixIdV6 := int64(4)
	parentPrefixV6 := "2001:db8:85a3:8d3::/64"
	ipAddressV6 := "2001:db8:85a3:8d3::2/64"
	singleIpAddressV6 := "2001:db8:85a3:8d3::2/128"

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
		input := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixIdV4)
		// ip address mock output
		output := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: availAddressesIPv4(),
		}

		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(input, nil).Return(output, nil)

		// init client
		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		actual, err := client.GetAvailableIpAddressesByParentPrefix(parentPrefixIdV4)

		// assert error return
		AssertNil(t, err)
		assert.Len(t, actual.Payload, 2)
		assert.Equal(t, ipAddressV4_1, actual.Payload[0].Address)
		assert.Equal(t, ipAddressV4_2, actual.Payload[1].Address)
	})

	t.Run("Fetch first available IP address by claim (IPv4).", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip address mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefixV4)
		// ip address mock output
		output := &ipam.IpamPrefixesListOK{
			Payload: &ipam.IpamPrefixesListOKBody{
				Results: []*netboxModels.Prefix{
					{
						ID:     parentPrefixIdV4,
						Prefix: &parentPrefixV4,
					},
				},
			},
		}

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixIdV4)
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{
					Address: ipAddressV4_2,
					Family:  int64(4),
				},
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
			ParentPrefix: parentPrefixV4,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		AssertNil(t, err)
		// assert nil output
		assert.Equal(t, singleIpAddressV4_2, actual.IpAddress)
	})

	t.Run("Fetch first available IP address by claim (IPv6).", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip address mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefixV6)
		// ip address mock output
		output := &ipam.IpamPrefixesListOK{
			Payload: &ipam.IpamPrefixesListOKBody{
				Results: []*netboxModels.Prefix{
					{
						ID:     parentPrefixIdV6,
						Prefix: &parentPrefixV6,
					},
				},
			},
		}

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixIdV6)
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{
					Address: ipAddressV6,
					Family:  int64(IPv6Family),
				},
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
			ParentPrefix: parentPrefixV6,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		AssertNil(t, err)
		// assert nil output
		assert.Equal(t, singleIpAddressV6, actual.IpAddress)
	})

	t.Run("Fetch first available IP address by claim (invalid IP family).", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip address mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefixV6)
		// ip address mock output
		output := &ipam.IpamPrefixesListOK{
			Payload: &ipam.IpamPrefixesListOKBody{
				Results: []*netboxModels.Prefix{
					{
						ID:     parentPrefixIdV6,
						Prefix: &parentPrefixV6,
					},
				},
			},
		}

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixIdV6)
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{
					Address: ipAddressV6,
					Family:  int64(5),
				},
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
			ParentPrefix: parentPrefixV6,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		AssertError(t, err, "available ip has unknown IP family")

		var expected *models.IPAddress

		assert.Equal(t, expected, actual)
	})

	t.Run("Fetch IP address's claim with incorrect parent prefix.", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip address mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefixV4)
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
			ParentPrefix: parentPrefixV4,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		expectedErrMsg := "failed to fetch parent prefix: not found"

		// assert error
		AssertError(t, err, expectedErrMsg)
		// assert nil output
		assert.Nil(t, actual)
	})

	t.Run("Fetch IP address's claim with exhausted parent prefix.", func(t *testing.T) {

		// ip address mock input
		input := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixIdV4)
		// ip address mock output
		output := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{},
		}

		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(input, nil).Return(output, nil)

		// init client
		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		actual, err := client.GetAvailableIpAddressesByParentPrefix(parentPrefixIdV4)

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

func TestIPAddressClaim_GetNoAvailableIPAddressWithTenancyChecks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	parentPrefix := "10.112.140.0/24"
	t.Run("No IP address asigned with an error when getting the tenant list", func(t *testing.T) {

		tenantName := "Tenant1"

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// expected error
		expectedErrorMsg := "cannot get the list" // testcase-defined error

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(nil, errors.New(expectedErrorMsg)).AnyTimes()

		// init client
		client := &NetboxClient{
			Tenancy: mockTenancy,
		}

		actual, err := client.GetAvailableIpAddressByClaim(&models.IPAddressClaim{
			ParentPrefix: parentPrefix,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		assert.Containsf(t, err.Error(), expectedErrorMsg, "Error should contain: %v, got: %v", expectedErrorMsg, err.Error())
		// assert nil output
		assert.Equal(t, actual, (*models.IPAddress)(nil))
	})

	t.Run("No IP address asigned with non-existing tenant", func(t *testing.T) {

		// non existing tenant
		nonExistingTenant := "non-existing-tenant"

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&nonExistingTenant)

		// empty tenant list
		emptyTenantList := &tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{
				Results: []*netboxModels.Tenant{},
			},
		}

		// expected error
		expectedErrorMsg := "failed to fetch tenant: not found"

		// mock empty list call
		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(emptyTenantList, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Tenancy: mockTenancy,
		}

		actual, err := client.GetAvailableIpAddressByClaim(&models.IPAddressClaim{
			ParentPrefix: parentPrefix,
			Metadata: &models.NetboxMetadata{
				Tenant: nonExistingTenant,
			},
		})

		// assert error
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
		// assert nil output
		assert.Equal(t, actual, (*models.IPAddress)(nil))
	})
}
