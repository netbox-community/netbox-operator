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
	"github.com/netbox-community/netbox-operator/pkg/netbox/interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// expectAsnListPages queues one IpamAsnsList expectation per given page, asserting that
// the pagination parameters are set correctly.
func expectAsnListPages(ctrl *gomock.Controller, mockIpamAPI *mock_interfaces.MockIpamAPI, count int32, pages ...[]v4client.ASN) {
	offset := int32(0)
	for _, page := range pages {
		req := mock_interfaces.NewMockIpamAsnsListRequest(ctrl)
		mockIpamAPI.EXPECT().IpamAsnsList(gomock.Any()).Return(req)
		req.EXPECT().Limit(int32(asnListPageSize)).Return(req)
		req.EXPECT().Offset(offset).Return(req)
		req.EXPECT().Execute().
			Return(&v4client.PaginatedASNList{Count: count, Results: page}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)
		offset += int32(len(page))
	}
}

// expectAsnRangeLookup queues the ASN Range lookup by name performed before claiming.
func expectAsnRangeLookup(ctrl *gomock.Controller, mockIpamAPI *mock_interfaces.MockIpamAPI, name string, results []v4client.ASNRange) {
	req := mock_interfaces.NewMockIpamAsnRangesListRequest(ctrl)
	mockIpamAPI.EXPECT().IpamAsnRangesList(gomock.Any()).Return(req)
	req.EXPECT().Limit(int32(asnListPageSize)).Return(req)
	req.EXPECT().Offset(int32(0)).Return(req)
	req.EXPECT().Name([]string{name}).Return(req)
	req.EXPECT().Execute().
		Return(&v4client.PaginatedASNRangeList{Count: int32(len(results)), Results: results}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)
}

func TestAsnClaim(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asnRangeName := "private-range"
	asnRangeId := int32(5)
	asnValue := int64(65001)
	description := "test description"
	tenantId := int64(2)
	tenantName := "Tenant1"
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName

	asnRanges := []v4client.ASNRange{{Id: asnRangeId, Name: asnRangeName, Start: 64512, End: 65534}}

	t.Run("restore existing ASN by hash", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		hash := "abc123hash"

		expectAsnListPages(ctrl, mockIpamAPI, 1, []v4client.ASN{
			{Id: AsnId, Asn: asnValue, CustomFields: map[string]interface{}{restorationHashKey: hash}},
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingAsnByHash(context.TODO(), hash)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, asnValue, result.Asn)
		assert.Equal(t, int64(AsnId), result.Id)
	})

	t.Run("restore existing ASN by hash - match on a later page", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		hash := "abc123hash"

		// Fill the first two pages so the match is only reachable via pagination.
		fillerPage := func(startId int32) []v4client.ASN {
			page := make([]v4client.ASN, asnListPageSize)
			for i := range page {
				page[i] = v4client.ASN{
					Id:           startId + int32(i),
					Asn:          int64(startId) + int64(i),
					CustomFields: map[string]interface{}{restorationHashKey: "other-hash"},
				}
			}
			return page
		}

		expectAsnListPages(ctrl, mockIpamAPI, 2*asnListPageSize+1,
			fillerPage(1000),
			fillerPage(2000),
			[]v4client.ASN{{Id: AsnId, Asn: asnValue, CustomFields: map[string]interface{}{restorationHashKey: hash}}},
		)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingAsnByHash(context.TODO(), hash)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, asnValue, result.Asn)
		assert.Equal(t, int64(AsnId), result.Id)
	})

	t.Run("restore existing ASN by hash - not found", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectAsnListPages(ctrl, mockIpamAPI, 1, []v4client.ASN{
			{Id: AsnId, Asn: asnValue, CustomFields: map[string]interface{}{restorationHashKey: "different-hash"}},
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingAsnByHash(context.TODO(), "nonexistent-hash")

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("restore existing ASN by hash - more than one match", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		hash := "abc123hash"

		expectAsnListPages(ctrl, mockIpamAPI, 2, []v4client.ASN{
			{Id: AsnId, Asn: asnValue, CustomFields: map[string]interface{}{restorationHashKey: hash}},
			{Id: AsnId + 1, Asn: asnValue + 1, CustomFields: map[string]interface{}{restorationHashKey: hash}},
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingAsnByHash(context.TODO(), hash)

		assert.ErrorContains(t, err, "incorrect number of restoration results")
		assert.Nil(t, result)
	})

	t.Run("restore existing ASN by hash - netbox error", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamAsnsListRequest(ctrl)

		mockIpamAPI.EXPECT().IpamAsnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(int32(asnListPageSize)).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().
			Return(nil, &http.Response{StatusCode: 500, Body: http.NoBody}, assert.AnError)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.RestoreExistingAsnByHash(context.TODO(), "abc123hash")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("get available ASN by claim", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockAvailableRequest := mock_interfaces.NewMockIpamAsnRangesAvailableAsnsCreateRequest(ctrl)
		hash := "abc123hash"

		tenancyListInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(&tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{
				Results: []*netboxModels.Tenant{{ID: tenantId, Name: &tenantName, Slug: &tenantName}},
			},
		}, nil)

		expectAsnRangeLookup(ctrl, mockIpamAPI, asnRangeName, asnRanges)

		mockIpamAPI.EXPECT().
			IpamAsnRangesAvailableAsnsCreate(gomock.Any(), asnRangeId).
			Return(mockAvailableRequest)

		// The allocation request must already carry the restoration hash and the tenant,
		// otherwise the ASN NetBox creates would be unidentifiable.
		var sentRequests []v4client.ASNRequest
		mockAvailableRequest.EXPECT().
			ASNRequest(gomock.Any()).
			DoAndReturn(func(reqs []v4client.ASNRequest) interfaces.IpamAsnRangesAvailableAsnsCreateRequest {
				sentRequests = reqs
				return mockAvailableRequest
			})

		mockAvailableRequest.EXPECT().
			Execute().
			Return([]v4client.ASN{{Id: AsnId, Asn: asnValue}}, &http.Response{StatusCode: 201, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy},
		}

		result, err := compositeClient.GetAvailableAsnByClaim(context.TODO(), &models.ASNClaim{
			ParentAsnRange: asnRangeName,
			Metadata: &models.NetboxMetadata{
				Description: description,
				Tenant:      tenantName,
				Custom:      map[string]string{restorationHashKey: hash},
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, asnValue, result.Asn)
		assert.Equal(t, int64(AsnId), result.Id)

		assert.Len(t, sentRequests, 1)
		assert.Equal(t, hash, sentRequests[0].CustomFields[restorationHashKey])
		assert.Equal(t, TruncateDescription(description), *sentRequests[0].Description)
		assert.True(t, sentRequests[0].Tenant.IsSet())
		assert.Equal(t, int32(tenantId), *sentRequests[0].Tenant.Get().Int32)
	})

	t.Run("get available ASN by claim - range not found", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectAsnRangeLookup(ctrl, mockIpamAPI, asnRangeName, []v4client.ASNRange{})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableAsnByClaim(context.TODO(), &models.ASNClaim{
			ParentAsnRange: asnRangeName,
			Metadata:       &models.NetboxMetadata{Description: description},
		})

		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrAsnRangeExhausted)
		assert.Nil(t, result)
	})

	t.Run("get available ASN by claim - range exhausted", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockAvailableRequest := mock_interfaces.NewMockIpamAsnRangesAvailableAsnsCreateRequest(ctrl)

		expectAsnRangeLookup(ctrl, mockIpamAPI, asnRangeName, asnRanges)

		mockIpamAPI.EXPECT().
			IpamAsnRangesAvailableAsnsCreate(gomock.Any(), asnRangeId).
			Return(mockAvailableRequest)
		mockAvailableRequest.EXPECT().ASNRequest(gomock.Any()).Return(mockAvailableRequest)
		mockAvailableRequest.EXPECT().Execute().
			Return(nil, &http.Response{StatusCode: 409, Body: http.NoBody}, assert.AnError)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableAsnByClaim(context.TODO(), &models.ASNClaim{
			ParentAsnRange: asnRangeName,
			Metadata:       &models.NetboxMetadata{Description: description},
		})

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAsnRangeExhausted)
		assert.Nil(t, result)
	})

	t.Run("get available ASN by claim - empty response is exhaustion", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockAvailableRequest := mock_interfaces.NewMockIpamAsnRangesAvailableAsnsCreateRequest(ctrl)

		expectAsnRangeLookup(ctrl, mockIpamAPI, asnRangeName, asnRanges)

		mockIpamAPI.EXPECT().
			IpamAsnRangesAvailableAsnsCreate(gomock.Any(), asnRangeId).
			Return(mockAvailableRequest)
		mockAvailableRequest.EXPECT().ASNRequest(gomock.Any()).Return(mockAvailableRequest)
		mockAvailableRequest.EXPECT().Execute().
			Return([]v4client.ASN{}, &http.Response{StatusCode: 201, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableAsnByClaim(context.TODO(), &models.ASNClaim{
			ParentAsnRange: asnRangeName,
			Metadata:       &models.NetboxMetadata{Description: description},
		})

		assert.ErrorIs(t, err, ErrAsnRangeExhausted)
		assert.Nil(t, result)
	})

	t.Run("get available ASN by claim - server error is not exhaustion", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockAvailableRequest := mock_interfaces.NewMockIpamAsnRangesAvailableAsnsCreateRequest(ctrl)

		expectAsnRangeLookup(ctrl, mockIpamAPI, asnRangeName, asnRanges)

		mockIpamAPI.EXPECT().
			IpamAsnRangesAvailableAsnsCreate(gomock.Any(), asnRangeId).
			Return(mockAvailableRequest)
		mockAvailableRequest.EXPECT().ASNRequest(gomock.Any()).Return(mockAvailableRequest)
		mockAvailableRequest.EXPECT().Execute().
			Return(nil, &http.Response{StatusCode: 500, Body: http.NoBody}, assert.AnError)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, err := compositeClient.GetAvailableAsnByClaim(context.TODO(), &models.ASNClaim{
			ParentAsnRange: asnRangeName,
			Metadata:       &models.NetboxMetadata{Description: description},
		})

		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrAsnRangeExhausted)
		assert.Nil(t, result)
	})
}
