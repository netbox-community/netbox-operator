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
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const VlanGroupId = int32(11)

func TestSlugify(t *testing.T) {
	assert.Equal(t, "my-vlan-group-1", slugify("my vlan group 1"))
	assert.Equal(t, "leading-trailing", slugify("--leading-trailing--"))
	assert.Equal(t, "vlangroup", slugify(""))
	assert.Equal(t, "vlangroup", slugify("###"))
}

func TestVlanGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := "vlangroup-sample"
	tenantId := int64(2)
	tenantName := "Tenant1"
	siteId := int64(3)
	siteName := "Site1"
	description := Description

	expectedLastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expectedVlanGroup := func() v4client.VLANGroup {
		lastUpdated := expectedLastUpdated
		vlanGroup := v4client.VLANGroup{
			Id:          VlanGroupId,
			Name:        name,
			Slug:        slugify(name),
			Description: &description,
			LastUpdated: *v4client.NewNullableTime(&lastUpdated),
		}
		return vlanGroup
	}

	t.Run("reserve new vlan group", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
		mockCreateRequest := mock_interfaces.NewMockIpamVlanGroupsCreateRequest(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlanGroupsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlanGroupsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANGroupList{Results: []v4client.VLANGroup{}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().IpamVlanGroupsCreate(gomock.Any()).Return(mockCreateRequest)
		mockCreateRequest.EXPECT().VLANGroupRequest(gomock.Any()).Return(mockCreateRequest)

		expectedResp := expectedVlanGroup()
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

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateVlanGroup(context.TODO(),
			&models.VlanGroup{
				Name:          name,
				VidRangeStart: 100,
				VidRangeEnd:   199,
				Metadata: &models.NetboxMetadata{
					Tenant: tenantName,
					Site:   siteName,
				},
			}, &netboxv1.VlanGroup{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, VlanGroupId, actual.Id)
		assert.Equal(t, name, actual.Name)
	})

	t.Run("update existing vlan group", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlanGroupsListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamVlanGroupsUpdateRequest(ctrl)

		existing := expectedVlanGroup()

		mockIpamAPI.EXPECT().IpamVlanGroupsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANGroupList{Results: []v4client.VLANGroup{existing}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().IpamVlanGroupsUpdate(gomock.Any(), VlanGroupId).Return(mockUpdateRequest)
		mockUpdateRequest.EXPECT().VLANGroupRequest(gomock.Any()).Return(mockUpdateRequest)

		updated := expectedVlanGroup()
		mockUpdateRequest.EXPECT().Execute().Return(&updated, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateVlanGroup(
			context.TODO(),
			&models.VlanGroup{Name: name},
			&netboxv1.VlanGroup{})

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, VlanGroupId, actual.Id)
	})

	t.Run("skip update when up to date", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamVlanGroupsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlanGroupsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Name([]string{name}).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().Return(&v4client.PaginatedVLANGroupList{Results: []v4client.VLANGroup{expectedVlanGroup()}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		lastUpdatedV1 := metav1.NewTime(*expectedVlanGroup().LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateVlanGroup(
			context.TODO(),
			&models.VlanGroup{Name: name},
			&netboxv1.VlanGroup{
				Status: netboxv1.VlanGroupStatus{
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

	t.Run("delete vlan group", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamVlanGroupsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlanGroupsDestroy(gomock.Any(), VlanGroupId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteVlanGroup(context.TODO(), VlanGroupId)
		AssertNil(t, err)
	})

	t.Run("delete vlan group ignore 404 error", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamVlanGroupsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlanGroupsDestroy(gomock.Any(), VlanGroupId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteVlanGroup(context.TODO(), VlanGroupId)
		AssertNil(t, err)
	})

	t.Run("delete vlan group returns non 404 errors", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamVlanGroupsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().IpamVlanGroupsDestroy(gomock.Any(), VlanGroupId).Return(mockDestroyRequest)
		mockDestroyRequest.EXPECT().Execute().Return(&http.Response{StatusCode: 400, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteVlanGroup(context.TODO(), VlanGroupId)
		AssertError(t, err, "failed to delete vlan group from netbox: status 400, body: ")
	})
}
