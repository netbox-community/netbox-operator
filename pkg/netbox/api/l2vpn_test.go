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

const L2VPNId = int32(7)

func TestSlugify(t *testing.T) {
	assert.Equal(t, "my-l2vpn-1", slugify("my l2vpn 1"))
	assert.Equal(t, "leading-trailing", slugify("--leading-trailing--"))
	assert.Equal(t, "l2vpn", slugify(""))
	assert.Equal(t, "l2vpn", slugify("###"))
}

func TestL2VPN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := "l2vpn-sample"
	l2vpnType := "vxlan-evpn"
	identifier := int64(5000123)
	tenantId := int64(2)
	tenantName := "Tenant1"
	comments := Comments
	description := Description

	expectedTenant := v4client.NewBriefTenant(int32(tenantId), "", "", tenantName, "")
	expectedLastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expectedL2VPN := func() v4client.L2VPN {
		lastUpdated := expectedLastUpdated
		l2vpn := v4client.L2VPN{
			Id:          L2VPNId,
			Name:        name,
			Slug:        slugify(name),
			Comments:    &comments,
			Description: &description,
			Tenant:      *v4client.NewNullableBriefTenant(expectedTenant),
			LastUpdated: *v4client.NewNullableTime(&lastUpdated),
		}
		l2vpn.Identifier = *v4client.NewNullableInt64(&identifier)
		return l2vpn
	}

	t.Run("reserve new l2vpn", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockCreateRequest := mock_interfaces.NewMockVpnL2vpnsCreateRequest(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Identifier([]int32{int32(identifier)}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{Results: []v4client.L2VPN{}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockVpnAPI.EXPECT().VpnL2vpnsCreate(gomock.Any()).Return(mockCreateRequest)
		mockCreateRequest.EXPECT().WritableL2VPNRequest(gomock.Any()).Return(mockCreateRequest)

		expectedResp := expectedL2VPN()
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
			clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy},
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateL2VPN(context.TODO(),
			&models.L2VPN{
				Name:       name,
				Type:       l2vpnType,
				Identifier: identifier,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
				},
			}, &netboxv1.L2VPN{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, L2VPNId, actual.Id)
		assert.Equal(t, name, actual.Name)
	})

	t.Run("restoration hash mismatch", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Identifier([]int32{int32(identifier)}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{Results: []v4client.L2VPN{
			{
				CustomFields: map[string]interface{}{"netboxOperatorRestorationHash": "abc"},
				LastUpdated:  *v4client.NewNullableTime(&expectedLastUpdated),
			},
		}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		expectedHash := "ffjrep8b29fdaikb"
		result, isUpToDate, err := compositeClient.ReserveOrUpdateL2VPN(
			context.TODO(),
			&models.L2VPN{
				Name:       name,
				Type:       l2vpnType,
				Identifier: identifier,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.L2VPN{})

		AssertError(t, err, "restoration hash mismatch, assigned l2vpn identifier 5000123")
		assert.False(t, isUpToDate)
		assert.Nil(t, result)
	})

	t.Run("update existing l2vpn", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockVpnL2vpnsUpdateRequest(ctrl)

		existing := expectedL2VPN()
		existing.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "abc"}

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Identifier([]int32{int32(identifier)}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{Results: []v4client.L2VPN{existing}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockVpnAPI.EXPECT().VpnL2vpnsUpdate(gomock.Any(), L2VPNId).Return(mockUpdateRequest)
		mockUpdateRequest.EXPECT().WritableL2VPNRequest(gomock.Any()).Return(mockUpdateRequest)

		updated := expectedL2VPN()
		mockUpdateRequest.EXPECT().Execute().Return(&updated, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateL2VPN(
			context.TODO(),
			&models.L2VPN{Name: name, Type: l2vpnType, Identifier: identifier},
			&netboxv1.L2VPN{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, L2VPNId, actual.Id)
	})

	t.Run("skip update when up to date", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockVpnL2vpnsListRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Identifier([]int32{int32(identifier)}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedL2VPNList{Results: []v4client.L2VPN{expectedL2VPN()}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		lastUpdatedV1 := metav1.NewTime(*expectedL2VPN().LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateL2VPN(
			context.TODO(),
			&models.L2VPN{Name: name, Type: l2vpnType, Identifier: identifier},
			&netboxv1.L2VPN{
				Status: netboxv1.L2VPNStatus{
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

	t.Run("delete l2vpn", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockVpnL2vpnsDestroyRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsDestroy(gomock.Any(), L2VPNId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		err := compositeClient.DeleteL2VPN(context.TODO(), int64(L2VPNId))
		AssertNil(t, err)
	})

	t.Run("delete l2vpn ignore 404 error", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockVpnL2vpnsDestroyRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsDestroy(gomock.Any(), L2VPNId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		err := compositeClient.DeleteL2VPN(context.TODO(), int64(L2VPNId))
		AssertNil(t, err)
	})

	t.Run("delete l2vpn returns non 404 errors", func(t *testing.T) {
		mockVpnAPI := mock_interfaces.NewMockVpnAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockVpnL2vpnsDestroyRequest(ctrl)

		mockVpnAPI.EXPECT().VpnL2vpnsDestroy(gomock.Any(), L2VPNId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 400, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{VpnAPI: mockVpnAPI}}

		err := compositeClient.DeleteL2VPN(context.TODO(), int64(L2VPNId))
		AssertError(t, err, "failed to delete l2vpn from netbox: status 400, body: ")
	})
}
