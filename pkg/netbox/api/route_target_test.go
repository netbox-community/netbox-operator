/*
Copyright 2026 Swisscom (Schweiz) AG.

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

const RouteTargetId = int32(11)

func TestRouteTarget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := "65000:100"
	tenantId := int64(2)
	tenantName := "Tenant1"
	comments := Comments
	description := Description

	expectedTenant := v4client.NewBriefTenant(int32(tenantId), "", "", tenantName, "")
	expectedLastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expectedRouteTarget := func() v4client.RouteTarget {
		lastUpdated := expectedLastUpdated
		return v4client.RouteTarget{
			Id:          RouteTargetId,
			Name:        name,
			Comments:    &comments,
			Description: &description,
			Tenant:      *v4client.NewNullableBriefTenant(expectedTenant),
			LastUpdated: *v4client.NewNullableTime(&lastUpdated),
		}
	}

	t.Run("reserve new route target", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockCreateRequest := mock_interfaces.NewMockIpamRouteTargetsCreateRequest(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamRouteTargetsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedRouteTargetList{Results: []v4client.RouteTarget{}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().IpamRouteTargetsCreate(gomock.Any()).Return(mockCreateRequest)
		mockCreateRequest.EXPECT().RouteTargetRequest(gomock.Any()).DoAndReturn(func(req v4client.RouteTargetRequest) *mock_interfaces.MockIpamRouteTargetsCreateRequest {
			assert.Equal(t, name, req.Name)
			assert.Equal(t, int32(tenantId), *req.Tenant.Get().Int32)
			return mockCreateRequest
		})

		expectedResp := expectedRouteTarget()
		mockCreateRequest.EXPECT().Execute().Return(&expectedResp, &http.Response{StatusCode: 201, Body: http.NoBody}, nil)

		tenancyListInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
		tenancyListOutput := &tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{
				Results: []*netboxModels.Tenant{
					{ID: tenantId, Name: &tenantName, Slug: &tenantName},
				},
			},
		}
		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(tenancyListOutput, nil).AnyTimes()

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy},
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateRouteTarget(context.TODO(),
			&models.RouteTarget{
				Name: name,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
				},
			}, &netboxv1.RouteTarget{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, RouteTargetId, actual.Id)
		assert.Equal(t, name, actual.Name)
	})

	t.Run("restoration hash mismatch", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamRouteTargetsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedRouteTargetList{Results: []v4client.RouteTarget{
			{
				CustomFields: map[string]interface{}{"netboxOperatorRestorationHash": "abc"},
				LastUpdated:  *v4client.NewNullableTime(&expectedLastUpdated),
			},
		}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		expectedHash := "ffjrep8b29fdaikb"
		result, isUpToDate, err := compositeClient.ReserveOrUpdateRouteTarget(
			context.TODO(),
			&models.RouteTarget{
				Name: name,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.RouteTarget{})

		AssertError(t, err, "restoration hash mismatch, assigned route target 65000:100")
		assert.ErrorIs(t, err, ErrRestorationHashMismatch)
		assert.False(t, isUpToDate)
		assert.Nil(t, result)
	})

	t.Run("update existing route target", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamRouteTargetsListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamRouteTargetsUpdateRequest(ctrl)

		existing := expectedRouteTarget()
		existing.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "abc"}

		mockIpamAPI.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedRouteTargetList{Results: []v4client.RouteTarget{existing}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().IpamRouteTargetsUpdate(gomock.Any(), RouteTargetId).Return(mockUpdateRequest)
		mockUpdateRequest.EXPECT().RouteTargetRequest(gomock.Any()).Return(mockUpdateRequest)

		updated := expectedRouteTarget()
		mockUpdateRequest.EXPECT().Execute().Return(&updated, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateRouteTarget(
			context.TODO(),
			&models.RouteTarget{Name: name},
			&netboxv1.RouteTarget{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, RouteTargetId, actual.Id)
	})

	t.Run("skip update when up to date", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamRouteTargetsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedRouteTargetList{Results: []v4client.RouteTarget{expectedRouteTarget()}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		lastUpdatedV1 := metav1.NewTime(*expectedRouteTarget().LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateRouteTarget(
			context.TODO(),
			&models.RouteTarget{Name: name},
			&netboxv1.RouteTarget{
				Status: netboxv1.RouteTargetStatus{
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

	t.Run("list error is returned", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamRouteTargetsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(nil, &http.Response{StatusCode: 500, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateRouteTarget(
			context.TODO(),
			&models.RouteTarget{Name: name},
			&netboxv1.RouteTarget{})
		AssertError(t, err, "failed to fetch route target details: status 500, body: ")
		assert.False(t, isUpToDate)
		assert.Nil(t, actual)
	})

	t.Run("delete route target", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamRouteTargetsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsDestroy(gomock.Any(), RouteTargetId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteRouteTarget(context.TODO(), int64(RouteTargetId))
		AssertNil(t, err)
	})

	t.Run("delete route target ignore 404 error", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamRouteTargetsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsDestroy(gomock.Any(), RouteTargetId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteRouteTarget(context.TODO(), int64(RouteTargetId))
		AssertNil(t, err)
	})

	t.Run("delete route target returns non 404 errors", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamRouteTargetsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamRouteTargetsDestroy(gomock.Any(), RouteTargetId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 400, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteRouteTarget(context.TODO(), int64(RouteTargetId))
		AssertError(t, err, "failed to delete route target from netbox: status 400, body: ")
	})
}
