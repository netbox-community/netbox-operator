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
	"time"

	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	comments := Comments
	description := Description
	markPopulatedTrue := true

	expectedTenant := v4client.NewBriefTenant(int32(tenantId), "", "", tenantName, "")
	expectedStatus := v4client.NewIPRangeStatus()
	expectedStatus.SetValue(v4client.IPRangeStatusValue(Value))
	expectedStatus.SetLabel(v4client.IPRangeStatusLabel(Label))

	expectedLastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Create expected response
	expectedIPRange := func() v4client.IPRange {
		lastUpdated := expectedLastUpdated
		return v4client.IPRange{
			Id:            IpRangeId,
			StartAddress:  startAddress,
			EndAddress:    endAddress,
			Comments:      &comments,
			Description:   &description,
			Tenant:        *v4client.NewNullableBriefTenant(expectedTenant),
			Status:        expectedStatus,
			MarkPopulated: &markPopulatedTrue,
			LastUpdated:   *v4client.NewNullableTime(&lastUpdated),
		}
	}

	t.Run("get existing IP Range", func(t *testing.T) {
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

	t.Run("reserve new ip range", func(t *testing.T) {
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
			Comments:      &comments,
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
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
				},
			}, &netboxv1.IpRange{})

		// Assert
		assert.NoError(t, err)
		assert.False(t, isUpToDate)
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

	t.Run("restoration hash mismatch", func(t *testing.T) {
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
					LastUpdated:  *v4client.NewNullableTime(&expectedLastUpdated),
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
		result, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.IpRange{})

		// Assert
		AssertError(t, err, "restoration hash mismatch, assigned ip range 10.0.0.1-10.0.0.10")
		assert.True(t, isUpToDate)
		assert.Nil(t, result)
	})

	t.Run("update existing ip range", func(t *testing.T) {
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
					Comments:      &comments,
					Description:   &description,
					MarkPopulated: &markPopulatedTrue,
					LastUpdated:   *v4client.NewNullableTime(&expectedLastUpdated),
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
			Comments:      &comments,
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
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
				},
			}, &netboxv1.IpRange{})

		// Assert
		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
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

	t.Run("skip update when LastUpdated matches and Condition is Ready and Generation matches (no hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
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

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expectedIPRange()}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		lastUpdatedV1 := metav1.NewTime(*expectedIPRange().LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
			}, &netboxv1.IpRange{
				Status: netboxv1.IpRangeStatus{
					LastUpdated: lastUpdatedV1,
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "True", ObservedGeneration: 0},
					},
				},
			})
		AssertNil(t, err)
		assert.True(t, isUpToDate)
		assert.Nil(t, actual)
	})

	t.Run("update when Condition is not Ready (no hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		mockIpamIpRangesUpdate := mock_interfaces.NewMockIpamIpRangesUpdateRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		ipRangeId := int32(4)
		expectedHash := "some_hash_value"

		mockIpamAPI.EXPECT().
			IpamIpRangesUpdate(gomock.Any(), ipRangeId).
			Return(mockIpamIpRangesUpdate)

		expected := expectedIPRange()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamIpRangesUpdate.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockIpamIpRangesUpdate)

		mockIpamIpRangesUpdate.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		lastUpdatedV1 := metav1.NewTime(*expectedIPRange().LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
			}, &netboxv1.IpRange{
				Status: netboxv1.IpRangeStatus{
					LastUpdated: lastUpdatedV1,
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "False", ObservedGeneration: 0},
					},
				},
			})
		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual, "expected update when Condition is not Ready")
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.StartAddress, actual.StartAddress)
		assert.Equal(t, expected.EndAddress, actual.EndAddress)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.MarkPopulated, actual.MarkPopulated)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when Generation differs (no hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		mockIpamIpRangesUpdate := mock_interfaces.NewMockIpamIpRangesUpdateRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		ipRangeId := int32(4)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expectedIPRange()}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamIpRangesUpdate(gomock.Any(), ipRangeId).
			Return(mockIpamIpRangesUpdate)

		mockIpamIpRangesUpdate.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockIpamIpRangesUpdate)

		expected := expectedIPRange()

		mockIpamIpRangesUpdate.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
			}, &netboxv1.IpRange{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: netboxv1.IpRangeStatus{
					LastUpdated: metav1.NewTime(*expected.LastUpdated.Get()),
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "True", ObservedGeneration: 1},
					},
				},
			})
		AssertNil(t, err)
		assert.NotNil(t, actual, "expected update when Generation differs")
		assert.False(t, isUpToDate)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.StartAddress, actual.StartAddress)
		assert.Equal(t, expected.EndAddress, actual.EndAddress)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.MarkPopulated, actual.MarkPopulated)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when LastUpdated differs (no hash)", func(t *testing.T) {
		lastUpdatedV1 := metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		mockIpamIpRangesUpdate := mock_interfaces.NewMockIpamIpRangesUpdateRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		ipRangeId := int32(4)

		mockIpamAPI.EXPECT().
			IpamIpRangesUpdate(gomock.Any(), ipRangeId).
			Return(mockIpamIpRangesUpdate)

		expected := expectedIPRange()
		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamIpRangesUpdate.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockIpamIpRangesUpdate)

		mockIpamIpRangesUpdate.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
			}, &netboxv1.IpRange{
				Status: netboxv1.IpRangeStatus{
					LastUpdated: lastUpdatedV1,
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "True", ObservedGeneration: 0},
					},
				},
			})
		AssertNil(t, err)
		assert.NotNil(t, actual, "expected update when Condition is not Ready")
		assert.False(t, isUpToDate)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.StartAddress, actual.StartAddress)
		assert.Equal(t, expected.EndAddress, actual.EndAddress)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.MarkPopulated, actual.MarkPopulated)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("skip update when LastUpdated matches and Condition is Ready and Generation matches (with hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
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

		expectedHash := "some_hash_value"
		expected := expectedIPRange()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}
		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		lastUpdatedV1 := metav1.NewTime(*expectedIPRange().LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.IpRange{
				Status: netboxv1.IpRangeStatus{
					LastUpdated: lastUpdatedV1,
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "True", ObservedGeneration: 0},
					},
				},
			})
		AssertNil(t, err)
		assert.True(t, isUpToDate)
		assert.Nil(t, actual)
	})

	t.Run("update when Condition is not Ready (with hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		mockIpamIpRangesUpdate := mock_interfaces.NewMockIpamIpRangesUpdateRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		ipRangeId := int32(4)

		mockIpamAPI.EXPECT().
			IpamIpRangesUpdate(gomock.Any(), ipRangeId).
			Return(mockIpamIpRangesUpdate)

		expectedHash := "some_hash_value"
		expected := expectedIPRange()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		req := v4client.NewWritableIPRangeRequest(startAddress, endAddress)
		req.SetStatus("active")
		req.SetMarkPopulated(true)
		mockIpamIpRangesUpdate.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockIpamIpRangesUpdate)

		mockIpamIpRangesUpdate.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		lastUpdatedV1 := metav1.NewTime(*expected.LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.IpRange{
				Status: netboxv1.IpRangeStatus{
					LastUpdated: lastUpdatedV1,
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "False", ObservedGeneration: 0},
					},
				},
			})
		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual, "expected update when Condition is not Ready")
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.StartAddress, actual.StartAddress)
		assert.Equal(t, expected.EndAddress, actual.EndAddress)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.MarkPopulated, actual.MarkPopulated)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when Generation differs (with hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		mockIpamIpRangesUpdate := mock_interfaces.NewMockIpamIpRangesUpdateRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		ipRangeId := int32(4)

		expectedHash := "some_hash_value"
		expected := expectedIPRange()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamIpRangesUpdate(gomock.Any(), ipRangeId).
			Return(mockIpamIpRangesUpdate)

		mockIpamIpRangesUpdate.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockIpamIpRangesUpdate)

		mockIpamIpRangesUpdate.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.IpRange{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: netboxv1.IpRangeStatus{
					LastUpdated: metav1.NewTime(*expected.LastUpdated.Get()),
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "True", ObservedGeneration: 1},
					},
				},
			})
		AssertNil(t, err)
		assert.NotNil(t, actual, "expected update when Generation differs")
		assert.False(t, isUpToDate)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.StartAddress, actual.StartAddress)
		assert.Equal(t, expected.EndAddress, actual.EndAddress)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.MarkPopulated, actual.MarkPopulated)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when LastUpdated differs (no hash)", func(t *testing.T) {
		lastUpdatedV1 := metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamIpRangesListRequest(ctrl)
		mockIpamIpRangesUpdate := mock_interfaces.NewMockIpamIpRangesUpdateRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamIpRangesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			StartAddress([]string{startAddress}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			EndAddress([]string{endAddress}).
			Return(mockListRequest)

		ipRangeId := int32(4)

		mockIpamAPI.EXPECT().
			IpamIpRangesUpdate(gomock.Any(), ipRangeId).
			Return(mockIpamIpRangesUpdate)

		expectedHash := "some_hash_value"
		expected := expectedIPRange()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedIPRangeList{Results: []v4client.IPRange{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamIpRangesUpdate.EXPECT().
			WritableIPRangeRequest(gomock.Any()).
			Return(mockIpamIpRangesUpdate)

		mockIpamIpRangesUpdate.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV4: clientV4,
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateIpRange(
			context.TODO(),
			&models.IpRange{
				StartAddress: startAddress,
				EndAddress:   endAddress,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.IpRange{
				Status: netboxv1.IpRangeStatus{
					LastUpdated: lastUpdatedV1,
					Conditions: []metav1.Condition{
						{Type: "Ready", Status: "True", ObservedGeneration: 0},
					},
				},
			})
		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual, "expected update when Condition is not Ready")
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.StartAddress, actual.StartAddress)
		assert.Equal(t, expected.EndAddress, actual.EndAddress)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.MarkPopulated, actual.MarkPopulated)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("delete ip range", func(t *testing.T) {
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

	t.Run("delete ip range ignore 404 error", func(t *testing.T) {
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

	t.Run("delete ip range return non 404 errors", func(t *testing.T) {
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
