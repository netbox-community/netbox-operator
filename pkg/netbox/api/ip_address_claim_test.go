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

func TestIPAddressClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIPAddress := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	// test data for IPv4 ip address claim
	parentPrefixIdV4 := int32(3)
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
	parentPrefixIdV6 := int32(4)
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
		input := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixIdV4))
		// ip address mock output
		output := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: availAddressesIPv4(),
		}

		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(input, nil).Return(output, nil)

		// init client
		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		actual, err := compositeClient.GetAvailableIpAddressesByParentPrefix(parentPrefixIdV4)

		// assert error return
		AssertNil(t, err)
		assert.Len(t, actual.Payload, 2)
		assert.Equal(t, ipAddressV4_1, actual.Payload[0].Address)
		assert.Equal(t, ipAddressV4_2, actual.Payload[1].Address)
	})

	t.Run("Fetch first available IP address by claim (IPv4).", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{parentPrefixV4}).
			Return(mockListRequest)

		expectedPrefix := nclient.Prefix{
			Id:     parentPrefixIdV4,
			Prefix: parentPrefixV4,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&nclient.PaginatedPrefixList{Results: []nclient.Prefix{expectedPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixIdV4))
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{
					Address: ipAddressV4_2,
					Family:  int64(IPv4Family),
				},
			}}

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init client
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
			Ipam:    mockIPAddress,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
			clientV4: clientV4,
		}

		actual, err := compositeClient.GetAvailableIpAddressByClaim(
			context.TODO(),
			&models.IPAddressClaim{
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
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixIdV6))
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{
					Address: ipAddressV6,
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
			Id:     parentPrefixIdV6,
			Prefix: parentPrefixV6,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&nclient.PaginatedPrefixList{Results: []nclient.Prefix{expectedPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init client
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
			Ipam:    mockIPAddress,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
			clientV4: clientV4,
		}

		actual, err := compositeClient.GetAvailableIpAddressByClaim(
			context.TODO(),
			&models.IPAddressClaim{
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
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		inputIps := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixIdV6))
		outputIps := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{
				{
					Address: ipAddressV6,
					Family:  int64(5),
				},
			}}

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{parentPrefixV6}).
			Return(mockListRequest)

		expectedPrefix := nclient.Prefix{
			Id:     parentPrefixIdV6,
			Prefix: parentPrefixV6,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&nclient.PaginatedPrefixList{Results: []nclient.Prefix{expectedPrefix}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(inputIps, nil).Return(outputIps, nil)

		// init client
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
			Ipam:    mockIPAddress,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
			clientV4: clientV4,
		}

		actual, err := compositeClient.GetAvailableIpAddressByClaim(
			context.TODO(),
			&models.IPAddressClaim{
				ParentPrefix: parentPrefixV6,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
				},
			})

		// assert error
		AssertError(t, err, "unknown IP family")

		var expected *models.IPAddress

		assert.Equal(t, expected, actual)
	})

	t.Run("Fetch IP address's claim with incorrect parent prefix.", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{parentPrefixV4}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&nclient.PaginatedPrefixList{Results: []nclient.Prefix{}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant, nil).AnyTimes()

		// init client
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
			Ipam:    mockIPAddress,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
			clientV4: clientV4,
		}

		actual, err := compositeClient.GetAvailableIpAddressByClaim(
			context.TODO(),
			&models.IPAddressClaim{
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
		input := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixIdV4))
		// ip address mock output
		output := &ipam.IpamPrefixesAvailableIpsListOK{
			Payload: []*netboxModels.AvailableIP{},
		}

		mockIPAddress.EXPECT().IpamPrefixesAvailableIpsList(input, nil).Return(output, nil)

		// init client
		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		actual, err := compositeClient.GetAvailableIpAddressesByParentPrefix(parentPrefixIdV4)

		// assert error
		AssertError(t, err, ErrParentPrefixExhausted.Error())
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
		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		// 3rd parameter should be the variable `customIpSearch` below but go cannot compare functions so this errors.
		// Using ´gomock.Any()´ to allow this test to pass (shouldn't change the outcome of the tests anyway)
		//https://github.com/golang/mock/issues/324

		mockIPAddress.EXPECT().IpamIPAddressesList(ipam.NewIpamIPAddressesListParams(), nil, gomock.Any()).Return(output, nil)

		actual, err := compositeClient.RestoreExistingIpByHash(input)

		assert.Nil(t, err)
		assert.Equal(t, ipAddressRestore, actual.IpAddress)
	})
}

func TestIPAddressClaim_GetNoAvailableIPAddressWithTenancyChecks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	parentPrefix := "10.112.140.0/24"
	t.Run("No IP address assigned with an error when getting the tenant list", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		tenantName := "Tenant1"

		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// expected error
		expectedErrorMsg := "cannot get the list" // testcase-defined error

		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(nil, errors.New(expectedErrorMsg)).AnyTimes()

		// init client
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
			clientV4: clientV4,
		}

		actual, err := compositeClient.GetAvailableIpAddressByClaim(
			context.TODO(),
			&models.IPAddressClaim{
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

	t.Run("No IP address assigned with non-existing tenant", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

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
		expectedErrorMsg := "failed to fetch tenant 'non-existing-tenant': not found"

		// mock empty list call
		mockTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(emptyTenantList, nil).AnyTimes()

		// init client
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
			clientV4: clientV4,
		}

		actual, err := compositeClient.GetAvailableIpAddressByClaim(
			context.TODO(),
			&models.IPAddressClaim{
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
