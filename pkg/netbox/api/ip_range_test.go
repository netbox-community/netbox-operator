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

	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	IpRangeId = int32(4)
)

func TestIpRange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	startAddress := "10.0.0.1"
	endAddress := "10.0.0.10"
	tenantId := int64(2)
	tenantName := "Tenant1"
	Label := "Status"
	Value := "active"
	comment := Comments
	description := Description
	markPopulatedTrue := true

	expectedTenant := v4client.NewBriefTenant(int32(tenantId), "", "", tenantName, "")
	expectedStatus := v4client.NewIPRangeStatus()
	expectedStatus.SetValue(v4client.IPRangeStatusValue(Value))
	expectedStatus.SetLabel(v4client.IPRangeStatusLabel(Label))

	// Create expected response
	expectedIPRange := func() v4client.IPRange {
		return v4client.IPRange{

			Id:            IpRangeId,
			StartAddress:  startAddress,
			EndAddress:    endAddress,
			Comments:      &comment,
			Description:   &description,
			Tenant:        *v4client.NewNullableBriefTenant(expectedTenant),
			Status:        expectedStatus,
			MarkPopulated: &markPopulatedTrue,
		}
	}

	t.Run("Retrieve Existing IP Range.", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		// Setup expectations
		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expectedIPRange()}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		// init client
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		actual, err := compositeClient.getIpRange(context.TODO(), &models.IpRange{
			StartAddress: startAddress,
			EndAddress:   endAddress,
			Metadata: &models.NetboxMetadata{
				Tenant: tenantName,
			},
		})

		// assert error return
		AssertNil(t, err)
		assert.Equal(t, expectedIPRange().Id, actual.Results[0].Id)
		assert.Equal(t, expectedIPRange().Comments, actual.Results[0].Comments)
		assert.Equal(t, expectedIPRange().Description, actual.Results[0].Description)
		assert.Equal(t, expectedIPRange().StartAddress, actual.Results[0].StartAddress)
		assert.Equal(t, expectedIPRange().EndAddress, actual.Results[0].EndAddress)
		assert.Equal(t, expectedIPRange().Tenant.Get().Id, actual.Results[0].Tenant.Get().Id)
		assert.Equal(t, expectedIPRange().Tenant.Get().Name, actual.Results[0].Tenant.Get().Name)
		assert.Equal(t, expectedIPRange().Tenant.Get().Slug, actual.Results[0].Tenant.Get().Slug)
		assert.Equal(t, expectedIPRange().MarkPopulated, actual.Results[0].MarkPopulated)
	})

	t.Run("ReserveOrUpdate, reserve new ip range", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockCreateRequest := mock_interfaces.NewMockIpamIpRangesCreateRequest(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)

		// Setup expectations
		mockIpamAPI.EXPECT().
			IpamIpRangesCreate(gomock.Any()).
			Return(mockCreateRequest)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		// List should return empty results to trigger create path
		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockCreateRequest.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockCreateRequest)

		// Create expected response
		expectedResp := &v4client.IPRange{
			Id:            IpRangeId,
			StartAddress:  startAddress,
			EndAddress:    endAddress,
			Comments:      &comment,
			Description:   &description,
			MarkPopulated: &markPopulatedTrue,
			Tenant:        expectedIPRange().Tenant,
		}

		mockCreateRequest.EXPECT().
			Execute().
			Return(expectedResp, &http.Response{StatusCode: 201, Body: http.NoBody}, nil)

		// Create client with mock
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
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

		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(tenancyListOutput, nil).AnyTimes()
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
			clientV3: clientV3,
		}

		// Create request
		ipRangeRequest := v4client.NewWritableIPRangeRequest(startAddress, endAddress)
		ipRangeRequest.SetStatus("active")

		// Test
		actual, err := compositeClient.ReserveOrUpdateIpRange(context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
				},
			})

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, actual)
		assert.Equal(t, IpRangeId, actual.Id)
		assert.Equal(t, expectedIPRange().Comments, actual.Comments)
		assert.Equal(t, expectedIPRange().Description, actual.Description)
		assert.Equal(t, startAddress, actual.StartAddress)
		assert.Equal(t, endAddress, actual.EndAddress)
		assert.Equal(t, expectedIPRange().Tenant.Get().Id, actual.Tenant.Get().Id)
		assert.Equal(t, expectedIPRange().Tenant.Get().Name, actual.Tenant.Get().Name)
		assert.Equal(t, expectedIPRange().Tenant.Get().Slug, actual.Tenant.Get().Slug)
		assert.Equal(t, expectedIPRange().MarkPopulated, actual.MarkPopulated)
	})

	t.Run("ReserveOrUpdate, restoration hash mismatch", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		// List should return empty results to trigger create path
		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{
				{
					CustomFields: map[string]interface{}{"netboxOperatorRestorationHash": "abc"},
				},
			}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		startAddress := "10.0.0.1"
		endAddress := "10.0.0.10"

		// Create client with mock
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
			clientV3: clientV3,
		}

		// Create request
		ipRangeRequest := v4client.NewWritableIPRangeRequest(startAddress, endAddress)
		ipRangeRequest.SetStatus("active")
		ipRangeRequest.SetDescription("Updated range")

		// Test
		expectedHash := "ffjrep8b29fdaikb"
		_, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			})

		// Assert
		AssertError(t, err, "restoration hash mismatch, assigned ip range 10.0.0.1-10.0.0.10")
	})

	t.Run("ReserveOrUpdate, update existing ip range", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamIpRangesUpdateRequest(ctrl)

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

		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(tenancyListOutput, nil).AnyTimes()

		ipRangeId := int32(1)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{
				{
					Id:            ipRangeId,
					CustomFields:  map[string]interface{}{"netboxOperatorRestorationHash": "abc"},
					Comments:      &comment,
					Description:   &description,
					MarkPopulated: &markPopulatedTrue,
				},
			}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		// Setup expectations
		mockIpamAPI.EXPECT().
			IpamIpRangesUpdate(gomock.Any(), ipRangeId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockUpdateRequest)

		// Create expected response
		expectedResp := &v4client.IPRange{
			Id:            ipRangeId,
			StartAddress:  startAddress,
			EndAddress:    endAddress,
			Comments:      &comment,
			Description:   &description,
			Tenant:        *v4client.NewNullableBriefTenant(expectedTenant),
			MarkPopulated: &markPopulatedTrue,
		}

		mockUpdateRequest.EXPECT().
			Execute().
			Return(expectedResp, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		// Create client with mock
		clientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
		}
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
			clientV3: clientV3,
		}

		// Test
		actual, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
				},
			})

		// Assert
		AssertNil(t, err)
		assert.Equal(t, ipRangeId, actual.Id)
		assert.Equal(t, expectedIPRange().Comments, actual.Comments)
		assert.Equal(t, expectedIPRange().Description, actual.Description)
		assert.Equal(t, expectedIPRange().StartAddress, actual.StartAddress)
		assert.Equal(t, expectedIPRange().EndAddress, actual.EndAddress)
		assert.Equal(t, expectedIPRange().Tenant.Get().Id, actual.Tenant.Get().Id)
		assert.Equal(t, expectedIPRange().Tenant.Get().Name, actual.Tenant.Get().Name)
		assert.Equal(t, expectedIPRange().Tenant.Get().Slug, actual.Tenant.Get().Slug)
		assert.Equal(t, expectedIPRange().MarkPopulated, actual.MarkPopulated)
	})

	t.Run("Delete ip range", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamIpRangesDestroyRequest(ctrl)

		ipRangeId := int32(1)

		// Setup mock expectations
		mockIpamAPI.EXPECT().
			IpamIpRangesDestroy(gomock.Any(), ipRangeId).
			Return(mockDestroyRequest)

		mockDestroyRequest.EXPECT().
			Execute().
			Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil)

		// init client with mock
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		err := compositeClient.DeleteIpRange(context.TODO(), ipRangeId)

		// assert error return
		AssertNil(t, err)
	})

	t.Run("Delete ip range, ignore 404 error", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamIpRangesDestroyRequest(ctrl)

		ipRangeId := int32(1)

		// Setup mock expectations
		mockIpamAPI.EXPECT().
			IpamIpRangesDestroy(gomock.Any(), ipRangeId).
			Return(mockDestroyRequest)

		mockDestroyRequest.EXPECT().
			Execute().
			Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil)

		// init client with mock
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		err := compositeClient.DeleteIpRange(context.TODO(), ipRangeId)

		// assert error return
		AssertNil(t, err)
	})

	t.Run("Delete ip range, return non 404 errors", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamIpRangesDestroyRequest(ctrl)

		ipRangeId := int32(1)

		// Setup mock expectations
		mockIpamAPI.EXPECT().
			IpamIpRangesDestroy(gomock.Any(), ipRangeId).
			Return(mockDestroyRequest)

		mockDestroyRequest.EXPECT().
			Execute().
			Return(&http.Response{StatusCode: 400, Body: http.NoBody}, nil)

		// init client with mock
		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		err := compositeClient.DeleteIpRange(context.TODO(), ipRangeId)

		// assert error return
		AssertError(t, err, "failed to delete ip range from netbox: status 400, body: ")
	})
}
