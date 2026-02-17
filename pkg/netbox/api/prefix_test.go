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

	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPrefix_ListExistingPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

	//tenant mock input
	tenant := "Tenant1"
	tenantId := int32(1)
	tenantSlug := "tenant1"

	//prefix mock input
	prefix := "10.112.140.0/24"

	//prefix mock output
	prefixId := int32(4)
	siteId := int32(2)
	scopeType := "dcim.site"
	comments := "blabla"
	description := "very useful prefix"
	expectedTenant := v4client.NewBriefTenant(tenantId, "", "", tenant, tenantSlug)

	prefixListOutput := v4client.PaginatedPrefixList{
		Results: []v4client.Prefix{
			{
				Id:          prefixId,
				Comments:    &comments,
				Description: &description,
				Display:     prefix,
				Prefix:      prefix,
				ScopeType:   *v4client.NewNullableString(&scopeType),
				ScopeId:     *v4client.NewNullableInt32(&siteId),
				Tenant:      *v4client.NewNullableBriefTenant(expectedTenant),
			},
		},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{prefix}).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Execute().
		Return(&prefixListOutput, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV4: clientV4,
	}

	actual, err := compositeClient.getPrefix(
		context.TODO(),
		&models.Prefix{
			Prefix: prefix,
			Metadata: &models.NetboxMetadata{
				Tenant:      tenant,
				Comments:    comments,
				Description: description,
			},
		})

	assert.Nil(t, err)
	assert.Equal(t, prefixId, actual.Results[0].Id)
	assert.Equal(t, comments, *actual.Results[0].Comments)
	assert.Equal(t, description, *actual.Results[0].Description)
	assert.Equal(t, prefix, actual.Results[0].Display)
	assert.Equal(t, prefix, actual.Results[0].Prefix)
	assert.Equal(t, tenant, actual.Results[0].Tenant.Get().Name)
	assert.Equal(t, tenantId, actual.Results[0].Tenant.Get().Id)
	assert.Equal(t, tenantSlug, actual.Results[0].Tenant.Get().Slug)
	assert.Equal(t, "dcim.site", *actual.Results[0].ScopeType.Get())
	assert.Equal(t, siteId, *actual.Results[0].ScopeId.Get())
}

func TestPrefix_ListNonExistingPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

	//prefix mock input
	prefix := "10.112.140.0/24"

	//tenant mock input
	tenant := "tenant1"

	//prefix mock output
	prefixListOutput := v4client.PaginatedPrefixList{
		Results: []v4client.Prefix{},
	}

	mockIpamAPI.EXPECT().
		IpamPrefixesList(gomock.Any()).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Prefix([]string{prefix}).
		Return(mockListRequest)

	mockListRequest.EXPECT().
		Execute().
		Return(&prefixListOutput, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	compositeClient := &NetboxCompositeClient{
		clientV4: clientV4,
	}

	actual, err := compositeClient.getPrefix(
		context.TODO(),
		&models.Prefix{
			Prefix: prefix,
			Metadata: &models.NetboxMetadata{
				Tenant: tenant,
			},
		})

	assert.Nil(t, err)
	assert.Len(t, actual.Results, 0)
}

func TestPrefix_DeletePrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockDestroyRequest := mock_interfaces.NewMockIpamPrefixesDestroyRequest(ctrl)

	//prefix mock input
	prefixId := int32(4)
	//prefix mock output

	mockIpamAPI.EXPECT().
		IpamPrefixesDestroy(gomock.Any(), prefixId).
		Return(mockDestroyRequest)

	mockDestroyRequest.EXPECT().
		Execute().
		Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil)

	clientV4 := &NetboxClientV4{
		IpamAPI: mockIpamAPI,
	}
	comositeClient := &NetboxCompositeClient{
		clientV4: clientV4,
	}

	err := comositeClient.DeletePrefix(context.TODO(), prefixId)
	assert.Nil(t, err)
}

func TestPrefix_ReserveOrUpdate(t *testing.T) {
	// tenant mock input
	tenant := "Tenant1"
	tenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&tenant)

	// tenant mock output
	tenantOutputId := int64(1)
	tenantOutputSlug := "tenant1"
	tenantListRequestOutput := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantOutputId,
					Name: &tenant,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	// site mock input
	site := "Site3"
	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&site)

	// site mock output
	siteOutputId := int64(3)
	siteOutputSlug := "site3"
	siteListRequestOutput := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{
				{
					ID:   siteOutputId,
					Name: &site,
					Slug: &siteOutputSlug,
				},
			},
		},
	}

	// prefix mock input
	prefix := "10.112.140.0/24"

	//prefix mock output
	prefixId := int32(4)
	comments := "blabla"
	description := "very useful prefix"

	emptyPrefixListOutput := &v4client.PaginatedPrefixList{
		Results: []v4client.Prefix{},
	}

	t.Run("reserve with tenant and site (v4 NetBox client)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockCreateRequest := mock_interfaces.NewMockIpamPrefixesCreateRequest(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.4.10")
		_ = mockStatusRequest

		completeComment := comments + warningComment
		completeDescription := description + warningComment
		scopeType := "dcim.site"
		scopeId := int32(siteOutputId)
		tenantId := int32(1)
		tenantSlug := "tenant1"
		expectedTenant := v4client.NewBriefTenant(tenantId, "", "", tenant, tenantSlug)

		//prefix mock output
		createPrefixOutput := &v4client.Prefix{
			Id:          int32(1),
			Comments:    &completeComment,
			Description: &completeDescription,
			Display:     prefix,
			Prefix:      prefix,
			ScopeType:   *v4client.NewNullableString(&scopeType),
			ScopeId:     *v4client.NewNullableInt32(&scopeId),
			Tenant:      *v4client.NewNullableBriefTenant(expectedTenant),
		}

		mockTenancy.EXPECT().TenancyTenantsList(tenantListRequestInput, nil).Return(tenantListRequestOutput, nil).AnyTimes()
		mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(siteListRequestOutput, nil).AnyTimes()

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(emptyPrefixListOutput, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamPrefixesCreate(gomock.Any()).
			Return(mockCreateRequest)

		mockCreateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockCreateRequest)

		mockCreateRequest.EXPECT().
			Execute().
			Return(createPrefixOutput, &http.Response{StatusCode: 201, Body: http.NoBody}, nil)

		netboxClientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
			Dcim:    mockDcim,
		}
		netboxClientV4 := &NetboxClientV4{
			IpamAPI:   mockIpamAPI,
			StatusAPI: mockStatusAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: netboxClientV3,
			clientV4: netboxClientV4,
		}

		prefixModel := models.Prefix{
			Prefix: prefix,
			Metadata: &models.NetboxMetadata{
				Comments:    comments,
				Description: description,
				Site:        site,
				Custom:      make(map[string]string),
				Tenant:      tenant,
			},
		}

		_, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&prefixModel)
		// skip assertion on returned values as the payload of IpamPrefixesCreate() is returned
		// without manipulation by the code
		assert.Nil(t, err)
	})

	t.Run("update without tenant and site (v4 netbox client)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamPrefixesUpdateRequest(ctrl)
		mockIpam := mock_interfaces.NewMockIpamInterface(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.2.0")
		_ = mockStatusRequest

		prefixOutput := v4client.Prefix{
			Id:          int32(4),
			Comments:    &comments,
			Description: &description,
			Display:     prefix,
			Prefix:      prefix,
			CustomFields: map[string]interface{}{
				config.GetOperatorConfig().NetboxRestorationHashFieldName: "wrongHash",
			},
		}

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{prefixOutput}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		// Setup expectations
		mockIpamAPI.EXPECT().
			IpamPrefixesUpdate(gomock.Any(), prefixId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			Execute().
			Return(&prefixOutput, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		netboxClientV3 := &NetboxClientV3{
			Tenancy: mockTenancy,
			Ipam:    mockIpam,
		}
		netboxClientV4 := &NetboxClientV4{
			IpamAPI:   mockIpamAPI,
			StatusAPI: mockStatusAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: netboxClientV3,
			clientV4: netboxClientV4,
		}

		prefixModel := models.Prefix{
			Prefix: prefix,
			Metadata: &models.NetboxMetadata{
				Comments:    comments,
				Description: description,
			},
		}

		_, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&prefixModel)
		// skip assertion on returned values as the payload of IpamPrefixesUpdate() is returned
		// without manipulation by the code
		assert.Nil(t, err)
	})

	t.Run("restoration hash mismatch", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		prefixListOutput := &v4client.PaginatedPrefixList{
			Results: []v4client.Prefix{
				{
					Id:          int32(5),
					Comments:    &comments,
					Description: &description,
					Display:     prefix,
					Prefix:      prefix,
					CustomFields: map[string]interface{}{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: "wrongHash",
					},
				},
			},
		}

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(prefixListOutput, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		netboxClientV3 := &NetboxClientV3{}
		netboxClientV4 := &NetboxClientV4{
			IpamAPI: mockIpamAPI,
		}
		compositeClient := &NetboxCompositeClient{
			clientV3: netboxClientV3,
			clientV4: netboxClientV4,
		}

		expectedHash := "jfioaw0e9gh"
		prefixModel := models.Prefix{
			Prefix: prefix,
			Metadata: &models.NetboxMetadata{
				Custom: map[string]string{config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash},
			},
		}

		_, err := compositeClient.ReserveOrUpdatePrefix(context.TODO(), &prefixModel)
		// skip assertion on returned values as the payload of IpamPrefixesCreate() is returned
		// without manipulation by the code
		AssertError(t, err, "restoration hash mismatch, assigned prefix 10.112.140.0/24")
	})
}
