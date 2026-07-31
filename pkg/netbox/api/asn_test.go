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
	"github.com/netbox-community/netbox-operator/pkg/netbox/interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AsnId = int32(10)

// expectAsnRangesListAll queues the paginated ASN Range listing used to resolve the RIR.
func expectAsnRangesListAll(ctrl *gomock.Controller, mockIpamAPI *mock_interfaces.MockIpamAPI, results []v4client.ASNRange) {
	req := mock_interfaces.NewMockIpamAsnRangesListRequest(ctrl)
	mockIpamAPI.EXPECT().IpamAsnRangesList(gomock.Any()).Return(req)
	req.EXPECT().Limit(int32(asnListPageSize)).Return(req)
	req.EXPECT().Offset(int32(0)).Return(req)
	req.EXPECT().Execute().
		Return(&v4client.PaginatedASNRangeList{Count: int32(len(results)), Results: results}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)
}

// expectAsnListByValue queues the single-page ASN lookup by the `asn=` filter.
func expectAsnListByValue(ctrl *gomock.Controller, mockIpamAPI *mock_interfaces.MockIpamAPI, asnValue int64, results []v4client.ASN) {
	req := mock_interfaces.NewMockIpamAsnsListRequest(ctrl)
	mockIpamAPI.EXPECT().IpamAsnsList(gomock.Any()).Return(req)
	req.EXPECT().Limit(int32(asnListPageSize)).Return(req)
	req.EXPECT().Offset(int32(0)).Return(req)
	req.EXPECT().Asn([]int32{int32(asnValue)}).Return(req)
	req.EXPECT().Execute().
		Return(&v4client.PaginatedASNList{Count: int32(len(results)), Results: results}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)
}

func TestAsn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	asnValue := int64(65001)
	tenantId := int64(2)
	tenantName := "Tenant1"
	comments := Comments
	description := Description
	rirId := int32(9)

	asnRanges := []v4client.ASNRange{{Id: 1, Start: 64512, End: 65534, Rir: v4client.BriefRIR{Id: rirId}}}

	expectedLastUpdated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expectedASN := func() v4client.ASN {
		return v4client.ASN{
			Id:           AsnId,
			Asn:          asnValue,
			Comments:     &comments,
			Description:  &description,
			CustomFields: map[string]interface{}{"example_field": "example_value"},
			LastUpdated:  *v4client.NewNullableTime(&expectedLastUpdated),
			Rir:          *v4client.NewNullableBriefRIR(&v4client.BriefRIR{Id: rirId}),
		}
	}

	tenantLookup := func(mockTenancy *mock_interfaces.MockTenancyInterface) {
		tenancyListInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
		mockTenancy.EXPECT().TenancyTenantsList(tenancyListInput, nil).Return(&tenancy.TenancyTenantsListOK{
			Payload: &tenancy.TenancyTenantsListOKBody{
				Results: []*netboxModels.Tenant{
					{ID: tenantId, Name: &tenantName, Slug: &tenantName},
				},
			},
		}, nil).AnyTimes()
	}

	t.Run("get existing ASN", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectAsnListByValue(ctrl, mockIpamAPI, asnValue, []v4client.ASN{expectedASN()})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getAsn(context.TODO(), &models.ASN{Asn: asnValue}, 0)

		AssertNil(t, err)
		assert.NotNil(t, actual)
		assert.Equal(t, expectedASN().Id, actual.Id)
		assert.Equal(t, expectedASN().Asn, actual.Asn)
		assert.Equal(t, expectedASN().Comments, actual.Comments)
		assert.Equal(t, expectedASN().Description, actual.Description)
	})

	t.Run("get existing ASN by netbox id", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockRetrieveRequest := mock_interfaces.NewMockIpamAsnsRetrieveRequest(ctrl)

		// Once the netbox id is known the `asn=` filter must not be used at all.
		mockIpamAPI.EXPECT().IpamAsnsRetrieve(gomock.Any(), AsnId).Return(mockRetrieveRequest)
		expected := expectedASN()
		mockRetrieveRequest.EXPECT().Execute().
			Return(&expected, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getAsn(context.TODO(), &models.ASN{Asn: asnValue}, int64(AsnId))

		AssertNil(t, err)
		assert.NotNil(t, actual)
		assert.Equal(t, AsnId, actual.Id)
	})

	t.Run("get existing ASN by netbox id - gone", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockRetrieveRequest := mock_interfaces.NewMockIpamAsnsRetrieveRequest(ctrl)

		mockIpamAPI.EXPECT().IpamAsnsRetrieve(gomock.Any(), AsnId).Return(mockRetrieveRequest)
		mockRetrieveRequest.EXPECT().Execute().
			Return(nil, &http.Response{StatusCode: 404, Body: http.NoBody}, assert.AnError)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getAsn(context.TODO(), &models.ASN{Asn: asnValue}, int64(AsnId))

		AssertNil(t, err)
		assert.Nil(t, actual)
	})

	t.Run("get existing 32 bit ASN above MaxInt32", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockListRequest := mock_interfaces.NewMockIpamAsnsListRequest(ctrl)

		largeAsn := int64(4200000000)

		// The generated `asn=` filter cannot express this value, so the client has to
		// page through the ASN list instead.
		mockIpamAPI.EXPECT().IpamAsnsList(gomock.Any()).Return(mockListRequest)
		mockListRequest.EXPECT().Limit(int32(asnListPageSize)).Return(mockListRequest)
		mockListRequest.EXPECT().Offset(int32(0)).Return(mockListRequest)
		mockListRequest.EXPECT().Execute().
			Return(&v4client.PaginatedASNList{Count: 1, Results: []v4client.ASN{
				{Id: AsnId, Asn: largeAsn},
			}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getAsn(context.TODO(), &models.ASN{Asn: largeAsn}, 0)

		AssertNil(t, err)
		assert.NotNil(t, actual)
		assert.Equal(t, largeAsn, actual.Asn)
	})

	t.Run("reserve new ASN", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockCreateRequest := mock_interfaces.NewMockIpamAsnsCreateRequest(ctrl)

		// Setup: list returns empty → create
		expectAsnListByValue(ctrl, mockIpamAPI, asnValue, []v4client.ASN{})

		// Mock ASN range lookup for RIR
		expectAsnRangesListAll(ctrl, mockIpamAPI, asnRanges)

		mockIpamAPI.EXPECT().
			IpamAsnsCreate(gomock.Any()).
			Return(mockCreateRequest)

		var createdAsn v4client.ASNRequest
		mockCreateRequest.EXPECT().
			ASNRequest(gomock.Any()).
			DoAndReturn(func(req v4client.ASNRequest) interfaces.IpamAsnsCreateRequest {
				createdAsn = req
				return mockCreateRequest
			})

		mockCreateRequest.EXPECT().
			Execute().
			Return(&v4client.ASN{
				Id:          AsnId,
				Asn:         asnValue,
				Comments:    &comments,
				Description: &description,
				LastUpdated: *v4client.NewNullableTime(&expectedLastUpdated),
			}, &http.Response{StatusCode: 201, Body: http.NoBody}, nil)

		tenantLookup(mockTenancy)

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy},
		}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateAsn(context.TODO(),
			&models.ASN{
				Asn: asnValue,
				Metadata: &models.NetboxMetadata{
					Tenant:      tenantName,
					Comments:    comments,
					Description: description,
				},
			}, &netboxv1.Asn{})

		assert.NoError(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, AsnId, actual.Id)
		assert.Equal(t, asnValue, actual.Asn)
		assert.True(t, createdAsn.Rir.IsSet())
		assert.Equal(t, rirId, *createdAsn.Rir.Get().Int32)
	})

	t.Run("update existing ASN preserves the RIR", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamAsnsUpdateRequest(ctrl)

		// List returns an existing ASN
		expectAsnListByValue(ctrl, mockIpamAPI, asnValue, []v4client.ASN{expectedASN()})

		mockIpamAPI.EXPECT().
			IpamAsnsUpdate(gomock.Any(), AsnId).
			Return(mockUpdateRequest)

		// NetBox replaces the whole object on update, so the request has to carry the RIR.
		var updatedAsn v4client.ASNRequest
		mockUpdateRequest.EXPECT().
			ASNRequest(gomock.Any()).
			DoAndReturn(func(req v4client.ASNRequest) interfaces.IpamAsnsUpdateRequest {
				updatedAsn = req
				return mockUpdateRequest
			})

		updatedDesc := "updated description"
		mockUpdateRequest.EXPECT().
			Execute().
			Return(&v4client.ASN{
				Id:          AsnId,
				Asn:         asnValue,
				Description: &updatedDesc,
				LastUpdated: *v4client.NewNullableTime(&expectedLastUpdated),
			}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		tenantLookup(mockTenancy)

		compositeClient := &NetboxCompositeClient{
			clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI},
			clientV3: &NetboxClientV3{Tenancy: mockTenancy},
		}

		// Asn with old last updated → not up to date → triggers update
		actual, isUpToDate, err := compositeClient.ReserveOrUpdateAsn(context.TODO(),
			&models.ASN{
				Asn: asnValue,
				Metadata: &models.NetboxMetadata{
					Tenant:      tenantName,
					Comments:    comments,
					Description: updatedDesc,
				},
			}, &netboxv1.Asn{
				Status: netboxv1.AsnStatus{
					LastUpdated: metav1.NewTime(expectedLastUpdated.Add(-1 * time.Hour)),
				},
			})

		assert.NoError(t, err)
		assert.False(t, isUpToDate)
		assert.NotNil(t, actual)
		assert.Equal(t, AsnId, actual.Id)
		assert.True(t, updatedAsn.Rir.IsSet())
		assert.Equal(t, rirId, *updatedAsn.Rir.Get().Int32)
	})

	t.Run("update existing ASN without a RIR falls back to the ASN range", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockUpdateRequest := mock_interfaces.NewMockIpamAsnsUpdateRequest(ctrl)

		expectAsnListByValue(ctrl, mockIpamAPI, asnValue, []v4client.ASN{
			{Id: AsnId, Asn: asnValue, LastUpdated: *v4client.NewNullableTime(&expectedLastUpdated)},
		})
		expectAsnRangesListAll(ctrl, mockIpamAPI, asnRanges)

		mockIpamAPI.EXPECT().IpamAsnsUpdate(gomock.Any(), AsnId).Return(mockUpdateRequest)

		var updatedAsn v4client.ASNRequest
		mockUpdateRequest.EXPECT().
			ASNRequest(gomock.Any()).
			DoAndReturn(func(req v4client.ASNRequest) interfaces.IpamAsnsUpdateRequest {
				updatedAsn = req
				return mockUpdateRequest
			})
		mockUpdateRequest.EXPECT().Execute().
			Return(&v4client.ASN{Id: AsnId, Asn: asnValue}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		_, isUpToDate, err := compositeClient.ReserveOrUpdateAsn(context.TODO(),
			&models.ASN{Asn: asnValue, Metadata: &models.NetboxMetadata{Description: description}},
			&netboxv1.Asn{
				Status: netboxv1.AsnStatus{
					LastUpdated: metav1.NewTime(expectedLastUpdated.Add(-1 * time.Hour)),
				},
			})

		assert.NoError(t, err)
		assert.False(t, isUpToDate)
		assert.True(t, updatedAsn.Rir.IsSet())
		assert.Equal(t, rirId, *updatedAsn.Rir.Get().Int32)
	})

	t.Run("up to date ASN is not updated", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		expectAsnListByValue(ctrl, mockIpamAPI, asnValue, []v4client.ASN{expectedASN()})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, isUpToDate, err := compositeClient.ReserveOrUpdateAsn(context.TODO(),
			&models.ASN{Asn: asnValue},
			&netboxv1.Asn{
				Status: netboxv1.AsnStatus{
					LastUpdated: metav1.NewTime(expectedLastUpdated),
					Conditions: []metav1.Condition{
						{Type: netboxv1.ConditionAsnReadyTrue.Type, Status: metav1.ConditionTrue},
					},
				},
			})

		assert.NoError(t, err)
		assert.True(t, isUpToDate)
		assert.NotNil(t, actual)
	})

	t.Run("restoration hash mismatch", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		// Existing ASN has a different hash
		expectAsnListByValue(ctrl, mockIpamAPI, asnValue, []v4client.ASN{
			{
				Id:           AsnId,
				Asn:          asnValue,
				CustomFields: map[string]interface{}{config.GetOperatorConfig().NetboxRestorationHashFieldName: "different-hash"},
				LastUpdated:  *v4client.NewNullableTime(&expectedLastUpdated),
			},
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, isUpToDate, err := compositeClient.ReserveOrUpdateAsn(context.TODO(),
			&models.ASN{
				Asn: asnValue,
				Metadata: &models.NetboxMetadata{
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: "my-hash",
					},
				},
			}, &netboxv1.Asn{})

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrRestorationHashMismatch)
		assert.False(t, isUpToDate)
		assert.Nil(t, result)
	})

	t.Run("ASN without a restoration hash in NetBox is not adopted", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		// NetBox returns null for unset custom fields. Such an ASN was never claimed by
		// this operator and must never be taken over silently.
		expectAsnListByValue(ctrl, mockIpamAPI, asnValue, []v4client.ASN{
			{
				Id:           AsnId,
				Asn:          asnValue,
				CustomFields: map[string]interface{}{config.GetOperatorConfig().NetboxRestorationHashFieldName: nil},
				LastUpdated:  *v4client.NewNullableTime(&expectedLastUpdated),
			},
		})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		result, isUpToDate, err := compositeClient.ReserveOrUpdateAsn(context.TODO(),
			&models.ASN{
				Asn: asnValue,
				Metadata: &models.NetboxMetadata{
					Description: description,
					Custom: map[string]string{
						config.GetOperatorConfig().NetboxRestorationHashFieldName: "my-hash",
					},
				},
			}, &netboxv1.Asn{
				Status: netboxv1.AsnStatus{
					LastUpdated: metav1.NewTime(expectedLastUpdated.Add(-1 * time.Hour)),
				},
			})

		assert.ErrorIs(t, err, ErrRestorationHashMismatch)
		assert.False(t, isUpToDate)
		assert.Nil(t, result)
	})

	t.Run("get RIR pages through all ASN ranges", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)

		fillerPage := make([]v4client.ASNRange, asnListPageSize)
		for i := range fillerPage {
			fillerPage[i] = v4client.ASNRange{Id: int32(i + 1), Start: 1, End: 2}
		}

		firstPage := mock_interfaces.NewMockIpamAsnRangesListRequest(ctrl)
		mockIpamAPI.EXPECT().IpamAsnRangesList(gomock.Any()).Return(firstPage)
		firstPage.EXPECT().Limit(int32(asnListPageSize)).Return(firstPage)
		firstPage.EXPECT().Offset(int32(0)).Return(firstPage)
		firstPage.EXPECT().Execute().
			Return(&v4client.PaginatedASNRangeList{Count: asnListPageSize + 1, Results: fillerPage}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		secondPage := mock_interfaces.NewMockIpamAsnRangesListRequest(ctrl)
		mockIpamAPI.EXPECT().IpamAsnRangesList(gomock.Any()).Return(secondPage)
		secondPage.EXPECT().Limit(int32(asnListPageSize)).Return(secondPage)
		secondPage.EXPECT().Offset(int32(asnListPageSize)).Return(secondPage)
		secondPage.EXPECT().Execute().
			Return(&v4client.PaginatedASNRangeList{Count: asnListPageSize + 1, Results: asnRanges}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getRirIdForAsn(context.TODO(), asnValue)

		assert.NoError(t, err)
		assert.Equal(t, rirId, actual)
	})

	t.Run("delete ASN", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamAsnsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamAsnsDestroy(gomock.Any(), AsnId).
			Return(mockDestroyRequest)

		mockDestroyRequest.EXPECT().
			Execute().
			Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		err := compositeClient.DeleteAsn(context.TODO(), int64(AsnId))
		assert.NoError(t, err)
	})

	t.Run("delete ASN not found", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamAsnsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamAsnsDestroy(gomock.Any(), AsnId).
			Return(mockDestroyRequest)

		mockDestroyRequest.EXPECT().
			Execute().
			Return(&http.Response{StatusCode: 404, Body: http.NoBody}, assert.AnError)

		clientV4 := &NetboxClientV4{IpamAPI: mockIpamAPI}
		compositeClient := &NetboxCompositeClient{clientV4: clientV4}

		err := compositeClient.DeleteAsn(context.TODO(), int64(AsnId))
		assert.NoError(t, err) // Should not error on 404
	})

	t.Run("delete ASN server error", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		mockDestroyRequest := mock_interfaces.NewMockIpamAsnsDestroyRequest(ctrl)

		mockIpamAPI.EXPECT().
			IpamAsnsDestroy(gomock.Any(), AsnId).
			Return(mockDestroyRequest)

		mockDestroyRequest.EXPECT().
			Execute().
			Return(&http.Response{StatusCode: 500, Body: http.NoBody}, assert.AnError)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		err := compositeClient.DeleteAsn(context.TODO(), int64(AsnId))
		assert.Error(t, err)
	})
}
