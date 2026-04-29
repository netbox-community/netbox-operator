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

func TestPrefix_ListExistingPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
	mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

	//tenant mock input
	tenantName := "Tenant1"
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
	expectedTenant := v4client.NewBriefTenant(tenantId, "", "", tenantName, tenantSlug)

	expectedLastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

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
				LastUpdated: *v4client.NewNullableTime(&expectedLastUpdated),
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
				Tenant:      tenantName,
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
	assert.Equal(t, tenantName, actual.Results[0].Tenant.Get().Name)
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
	tenantName := "Tenant1"

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
				Tenant: tenantName,
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// tenant mock input
	tenantName := "Tenant1"
	tenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	// tenant mock output
	tenantOutputId := int64(1)
	tenantOutputSlug := "tenant1"
	tenantListRequestOutput := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantOutputId,
					Name: &tenantName,
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

	expectedStatusValue := v4client.PrefixStatusValue("active")
	expectedPrefix := func() v4client.Prefix {
		lastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		return v4client.Prefix{
			Id:          prefixId,
			Display:     prefix,
			Prefix:      prefix,
			Status:      &v4client.PrefixStatus{Value: &expectedStatusValue},
			LastUpdated: *v4client.NewNullableTime(&lastUpdated),
		}
	}

	t.Run("reserve with tenant and site (v4 NetBox client)", func(t *testing.T) {
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

		//tenant mock input
		tenantId := int32(tenantOutputId)

		expectedTenant := v4client.NewBriefTenant(tenantId, "", "", tenantName, tenantOutputSlug)

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
				Tenant:      tenantName,
			},
		}

		_, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&prefixModel, &netboxv1.Prefix{})
		// skip assertion on returned values as the payload of IpamPrefixesCreate() is returned
		// without manipulation by the code
		assert.Nil(t, err)
		assert.False(t, isUpToDate)
	})

	t.Run("update without tenant and site (v4 netbox client)", func(t *testing.T) {
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
			LastUpdated: expectedPrefix().LastUpdated,
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

		_, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&prefixModel, &netboxv1.Prefix{})

		// skip assertion on returned values as the payload of IpamPrefixesUpdate() is returned
		// without manipulation by the code
		assert.Nil(t, err)
		assert.False(t, isUpToDate)
	})

	t.Run("restoration hash mismatch", func(t *testing.T) {
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
					LastUpdated: expectedPrefix().LastUpdated,
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

		result, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(context.TODO(), &prefixModel, &netboxv1.Prefix{})
		// skip assertion on returned values as the payload of IpamPrefixesCreate() is returned
		// without manipulation by the code
		AssertError(t, err, "restoration hash mismatch, assigned prefix 10.112.140.0/24")
		assert.False(t, isUpToDate)
		assert.Nil(t, result)
	})

	t.Run("skip update when LastUpdated matches and Condition is Ready and Generation matches (no hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

		expected := expectedPrefix()

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		lastUpdatedV1 := metav1.NewTime(*expected.LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{Prefix: prefix},
			&netboxv1.Prefix{
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "True", ObservedGeneration: 0}},
				},
			},
		)

		AssertNil(t, err)
		assert.True(t, isUpToDate)
		assert.Nil(t, actual)
	})

	t.Run("update when Condition is not Ready (no hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamPrefixesUpdateRequest(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.2.0")
		_ = mockStatusRequest

		expected := expectedPrefix()

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamPrefixesUpdate(gomock.Any(), prefixId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI, StatusAPI: mockStatusAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		lastUpdatedV1 := metav1.NewTime(*expected.LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{Prefix: prefix},
			&netboxv1.Prefix{
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "False", ObservedGeneration: 0}},
				},
			},
		)

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.Prefix, actual.Prefix)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when LastUpdated differs (no hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamPrefixesUpdateRequest(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.2.0")
		_ = mockStatusRequest

		expected := expectedPrefix()
		lastUpdatedV1 := metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamPrefixesUpdate(gomock.Any(), prefixId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI, StatusAPI: mockStatusAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{Prefix: prefix},
			&netboxv1.Prefix{
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "True", ObservedGeneration: 0}},
				},
			},
		)

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.Prefix, actual.Prefix)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when Generation differs (no hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamPrefixesUpdateRequest(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.2.0")
		_ = mockStatusRequest

		expected := expectedPrefix()

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamPrefixesUpdate(gomock.Any(), prefixId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI, StatusAPI: mockStatusAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		lastUpdatedV1 := metav1.NewTime(*expected.LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{Prefix: prefix},
			&netboxv1.Prefix{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "True", ObservedGeneration: 1}},
				},
			},
		)

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.Prefix, actual.Prefix)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("skip update when LastUpdated matches and Condition is Ready and Generation matches (with hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)

		expectedHash := "some_hash_value"
		expected := expectedPrefix()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		lastUpdatedV1 := metav1.NewTime(*expected.LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{
				Prefix:   prefix,
				Metadata: &models.NetboxMetadata{Custom: map[string]string{config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash}},
			},
			&netboxv1.Prefix{
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "True", ObservedGeneration: 0}},
				},
			},
		)

		AssertNil(t, err)
		assert.True(t, isUpToDate)
		assert.Nil(t, actual)
	})

	t.Run("update when Condition is not Ready (with hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamPrefixesUpdateRequest(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.2.0")
		_ = mockStatusRequest

		expectedHash := "some_hash_value"
		expected := expectedPrefix()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamPrefixesUpdate(gomock.Any(), prefixId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI, StatusAPI: mockStatusAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		lastUpdatedV1 := metav1.NewTime(*expected.LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{
				Prefix:   prefix,
				Metadata: &models.NetboxMetadata{Custom: map[string]string{config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash}},
			},
			&netboxv1.Prefix{
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "False", ObservedGeneration: 0}},
				},
			},
		)

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.Prefix, actual.Prefix)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when LastUpdated differs (with hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamPrefixesUpdateRequest(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.2.0")
		_ = mockStatusRequest

		expectedHash := "some_hash_value"
		expected := expectedPrefix()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}
		lastUpdatedV1 := metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamPrefixesUpdate(gomock.Any(), prefixId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI, StatusAPI: mockStatusAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{
				Prefix:   prefix,
				Metadata: &models.NetboxMetadata{Custom: map[string]string{config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash}},
			},
			&netboxv1.Prefix{
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "True", ObservedGeneration: 0}},
				},
			},
		)

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.Prefix, actual.Prefix)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})

	t.Run("update when Generation differs (with hash)", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamPrefixesListRequest(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamPrefixesUpdateRequest(ctrl)
		mockStatusAPI, mockStatusRequest := GetNetBoxVersionMock(ctrl, "4.2.0")
		_ = mockStatusRequest

		expectedHash := "some_hash_value"
		expected := expectedPrefix()
		expected.CustomFields = map[string]interface{}{
			config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash,
		}

		mockIpamAPI.EXPECT().
			IpamPrefixesList(gomock.Any()).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Prefix([]string{prefix}).
			Return(mockListRequest)

		mockListRequest.EXPECT().
			Execute().
			Return(&v4client.PaginatedPrefixList{Results: []v4client.Prefix{expected}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		mockIpamAPI.EXPECT().
			IpamPrefixesUpdate(gomock.Any(), prefixId).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			WritablePrefixRequest(gomock.Any()).
			Return(mockUpdateRequest)

		mockUpdateRequest.EXPECT().
			Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI, StatusAPI: mockStatusAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		lastUpdatedV1 := metav1.NewTime(*expected.LastUpdated.Get())
		actual, isUpToDate, err := compositeClient.ReserveOrUpdatePrefix(
			context.TODO(),
			&models.Prefix{
				Prefix:   prefix,
				Metadata: &models.NetboxMetadata{Custom: map[string]string{config.GetOperatorConfig().NetboxRestorationHashFieldName: expectedHash}},
			},
			&netboxv1.Prefix{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Status: netboxv1.PrefixStatus{
					LastUpdated: lastUpdatedV1,
					Conditions:  []metav1.Condition{{Type: "Ready", Status: "True", ObservedGeneration: 1}},
				},
			},
		)

		AssertNil(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, expected.Id, actual.Id)
		assert.Equal(t, expected.Prefix, actual.Prefix)
		assert.Equal(t, expected.Status, actual.Status)
		assert.Equal(t, expected.LastUpdated, actual.LastUpdated)
	})
}
