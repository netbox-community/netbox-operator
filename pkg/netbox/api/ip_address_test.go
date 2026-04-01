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
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Comments    = "reservation for test comments"
	Status      = "active"
	Description = "reservation for test description"
	IpAddressId = int64(4)
)

func TestIPAddress(t *testing.T) {
	ctrl := gomock.NewController(t, gomock.WithOverridableExpectations())
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

	expectedHash := "fioaf9289rjfhaeuih"

	customFields := map[string]interface{}{
		config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
	}

	// example output IP address
	expectedIPAddress := func() *netboxModels.IPAddress {
		lastUpdated := strfmt.DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		return &netboxModels.IPAddress{
			ID:           int64(1),
			Address:      &ipAddress,
			Display:      ipAddress,
			Comments:     Comments,
			Description:  Description,
			CustomFields: customFields,
			Tenant: &netboxModels.NestedTenant{
				ID: tenantId,
			},
			Status: &netboxModels.IPAddressStatus{
				Label: &Label,
				Value: &Value,
			},
			LastUpdated: &lastUpdated,
		}
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

	ipAddressModel := func(restorationHash string) *models.IPAddress {
		model := &models.IPAddress{
			IpAddress: ipAddress,
		}
		if restorationHash != "" {
			if model.Metadata == nil {
				model.Metadata = &models.NetboxMetadata{}
			}
			model.Metadata.Custom = map[string]string{
				config.GetOperatorConfig().NetboxRestorationHashFieldName: restorationHash,
			}
		}
		return model
	}

	t.Run("Retrieve Existing static IP Address", func(t *testing.T) {

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
		clientV3 := &NetboxClientV3{
			Ipam:    mockIPAddress,
			Tenancy: mockPrefixTenancy,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		actual, err := compositeClient.getIpAddress(&models.IPAddress{
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

	t.Run("Retrieve Non Existing Static IP Address", func(t *testing.T) {

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
		clientV3 := &NetboxClientV3{
			Ipam:    mockIPAddress,
			Tenancy: mockPrefixTenancy,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		res, err := compositeClient.getIpAddress(&models.IPAddress{
			IpAddress: ipAddress,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error return
		AssertNil(t, err)
		assert.Zero(t, res.Payload.Count)
	})

	t.Run("create Static IP Address", func(t *testing.T) {

		// ip address mock input
		input := ipam.NewIpamIPAddressesCreateParams().WithDefaults().WithData(writeableAddress())
		// ip address mock output
		output := &ipam.IpamIPAddressesCreateCreated{
			Payload: expectedIPAddress(),
		}

		mockIPAddress.EXPECT().IpamIPAddressesCreate(input, nil).Return(output, nil)

		// init client
		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		ipaddress, err := compositeClient.createIpAddress(writeableAddress())

		// assert error return
		AssertNil(t, err)

		// assert address elements
		AssertIpAddress(t, writeableAddress(), ipaddress)

	})

	t.Run("update of Static IP Address", func(t *testing.T) {

		input := ipam.NewIpamIPAddressesUpdateParams().WithDefaults().WithData(writeableAddress()).WithID(IpAddressId)

		output := &ipam.IpamIPAddressesUpdateOK{
			Payload: expectedIPAddress(),
		}

		mockIPAddress.EXPECT().IpamIPAddressesUpdate(input, nil).Return(output, nil)

		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		ipaddress, isUpToDate, err := compositeClient.updateIpAddress(IpAddressId, writeableAddress())

		// assertion for errors
		AssertNil(t, err)
		assert.False(t, isUpToDate, "expected update to not skip update")

		// assert address properties
		AssertIpAddress(t, writeableAddress(), ipaddress)
	})

	t.Run("delete IP Address", func(t *testing.T) {

		input := ipam.NewIpamIPAddressesDeleteParams().WithID(IpAddressId)
		output := &ipam.IpamIPAddressesDeleteNoContent{}

		mockIPAddress.EXPECT().IpamIPAddressesDelete(input, nil).Return(output, nil)

		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		err := compositeClient.DeleteIpAddress(IpAddressId)
		AssertNil(t, err)
	})

	t.Run("check without hash", func(t *testing.T) {
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:          expectedIPAddress().ID,
						Address:     expectedIPAddress().Address,
						Display:     expectedIPAddress().Display,
						LastUpdated: expectedIPAddress().LastUpdated,
					}},
			},
		}

		outputUpdate := &ipam.IpamIPAddressesUpdateOK{
			Payload: expectedIPAddress(),
		}

		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		// use gomock.Any() because the input contains a pointer
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(outputUpdate, nil)

		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		ipAddressModel := ipAddressModel("")
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(ipAddressModel, &netboxv1.IpAddress{})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected result when update is performed")
		assert.False(t, isUpToDate, "expected update to be performed")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, result.Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, result.Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, result.Tenant.Slug)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("check with hash", func(t *testing.T) {
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:           expectedIPAddress().ID,
						Address:      expectedIPAddress().Address,
						Display:      expectedIPAddress().Display,
						CustomFields: expectedIPAddress().CustomFields,
						LastUpdated:  expectedIPAddress().LastUpdated,
					}},
			},
		}

		outputUpdate := &ipam.IpamIPAddressesUpdateOK{
			Payload: expectedIPAddress(),
		}

		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		// use gomock.Any() because the input contains a pointer
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(outputUpdate, nil)

		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		ipAddressModel := ipAddressModel(expectedHash)
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(ipAddressModel, &netboxv1.IpAddress{})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected result when update is performed")
		assert.False(t, isUpToDate, "expected update to be performed")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, result.Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, result.Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, result.Tenant.Slug)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("skip update when LastUpdated matches and Condition is Ready and Generation matches (no hash)", func(t *testing.T) {
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{ID: expectedIPAddress().ID, Address: expectedIPAddress().Address, LastUpdated: expectedIPAddress().LastUpdated},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		lastUpdatedV1 := metav1.NewTime(time.Time(*expectedIPAddress().LastUpdated))
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(&models.IPAddress{IpAddress: ipAddress}, &netboxv1.IpAddress{
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1,
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "True", ObservedGeneration: 0},
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected existing NetBox IP when LastUpdated matches and Condition is Ready and Generation matches")
		assert.True(t, isUpToDate, "expected skip update when LastUpdated matches and Condition is Ready and Generation matches")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("update when Condition is not Ready (no hash)", func(t *testing.T) {
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{ID: expectedIPAddress().ID, Address: expectedIPAddress().Address, LastUpdated: expectedIPAddress().LastUpdated},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(
			&ipam.IpamIPAddressesUpdateOK{Payload: expectedIPAddress()}, nil)

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		lastUpdatedV1 := metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(&models.IPAddress{IpAddress: ipAddress}, &netboxv1.IpAddress{
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1,
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "True", ObservedGeneration: 0},
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result)
		assert.False(t, isUpToDate)
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("update when LastUpdated differs (no hash)", func(t *testing.T) {
		lastUpdatedV1 := metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{ID: expectedIPAddress().ID, Address: expectedIPAddress().Address, LastUpdated: expectedIPAddress().LastUpdated},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(
			&ipam.IpamIPAddressesUpdateOK{Payload: expectedIPAddress()}, nil)

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(&models.IPAddress{IpAddress: ipAddress}, &netboxv1.IpAddress{
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1, // different from NetBox
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "True", ObservedGeneration: 0},
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected update when LastUpdated differs")
		assert.False(t, isUpToDate, "expected update when LastUpdated differs")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, result.Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, result.Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, result.Tenant.Slug)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("update when Generation differs (no hash)", func(t *testing.T) {
		lastUpdated := strfmt.DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{ID: expectedIPAddress().ID, Address: expectedIPAddress().Address, LastUpdated: &lastUpdated},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(
			&ipam.IpamIPAddressesUpdateOK{Payload: expectedIPAddress()}, nil)

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		lastUpdatedV1 := metav1.NewTime(time.Time(lastUpdated))
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(&models.IPAddress{IpAddress: ipAddress}, &netboxv1.IpAddress{
			ObjectMeta: metav1.ObjectMeta{Generation: 2}, // Generation 2
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1,
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "True", ObservedGeneration: 1}, // Generation 1
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected update when Generation differs")
		assert.False(t, isUpToDate, "expected update when Generation differs")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, result.Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, result.Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, result.Tenant.Slug)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("skip update when LastUpdated matches and Condition is Ready and Generation matches (with hash)", func(t *testing.T) {
		lastUpdated := strfmt.DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:           expectedIPAddress().ID,
						Address:      expectedIPAddress().Address,
						LastUpdated:  &lastUpdated,
						CustomFields: expectedIPAddress().CustomFields, // contains expectedHash
					},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		lastUpdatedV1 := metav1.NewTime(time.Time(lastUpdated))
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(ipAddressModel(expectedHash), &netboxv1.IpAddress{
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1,
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "True", ObservedGeneration: 0},
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected existing NetBox IP when LastUpdated matches and Condition is Ready and Generation matches (with hash)")
		assert.True(t, isUpToDate, "expected skip update when LastUpdated matches and Condition is Ready and Generation matches (with hash)")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("update when Condition is not Ready (with hash)", func(t *testing.T) {
		lastUpdated := strfmt.DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:           expectedIPAddress().ID,
						Address:      expectedIPAddress().Address,
						LastUpdated:  &lastUpdated,
						CustomFields: expectedIPAddress().CustomFields,
					},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(
			&ipam.IpamIPAddressesUpdateOK{Payload: expectedIPAddress()}, nil)

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		lastUpdatedV1 := metav1.NewTime(time.Time(lastUpdated))
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(ipAddressModel(expectedHash), &netboxv1.IpAddress{
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1,
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "False", ObservedGeneration: 0}, // not ready
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected update when Condition is not Ready (with hash)")
		assert.False(t, isUpToDate, "expected update when Condition is not Ready (with hash)")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, result.Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, result.Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, result.Tenant.Slug)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("update when LastUpdated differs (with hash)", func(t *testing.T) {
		lastUpdatedNetBox := strfmt.DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		lastUpdatedV1 := metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:           expectedIPAddress().ID,
						Address:      expectedIPAddress().Address,
						LastUpdated:  &lastUpdatedNetBox,
						CustomFields: expectedIPAddress().CustomFields,
					},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(
			&ipam.IpamIPAddressesUpdateOK{Payload: expectedIPAddress()}, nil)

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(ipAddressModel(expectedHash), &netboxv1.IpAddress{
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1,
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "True", ObservedGeneration: 0},
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected update when LastUpdated differs")
		assert.False(t, isUpToDate, "expected update when LastUpdated differs")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, result.Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, result.Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, result.Tenant.Slug)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("update when Generation differs (with hash)", func(t *testing.T) {
		lastUpdated := strfmt.DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:           expectedIPAddress().ID,
						Address:      expectedIPAddress().Address,
						CustomFields: expectedIPAddress().CustomFields,
						LastUpdated:  &lastUpdated,
					},
				},
			},
		}
		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()
		mockIPAddress.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Return(
			&ipam.IpamIPAddressesUpdateOK{Payload: expectedIPAddress()}, nil)

		clientV3 := &NetboxClientV3{Ipam: mockIPAddress}
		compositeClient := &NetboxCompositeClient{clientV3: clientV3}

		lastUpdatedV1 := metav1.NewTime(time.Time(lastUpdated))
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(ipAddressModel(expectedHash), &netboxv1.IpAddress{
			ObjectMeta: metav1.ObjectMeta{Generation: 2}, // Generation 2
			Status: netboxv1.IpAddressStatus{
				LastUpdated: &lastUpdatedV1,
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: "True", ObservedGeneration: 1}, // Generation 1
				},
			},
		})
		AssertNil(t, err)
		assert.NotNil(t, result, "expected update when LastUpdated differs")
		assert.False(t, isUpToDate, "expected update when Generation differs (with hash)")
		assert.Equal(t, expectedIPAddress().ID, result.ID)
		assert.Equal(t, expectedIPAddress().Comments, result.Comments)
		assert.Equal(t, expectedIPAddress().Description, result.Description)
		assert.Equal(t, expectedIPAddress().Display, result.Display)
		assert.Equal(t, expectedIPAddress().Address, result.Address)
		assert.Equal(t, expectedIPAddress().Tenant.ID, result.Tenant.ID)
		assert.Equal(t, expectedIPAddress().Tenant.Name, result.Tenant.Name)
		assert.Equal(t, expectedIPAddress().Tenant.Slug, result.Tenant.Slug)
		assert.Equal(t, expectedIPAddress().CustomFields, result.CustomFields)
		assert.Equal(t, expectedIPAddress().Status.Label, result.Status.Label)
		assert.Equal(t, expectedIPAddress().Status.Value, result.Status.Value)
		assert.Equal(t, expectedIPAddress().LastUpdated, result.LastUpdated)
	})

	t.Run("Check ReserveOrUpdate with hash mismatch", func(t *testing.T) {
		inputList := ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)
		outputList := &ipam.IpamIPAddressesListOK{
			Payload: &ipam.IpamIPAddressesListOKBody{
				Results: []*netboxModels.IPAddress{
					{
						ID:           expectedIPAddress().ID,
						Address:      expectedIPAddress().Address,
						Display:      expectedIPAddress().Display,
						CustomFields: expectedIPAddress().CustomFields,
						LastUpdated:  expectedIPAddress().LastUpdated,
					}},
			},
		}

		mockIPAddress.EXPECT().IpamIPAddressesList(inputList, nil).Return(outputList, nil).AnyTimes()

		clientV3 := &NetboxClientV3{
			Ipam: mockIPAddress,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: clientV3,
		}

		expectedHash := "iwfohs7v82fe9w0"
		ipAddressModel := ipAddressModel(expectedHash)
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpAddress(ipAddressModel, &netboxv1.IpAddress{})
		AssertError(t, err, "restoration hash mismatch, assigned ip address 10.112.140.0")
		assert.Nil(t, result)
		assert.True(t, isUpToDate)
	})
}
