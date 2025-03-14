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
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	IpRangeId = int64(4)
)

func TestIpRange(t *testing.T) {
	ctrl := gomock.NewController(t, gomock.WithOverridableExpectations())
	defer ctrl.Finish()
	mockIpam := mock_interfaces.NewMockIpamInterface(ctrl)
	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)

	startAddress := "10.112.140.1"
	endAddress := "10.112.140.3"
	tenantId := int64(2)
	tenantName := "Tenant1"
	Label := "Status"
	Value := "active"

	// example output ip range
	expectedIPRange := func() *netboxModels.IPRange {
		return &netboxModels.IPRange{
			ID:           int64(1),
			StartAddress: &startAddress,
			EndAddress:   &endAddress,
			Comments:     Comments,
			Description:  Description,
			Tenant: &netboxModels.NestedTenant{
				ID: tenantId,
			},
			Status: &netboxModels.IPRangeStatus{
				Label: &Label,
				Value: &Value,
			}}
	}

	t.Run("Retrieve Existing IP Range.", func(t *testing.T) {

		// ip range mock input
		input := ipam.NewIpamIPRangesListParams().
			WithStartAddress(&startAddress).
			WithEndAddress(&endAddress)

		// ip range mock output
		output := &ipam.IpamIPRangesListOK{
			Payload: &ipam.IpamIPRangesListOKBody{
				Results: []*netboxModels.IPRange{
					{
						ID:           expectedIPRange().ID,
						StartAddress: &startAddress,
						EndAddress:   &endAddress,
						Comments:     expectedIPRange().Comments,
						Description:  expectedIPRange().Description,
						Tenant:       expectedIPRange().Tenant,
					},
				},
			},
		}

		mockIpam.EXPECT().IpamIPRangesList(input, nil).Return(output, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIpam,
			Tenancy: mockTenancy,
		}

		actual, err := client.GetIpRange(&models.IpRange{
			StartAddress: startAddress,
			EndAddress:   endAddress,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error return
		AssertNil(t, err)
		assert.Equal(t, expectedIPRange().ID, actual.Payload.Results[0].ID)
		assert.Equal(t, expectedIPRange().Comments, actual.Payload.Results[0].Comments)
		assert.Equal(t, expectedIPRange().Description, actual.Payload.Results[0].Description)
		assert.Equal(t, expectedIPRange().StartAddress, actual.Payload.Results[0].StartAddress)
		assert.Equal(t, expectedIPRange().EndAddress, actual.Payload.Results[0].EndAddress)
		assert.Equal(t, expectedIPRange().Tenant.ID, actual.Payload.Results[0].Tenant.ID)
		assert.Equal(t, expectedIPRange().Tenant.Name, actual.Payload.Results[0].Tenant.Name)
		assert.Equal(t, expectedIPRange().Tenant.Slug, actual.Payload.Results[0].Tenant.Slug)
	})

	t.Run("ReserveOrUpdate, reserve new ip range", func(t *testing.T) {

		// ip range mock input
		listInput := ipam.NewIpamIPRangesListParams().
			WithStartAddress(&startAddress).
			WithEndAddress(&endAddress)

		// ip range mock output
		listOutput := &ipam.IpamIPRangesListOK{
			Payload: &ipam.IpamIPRangesListOKBody{
				Results: []*netboxModels.IPRange{},
			},
		}

		createOutput := &ipam.IpamIPRangesCreateCreated{
			Payload: &netboxModels.IPRange{
				ID:           expectedIPRange().ID,
				StartAddress: &startAddress,
				EndAddress:   &endAddress,
				Comments:     expectedIPRange().Comments,
				Description:  expectedIPRange().Description,
				Tenant:       expectedIPRange().Tenant,
			},
		}

		tenancyListInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
		tenancyListOutput := &tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{
				Results: []*netboxModels.Tenant{
					{
						ID:   tenantId,
						Name: &tenantName,
						Slug: &tenantName,
					},
				},
			},
		}

		mockIpam.EXPECT().IpamIPRangesList(listInput, nil).Return(listOutput, nil).AnyTimes()
		// use gomock.Any() because there where issues with comparing the pointers in the request
		mockIpam.EXPECT().IpamIPRangesCreate(gomock.Any(), nil).Return(createOutput, nil).AnyTimes()
		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(tenancyListOutput, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIpam,
			Tenancy: mockTenancy,
		}

		actual, err := client.ReserveOrUpdateIpRange(&models.IpRange{
			StartAddress: startAddress,
			EndAddress:   endAddress,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error return
		AssertNil(t, err)
		assert.Equal(t, expectedIPRange().ID, actual.ID)
		assert.Equal(t, expectedIPRange().Comments, actual.Comments)
		assert.Equal(t, expectedIPRange().Description, actual.Description)
		assert.Equal(t, expectedIPRange().StartAddress, actual.StartAddress)
		assert.Equal(t, expectedIPRange().EndAddress, actual.EndAddress)
		assert.Equal(t, expectedIPRange().Tenant.ID, actual.Tenant.ID)
		assert.Equal(t, expectedIPRange().Tenant.Name, actual.Tenant.Name)
		assert.Equal(t, expectedIPRange().Tenant.Slug, actual.Tenant.Slug)
	})

	t.Run("ReserveOrUpdate, restoration hash missmatch", func(t *testing.T) {

		// ip range mock input
		listInput := ipam.NewIpamIPRangesListParams().
			WithStartAddress(&startAddress).
			WithEndAddress(&endAddress)

		// ip range mock output
		listOutput := &ipam.IpamIPRangesListOK{
			Payload: &ipam.IpamIPRangesListOKBody{
				Results: []*netboxModels.IPRange{
					{
						ID:           expectedIPRange().ID,
						StartAddress: &startAddress,
						EndAddress:   &endAddress,
						CustomFields: map[string]string{
							config.GetOperatorConfig().NetboxRestorationHashFieldName: "different hash",
						},
						Comments:    expectedIPRange().Comments,
						Description: expectedIPRange().Description,
						Tenant:      expectedIPRange().Tenant,
					},
				},
			},
		}

		mockIpam.EXPECT().IpamIPRangesList(listInput, nil).Return(listOutput, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam: mockIpam,
		}

		_, err := client.ReserveOrUpdateIpRange(&models.IpRange{
			StartAddress: startAddress,
			EndAddress:   endAddress,
			Metadata: &models.NetboxMetadata{
				Custom: map[string]string{
					config.GetOperatorConfig().NetboxRestorationHashFieldName: "hash",
				},
			},
		})

		// assert error return
		AssertError(t, err, "restoration hash missmatch, assigned ip range 10.112.140.1-10.112.140.3")
	})

	t.Run("ReserveOrUpdate, update existing ip range", func(t *testing.T) {

		// ip range mock input
		listInput := ipam.NewIpamIPRangesListParams().
			WithStartAddress(&startAddress).
			WithEndAddress(&endAddress)

		// ip range mock output
		listOutput := &ipam.IpamIPRangesListOK{
			Payload: &ipam.IpamIPRangesListOKBody{
				Results: []*netboxModels.IPRange{
					{
						ID:           expectedIPRange().ID,
						StartAddress: &startAddress,
						EndAddress:   &endAddress,
						Comments:     expectedIPRange().Comments,
						Description:  expectedIPRange().Description,
						Tenant:       expectedIPRange().Tenant,
					},
				},
			},
		}

		updateOutput := &ipam.IpamIPRangesUpdateOK{
			Payload: &netboxModels.IPRange{
				ID:           expectedIPRange().ID,
				StartAddress: &startAddress,
				EndAddress:   &endAddress,
				Comments:     expectedIPRange().Comments,
				Description:  expectedIPRange().Description,
				Tenant:       expectedIPRange().Tenant,
			},
		}

		tenancyListInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
		tenancyListOutput := &tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{
				Results: []*netboxModels.Tenant{
					{
						ID:   tenantId,
						Name: &tenantName,
						Slug: &tenantName,
					},
				},
			},
		}

		mockIpam.EXPECT().IpamIPRangesList(listInput, nil).Return(listOutput, nil).AnyTimes()
		// use gomock.Any() because there where issues with comparing the pointers in the request
		mockIpam.EXPECT().IpamIPRangesUpdate(gomock.Any(), nil).Return(updateOutput, nil).AnyTimes()
		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(tenancyListOutput, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIpam,
			Tenancy: mockTenancy,
		}

		actual, err := client.ReserveOrUpdateIpRange(&models.IpRange{
			StartAddress: startAddress,
			EndAddress:   endAddress,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error return
		AssertNil(t, err)
		assert.Equal(t, expectedIPRange().ID, actual.ID)
		assert.Equal(t, expectedIPRange().Comments, actual.Comments)
		assert.Equal(t, expectedIPRange().Description, actual.Description)
		assert.Equal(t, expectedIPRange().StartAddress, actual.StartAddress)
		assert.Equal(t, expectedIPRange().EndAddress, actual.EndAddress)
		assert.Equal(t, expectedIPRange().Tenant.ID, actual.Tenant.ID)
		assert.Equal(t, expectedIPRange().Tenant.Name, actual.Tenant.Name)
		assert.Equal(t, expectedIPRange().Tenant.Slug, actual.Tenant.Slug)
	})

	t.Run("Delete ip range", func(t *testing.T) {
		// ip range mock input
		deleteInput := ipam.NewIpamIPRangesDeleteParams().WithID(expectedIPRange().ID)
		// ip range mock output
		deleteOutput := &ipam.IpamIPRangesDeleteNoContent{}

		mockIpam.EXPECT().IpamIPRangesDelete(deleteInput, nil).Return(deleteOutput, nil).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIpam,
			Tenancy: mockTenancy,
		}

		err := client.DeleteIpRange(expectedIPRange().ID)

		// assert error return
		AssertNil(t, err)
	})

	t.Run("Delete ip range, ignore 404 error", func(t *testing.T) {

		// ip range mock input
		deleteInput := ipam.NewIpamIPRangesDeleteParams().WithID(expectedIPRange().ID)
		// ip range mock output
		// ip range mock output
		deleteOutput := &ipam.IpamIPRangesDeleteNoContent{}

		mockIpam.EXPECT().IpamIPRangesDelete(deleteInput, nil).Return(deleteOutput, ipam.NewIpamIPRangesDeleteDefault(404)).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIpam,
			Tenancy: mockTenancy,
		}

		err := client.DeleteIpRange(expectedIPRange().ID)

		// assert error return
		// assert error return
		AssertNil(t, err)
	})

	t.Run("Delete ip range, return non 404 errors", func(t *testing.T) {

		// ip range mock input
		deleteInput := ipam.NewIpamIPRangesDeleteParams().WithID(expectedIPRange().ID)
		// ip range mock output
		// ip range mock output
		deleteOutput := &ipam.IpamIPRangesDeleteNoContent{}

		mockIpam.EXPECT().IpamIPRangesDelete(deleteInput, nil).Return(deleteOutput, ipam.NewIpamIPRangesDeleteDefault(400)).AnyTimes()

		// init client
		client := &NetboxClient{
			Ipam:    mockIpam,
			Tenancy: mockTenancy,
		}

		err := client.DeleteIpRange(expectedIPRange().ID)

		// assert error return
		// assert error return
		AssertError(t, err, "Failed to delete ip range from Netbox: [DELETE /ipam/ip-ranges/{id}/][400] ipam_ip-ranges_delete default  <nil>")
	})
}
