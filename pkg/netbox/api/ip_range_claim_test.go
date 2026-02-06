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
	"net/http"
	"testing"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	nclient "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIPRangeClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIPRange := mock_interfaces.NewMockIpamInterface(ctrl)
	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// test data for IPv4 ip range claim
	parentPrefixId := int32(3)
	parentPrefixV4 := "10.114.0.0/24"
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

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixId))
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: availableIpAdressesIPv4(),
		}

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{parentPrefixV4}).
			Return(mockListRequest)

		expectedPrefix := nclient.Prefix{
			Id:     parentPrefixId,
			Prefix: parentPrefixV4,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&nclient.PaginatedPrefixList{Results: []nclient.Prefix{expectedPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPRange.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init legacyClient
		legacyClient := &NetboxClient{
			Tenancy: mockTenancy,
			Ipam:    mockIPRange,
		}
		client := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}

		actual, err := legacyClient.GetAvailableIpRangeByClaim(
			context.TODO(),
			client,
			&models.IpRangeClaim{
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

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixId))
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

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{parentPrefixV6}).
			Return(mockListRequest)

		expectedPrefix := nclient.Prefix{
			Id:     parentPrefixId,
			Prefix: parentPrefixV6,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&nclient.PaginatedPrefixList{Results: []nclient.Prefix{expectedPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPRange.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init client
		legacyClient := &NetboxClient{
			Tenancy: mockTenancy,
			Ipam:    mockIPRange,
		}
		client := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}

		_, err := legacyClient.GetAvailableIpRangeByClaim(
			context.TODO(),
			client,
			&models.IpRangeClaim{
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
		legacyClient := &NetboxClient{
			Ipam: mockIPRange,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPRange.EXPECT().IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, gomock.Any()).Return(output, nil)

		actual, err := legacyClient.RestoreExistingIpRangeByHash(input)

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
		legacyClien := &NetboxClient{
			Ipam: mockIPRange,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPRange.EXPECT().IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, gomock.Any()).Return(output, nil)

		_, err := legacyClien.RestoreExistingIpRangeByHash(input)

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
		legacyClient := &NetboxClient{
			Ipam: mockIPRange,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPRange.EXPECT().IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, gomock.Any()).Return(output, nil)

		_, err := legacyClient.RestoreExistingIpRangeByHash(input)

		AssertError(t, err, "invalid IP range")
	})
}
