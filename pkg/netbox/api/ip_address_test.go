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

	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	Comments    = "reservation for test comments"
	Status      = "active"
	Description = "reservation for test description"
	IpAddressId = int64(4)
)

func TestIPAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIPAddress := mock_interfaces.NewMockIpamInterface(ctrl)
	mockPrefixTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	//
	//outputMock := &ipam.Ipam
	ipAddress := "10.112.140.0"
	tenantId := int64(2)
	tenantName := "Tenant1"
	tenantOutputSlug := "tenant1"
	Label := "Status"
	Value := "active"

	// example input IP address
	writeableAddress := func() *netboxModels.WritableIPAddress {
		return &netboxModels.WritableIPAddress{
			Address:     &ipAddress,
			Comments:    Comments,
			Description: Description,
			Tenant:      &tenantId,
			Status:      Status,
		}
	}

	// example output IP address
	expectedIPAddress := func() *netboxModels.IPAddress {
		return &netboxModels.IPAddress{
			ID:          int64(1),
			Address:     &ipAddress,
			Display:     ipAddress,
			Comments:    Comments,
			Description: Description,
			Tenant: &netboxModels.NestedTenant{
				ID: tenantId,
			},
			Status: &netboxModels.IPAddressStatus{
				Label: &Label,
				Value: &Value,
			}}
	}

	// example of tenant
	expectedTenant := func() *tenancy.TenancyTenantsListOK {
		return &tenancy.TenancyTenantsListOK{
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
	}

	t.Run("Retrieve Existing static IP Address.", func(t *testing.T) {

		// id, address conversion from int64 to string
		address := ipAddress

		// tenant mock input
		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// tenant mock output

		// ip address mock input
		input := ipam.NewIpamIPAddressesListParams().WithAddress(&address)
		// ip address mock output
		output := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:          expectedIPAddress().ID,
						Address:     expectedIPAddress().Address,
						Comments:    expectedIPAddress().Comments,
						Description: expectedIPAddress().Description,
						Display:     expectedIPAddress().Display,
						Tenant:      expectedIPAddress().Tenant,
					},
				},
			},
		}

		mockPrefixTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant(), nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesList(input, nil).Return(output, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIPAddress,
			Tenancy: mockPrefixTenancy,
		}

		actual, err := client.GetIpAddress(&models.IPAddress{
			IpAddress: ipAddress,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error return
		AssertNil(t, err)
		assert.Equal(t, expectedIPAddress().ID, actual.Payload.Results[0].ID)
		assert.Equal(t, expectedIPAddress().Comments, actual.Payload.Results[0].Comments)
		assert.Equal(t, expectedIPAddress().Description, actual.Payload.Results[0].Description)
		assert.Equal(t, expectedIPAddress().Display, actual.Payload.Results[0].Display)
		assert.Equal(t, expectedIPAddress().Address, actual.Payload.Results[0].Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, actual.Payload.Results[0].Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, actual.Payload.Results[0].Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, actual.Payload.Results[0].Tenant.Slug)
	})

	t.Run("Retrieve Non Existing Static IP Address.", func(t *testing.T) {

		// id, address conversion from int64 to string
		address := ipAddress

		// tenant mock input
		inputTenant := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

		// tenant mock output

		// ip address mock input
		input := ipam.NewIpamIPAddressesListParams().WithAddress(&address)
		// ip address mock output
		output := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{},
			},
		}

		mockPrefixTenancy.EXPECT().TenancyTenantsList(inputTenant, nil).Return(expectedTenant(), nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesList(input, nil).Return(output, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIPAddress,
			Tenancy: mockPrefixTenancy,
		}

		res, err := client.GetIpAddress(&models.IPAddress{
			IpAddress: ipAddress,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error return
		AssertNil(t, err)
		assert.Zero(t, res.Payload.Count)
	})

	t.Run("Reserve Static IP Address.", func(t *testing.T) {

		// ip address mock inout
		input := ipam.NewIpamIPAddressesCreateParams().WithDefaults().WithData(writeableAddress())
		// ip address mock output
		output := &ipam.IpamIPAddressesCreateCreated{
			Payload: expectedIPAddress(),
		}

		mockIPAddress.EXPECT().IpamIPAddressesCreate(input, nil).Return(output, nil)

		// init client
		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		ipaddress, err := client.CreateIpAddress(writeableAddress())

		// assert error return
		AssertNil(t, err)

		// assert address elements
		AssertIpAddress(t, writeableAddress(), ipaddress)

	})

	t.Run("Check update of Static IP Address", func(t *testing.T) {

		input := ipam.NewIpamIPAddressesUpdateParams().WithDefaults().WithData(writeableAddress()).WithID(IpAddressId)

		output := &ipam.IpamIPAddressesUpdateOK{
			Payload: expectedIPAddress(),
		}

		mockIPAddress.EXPECT().IpamIPAddressesUpdate(input, nil).Return(output, nil)

		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		ipaddress, err := client.UpdateIpAddress(IpAddressId, writeableAddress())

		// assertion for errors
		AssertNil(t, err)

		// assert address properties
		AssertIpAddress(t, writeableAddress(), ipaddress)

	})

	t.Run("Check deletion of IP Address", func(t *testing.T) {

		input := ipam.NewIpamIPAddressesDeleteParams().WithID(IpAddressId)
		output := &ipam.IpamIPAddressesDeleteNoContent{}

		mockIPAddress.EXPECT().IpamIPAddressesDelete(input, nil).Return(output, nil)

		client := &NetboxClient{
			Ipam: mockIPAddress,
		}

		err := client.DeleteIpAddress(IpAddressId)
		AssertNil(t, err)
	})

}
