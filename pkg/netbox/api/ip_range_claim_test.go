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

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIPRangeClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIPRange := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// test data for IPv4 ip range claim
	parentPrefixId := int64(3)
	parentPrefixV4 := "10.114.0.0"
	requestedRangeSize := 3
	ipRangeV4_1 := "10.112.140.1/24"
	ipRangeV4_3 := "10.112.140.3/24"
	ipRangeV4_5 := "10.112.140.5/24"
	ipRangeV4_6 := "10.112.140.6/24"
	ipRangeV4_7 := "10.112.140.7/24"
	ipRangeV4_8 := "10.112.140.8/24"

	expectedIpDot5 := "10.112.140.5/32"
	expectedIpDot7 := "10.112.140.7/32"

	// example of available IPv4 ip address
	availableIpAdressesIPv4 := func() []*netboxModels.AvailableIP {
		return []*netboxModels.AvailableIP{
			{
				Address: ipRangeV4_1,
				Family:  int64(IPv4Family),
			},
			{
				Address: ipRangeV4_5,
				Family:  int64(IPv4Family),
			},
			{
				Address: ipRangeV4_3,
				Family:  int64(IPv4Family),
			},
			{
				Address: ipRangeV4_7,
				Family:  int64(IPv4Family),
			},
			{
				Address: ipRangeV4_6,
				Family:  int64(IPv4Family),
			},
			{
				Address: ipRangeV4_8,
				Family:  int64(IPv4Family),
			},
		}
	}

	// test data for IPv6 ip address claim
	parentPrefixV6 := "2001:db8:85a3:8d3::/64"
	ipAddressV6_1 := "2001:db8:85a3:8d3::2/64"
	ipAddressV6_2 := "2001:db8:85a3:8d3::3/64"

	//example of tenant
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

	t.Run("Fetch first available IP range by claim (IPv4).", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip range mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefixV4)
		// ip range mock output
		output := &ipam.IpamPrefixesListOK{
			Payload: &ipam.IpamPrefixesListOKBody{
				Results: []*netboxModels.Prefix{
					{
						ID:     parentPrefixId,
						Prefix: &parentPrefixV4,
					},
				},
			},
		}

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixId)
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: availableIpAdressesIPv4(),
		}

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPRange.EXPECT().IpamPrefixesList(input, nil).Return(output, nil)
		mockIPRange.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init client
		client := &NetboxClient{
			Tenancy: mockTenancy,
			Ipam:    mockIPRange,
		}

		actual, err := client.GetAvailableIpRangeByClaim(&models.IpRangeClaim{
			ParentPrefix: parentPrefixV4,
			Size:         requestedRangeSize,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		AssertNil(t, err)
		// assert nil output
		assert.Equal(t, expectedIpDot5, actual.StartAddress)
		assert.Equal(t, expectedIpDot7, actual.EndAddress)
	})

	t.Run("Fail first available IP range by claim (IPv6) if not enough consequiteve ips.", func(t *testing.T) {

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// ip range mock input
		input := ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefixV6)
		// ip range mock output
		output := &ipam.IpamPrefixesListOK{
			Payload: &ipam.IpamPrefixesListOKBody{
				Results: []*netboxModels.Prefix{
					{
						ID:     parentPrefixId,
						Prefix: &parentPrefixV6,
					},
				},
			},
		}

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixId)
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{
					Address: ipAddressV6_1,
					Family:  int64(IPv6Family),
				},
				{
					Address: ipAddressV6_2,
					Family:  int64(IPv6Family),
				},
			}}

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPRange.EXPECT().IpamPrefixesList(input, nil).Return(output, nil)
		mockIPRange.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init client
		client := &NetboxClient{
			Tenancy: mockTenancy,
			Ipam:    mockIPRange,
		}

		_, err := client.GetAvailableIpRangeByClaim(&models.IpRangeClaim{
			ParentPrefix: parentPrefixV6,
			Size:         requestedRangeSize,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error
		AssertError(t, err, "not enough consecutive IPs available")
	})

	t.Run("Fail with invalid input when searching Available Ip Range", func(t *testing.T) {
		payload := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{Address: "invalid ip address"},
			},
		}

		startAddress, endAddress, err := searchAvailableIpRange(payload, 3, int64(4))

		assert.Equal(t, "", startAddress)
		assert.Equal(t, "", endAddress)
		AssertError(t, err, "failed to parse IP address: invalid CIDR address: invalid ip address")
	})

	t.Run("Reclaim IP Range", func(t *testing.T) {

		input := "dummy-hash"
		output := &ipam.IpamIPRangesListOK{
			Payload: &ipam.IpamIPRangesListOKBody{
				Count:    nil,
				Next:     nil,
				Previous: nil,
				Results: []*netboxModels.IPRange{
					{
						StartAddress: &expectedIpDot5,
						EndAddress:   &expectedIpDot7,
					},
				},
			},
		}

		// init client
		client := &NetboxClient{
			Ipam: mockIPRange,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPRange.EXPECT().IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, gomock.Any()).Return(output, nil)

		actual, err := client.RestoreExistingIpRangeByHash(input)

		assert.Nil(t, err)
		assert.Equal(t, expectedIpDot5, actual.StartAddress)
		assert.Equal(t, expectedIpDot7, actual.EndAddress)
	})

	t.Run("Fail reclaim IP Range if multiple results returned", func(t *testing.T) {

		input := "dummy-hash"
		output := &ipam.IpamIPRangesListOK{
			Payload: &ipam.IpamIPRangesListOKBody{
				Count:    nil,
				Next:     nil,
				Previous: nil,
				Results: []*netboxModels.IPRange{
					{
						StartAddress: &expectedIpDot5,
						EndAddress:   &expectedIpDot7,
					},
					{
						StartAddress: &expectedIpDot5,
						EndAddress:   &expectedIpDot7,
					},
				},
			},
		}

		// init client
		client := &NetboxClient{
			Ipam: mockIPRange,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPRange.EXPECT().IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, gomock.Any()).Return(output, nil)

		_, err := client.RestoreExistingIpRangeByHash(input)

		AssertError(t, err, "incorrect number of restoration results, number of results: 2")
	})

	t.Run("Fail if returned ip range is invalid", func(t *testing.T) {

		input := "dummy-hash"
		output := &ipam.IpamIPRangesListOK{
			Payload: &ipam.IpamIPRangesListOKBody{
				Count:    nil,
				Next:     nil,
				Previous: nil,
				Results:  []*netboxModels.IPRange{{}},
			},
		}

		// init client
		client := &NetboxClient{
			Ipam: mockIPRange,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPRange.EXPECT().IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, gomock.Any()).Return(output, nil)

		_, err := client.RestoreExistingIpRangeByHash(input)

		AssertError(t, err, "invalid IP range")
	})
}

func TestIPRangeClaim_GetNoAvailableIPRangeWithNonExistingVrf(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpam := mock_interfaces.NewMockIpamInterface(ctrl)
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

	// non-existing vrf
	vrfName := "non-existing-vrf"
	inputVrf := ipam.NewIpamVrfsListParams().WithName(&vrfName)
	// empty vrf list
	emptyVrfList := &ipam.IpamVrfsListOK{
		Payload: &ipam.IpamVrfsListOKBody{
			Results: []*netboxModels.VRF{},
		},
	}

	mockIpam.EXPECT().IpamVrfsList(inputVrf, nil).Return(emptyVrfList, nil).AnyTimes()
	mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

	// expected error
	expectedErrorMsg := "failed to fetch vrf 'non-existing-vrf': not found"

	parentPrefix := "10.112.140.0/24"

	// init client
	client := &NetboxClient{
		Ipam:    mockIpam,
		Tenancy: mockTenancy,
	}

	actual, err := client.GetAvailableIpRangeByClaim(&models.IpRangeClaim{
		ParentPrefix: parentPrefix,
		Size:         3,
		Metadata: &models.NetboxMetadata{
			Tenant: tenantName,
			Vrf:    vrfName,
		},
	})

	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	assert.Equal(t, actual, (*models.IpRange)(nil))
}
