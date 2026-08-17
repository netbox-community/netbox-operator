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

	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
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

const VlanId = int32(9)

func TestVlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := "vlan-sample"
	vid := int32(100)
	tenantId := int64(2)
	tenantName := "Tenant1"
	siteId := int64(3)
	siteName := "Site1"
	comments := Comments
	description := Description

	expectedTenant := v4client.NewBriefTenant(int32(tenantId), "", "", tenantName, "")
	expectedSite := v4client.NewBriefSite(int32(siteId), "", "", siteName, siteName)
	expectedLastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expectedVlan := func() v4client.VLAN {
		lastUpdated := expectedLastUpdated
		vlan := v4client.VLAN{
			Id:          VlanId,
			Name:        name,
			Vid:         vid,
			Comments:    &comments,
			Description: &description,
			Tenant:      *v4client.NewNullableBriefTenant(expectedTenant),
			Site:        *v4client.NewNullableBriefSite(expectedSite),
			LastUpdated: *v4client.NewNullableTime(&lastUpdated),
		}
		return vlan
	}

	t.Run("reserve new vlan", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
		mockCreateRequest := mock_interfaces.NewMockIpamVlansCreateRequest(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{Results: []v4client.VLAN{}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().IpamVlansCreate(gomock.Any()).Return(mockCreateRequest)
		mockCreateRequest.EXPECT().WritableVLANRequest(gomock.Any()).Return(mockCreateRequest)

		expectedResp := expectedVlan()
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

		dcimListInput := dcim.NewDcimSitesListParams().WithName(&siteName)
		dcimListOutput := &dcim.DcimSitesListOK{
			Payload: &dcim.DcimSitesListOKBody{
				Results: []*netboxModels.Site{
					{ID: siteId, Name: &siteName, Slug: &siteName},
				},
			},
		}
		mockDcim.EXPECT().DcimSitesList(dcimListInput, nil).Return(dcimListOutput, nil).AnyTimes()

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy, Dcim: mockDcim},
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateVlan(context.TODO(),
			&models.Vlan{
				Name: name,
				Vid:  vid,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
					Site:   siteName,
				},
			}, &netboxv1.Vlan{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, VlanId, actual.Id)
		assert.Equal(t, name, actual.Name)
	})

	t.Run("restoration hash mismatch", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{Results: []v4client.VLAN{
			{
				CustomFields: map[string]interface{}{"netboxOperatorRestorationHash": "abc"},
				LastUpdated:  *v4client.NewNullableTime(&expectedLastUpdated),
			},
		}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		expectedHash := "ffjrep8b29fdaikb"
		result, isUpToDate, err := compositeClient.ReserveOrUpdateVlan(
			context.TODO(),
			&models.Vlan{
				Name: name,
				Vid:  vid,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
					},
				},
			}, &netboxv1.Vlan{})

		AssertError(t, err, "restoration hash mismatch, assigned vlan vid 100")
		assert.False(t, isUpToDate)
		assert.Nil(t, result)
	})

	t.Run("update existing vlan", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamVlansUpdateRequest(ctrl)

		existing := expectedVlan()
		existing.CustomFields = map[string]interface{}{"netboxOperatorRestorationHash": "abc"}

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{Results: []v4client.VLAN{existing}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().IpamVlansUpdate(gomock.Any(), VlanId).Return(mockUpdateRequest)
		mockUpdateRequest.EXPECT().WritableVLANRequest(gomock.Any()).Return(mockUpdateRequest)

		updated := expectedVlan()
		mockUpdateRequest.EXPECT().Execute().Return(&updated, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateVlan(
			context.TODO(),
			&models.Vlan{Name: name, Vid: vid},
			&netboxv1.Vlan{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, VlanId, actual.Id)
	})

	t.Run("skip update when up to date", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlansListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANList{Results: []v4client.VLAN{expectedVlan()}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		lastUpdatedV1 := metav1.NewTime(*expectedVlan().LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateVlan(
			context.TODO(),
			&models.Vlan{Name: name, Vid: vid},
			&netboxv1.Vlan{
				Status: netboxv1.VlanStatus{
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

	t.Run("delete vlan", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamVlansDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansDestroy(gomock.Any(), VlanId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteVlan(context.TODO(), VlanId)
		AssertNil(t, err)
	})

	t.Run("delete vlan ignore 404 error", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamVlansDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansDestroy(gomock.Any(), VlanId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteVlan(context.TODO(), VlanId)
		AssertNil(t, err)
	})

	t.Run("delete vlan returns non 404 errors", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamVlansDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlansDestroy(gomock.Any(), VlanId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 400, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteVlan(context.TODO(), VlanId)
		AssertError(t, err, "failed to delete vlan from netbox: status 400, body: ")
	})
}
