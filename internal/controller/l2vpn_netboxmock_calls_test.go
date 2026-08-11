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

package controller

import (
	"fmt"
	"net/http"

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"go.uber.org/mock/gomock"
)

// -----------------------------
// VpnAPI mock functions (L2VPNReconciler side: mockVpnAPI/mockVpnListRequest/
// mockVpnCreateRequest/mockVpnUpdateRequest/mockVpnDestroyRequest)
// -----------------------------

func mockVpnAPIList(vpnAPIMock *mock_interfaces.MockVpnAPI, catchUnexpectedParams chan error) {
	vpnAPIMock.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockVpnListRequest).MinTimes(1)
}

func mockVpnAPICreate(vpnAPIMock *mock_interfaces.MockVpnAPI, catchUnexpectedParams chan error) {
	vpnAPIMock.EXPECT().VpnL2vpnsCreate(gomock.Any()).Return(mockVpnCreateRequest).MinTimes(1)
}

func mockVpnAPIUpdate(vpnAPIMock *mock_interfaces.MockVpnAPI, catchUnexpectedParams chan error) {
	vpnAPIMock.EXPECT().VpnL2vpnsUpdate(gomock.Any(), l2vpnId).Return(mockVpnUpdateRequest).MinTimes(1)
}

func mockVpnAPIDestroy(vpnAPIMock *mock_interfaces.MockVpnAPI, catchUnexpectedParams chan error) {
	vpnAPIMock.EXPECT().VpnL2vpnsDestroy(gomock.Any(), l2vpnId).Return(mockVpnDestroyRequest).MinTimes(1)
}

// mockVpnAPIDestroyZeroId matches the cleanup delete call issued when the CR
// is deleted after NetBox reservation never succeeded (o.Status.L2VPNId==0).
func mockVpnAPIDestroyZeroId(vpnAPIMock *mock_interfaces.MockVpnAPI, catchUnexpectedParams chan error) {
	vpnAPIMock.EXPECT().VpnL2vpnsDestroy(gomock.Any(), int32(0)).Return(mockVpnDestroyRequest).MinTimes(1)
}

func mockVpnListRequestByName(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Name([]string{l2vpnName}).Return(mockVpnListRequest).MinTimes(1)
}

func mockVpnListRequestExecuteEmpty(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedL2VPNListEmpty(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVpnListRequestExecuteExistingNoHash(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedL2VPNListExisting(nil), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVpnListRequestExecuteExistingWithMismatchedHash(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedL2VPNListExisting(l2vpnCustomFieldsWithHashMismatchNetboxFmt), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

// mockVpnListRequestByClaimName matches the name-based lookup L2VPNReconciler
// issues for a child L2VPN CR created by a L2VPNClaim (CR name == claim name).
func mockVpnListRequestByClaimName(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Name([]string{l2vpnClaimName}).Return(mockVpnListRequest).MinTimes(1)
}

// mockVpnListRequestExecuteClaimExistingWithHash returns a mock for
// L2VPNReconciler's name-based lookup finding a pre-existing NetBox object
// (named after the claim) carrying the given restoration hash/identifier.
func mockVpnListRequestExecuteClaimExistingWithHash(hash string, identifier int64) func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error) {
	return func(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
		lastUpdated := l2vpnLastUpdated
		existing := v4client.L2VPN{
			Id:   l2vpnId,
			Name: l2vpnClaimName,
			Slug: l2vpnSlug,
			CustomFields: map[string]interface{}{
				config.GetOperatorConfig().NetboxRestorationHashFieldName: hash,
			},
		}
		existing.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
		existing.Identifier = *v4client.NewNullableInt64(&identifier)
		reqMock.EXPECT().Execute().
			Return(&v4client.PaginatedL2VPNList{Count: 1, Results: []v4client.L2VPN{existing}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
			MinTimes(1)
	}
}

func mockVpnCreateRequestBuild(reqMock *mock_interfaces.MockVpnL2vpnsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().WritableL2VPNRequest(gomock.Any()).Return(mockVpnCreateRequest).MinTimes(1)
}

func mockVpnCreateRequestExecuteSuccess(reqMock *mock_interfaces.MockVpnL2vpnsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedL2VPNResponse(), &http.Response{StatusCode: 201, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVpnCreateRequestExecuteFail(reqMock *mock_interfaces.MockVpnL2vpnsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return((*v4client.L2VPN)(nil), &http.Response{StatusCode: 500, Body: http.NoBody}, fmt.Errorf("mock error in netbox")).
		MinTimes(1)
}

func mockVpnUpdateRequestBuild(reqMock *mock_interfaces.MockVpnL2vpnsUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().WritableL2VPNRequest(gomock.Any()).Return(mockVpnUpdateRequest).MinTimes(1)
}

func mockVpnUpdateRequestExecuteSuccess(reqMock *mock_interfaces.MockVpnL2vpnsUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedL2VPNResponse(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVpnDestroyRequestExecuteSuccess(reqMock *mock_interfaces.MockVpnL2vpnsDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVpnDestroyRequestExecuteNotFound(reqMock *mock_interfaces.MockVpnL2vpnsDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil).
		MinTimes(1)
}

// -----------------------------
// VpnAPI mock functions (L2VPNClaimReconciler side: mockVpnAPIClaim/
// mockVpnClaimListRequest). Only ever exercises VpnL2vpnsList via
// forEachL2VPN (Limit/Offset paging), never Name/Create/Update/Destroy.
// -----------------------------

func mockVpnAPIClaimList(vpnAPIMock *mock_interfaces.MockVpnAPI, catchUnexpectedParams chan error) {
	vpnAPIMock.EXPECT().VpnL2vpnsList(gomock.Any()).Return(mockVpnClaimListRequest).MinTimes(1)
}

func mockVpnClaimListRequestPaging(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Limit(gomock.Any()).Return(mockVpnClaimListRequest).MinTimes(1)
	reqMock.EXPECT().Offset(gomock.Any()).Return(mockVpnClaimListRequest).MinTimes(1)
}

func mockVpnClaimListRequestExecuteEmpty(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedL2VPNListEmpty(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVpnClaimListRequestExecuteWithMatchingHash(reqMock *mock_interfaces.MockVpnL2vpnsListRequest, catchUnexpectedParams chan error) {
	hash := generateL2VPNRestorationHash(defaultL2VPNClaimCRWithRange())
	reqMock.EXPECT().Execute().
		Return(mockedL2VPNListWithHash(hash, l2vpnRestoredIdentifier), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

// -----------------------------
// Reset mock functions
// -----------------------------

func resetVpnMockFunctions() {
	mockVpnAPI.EXPECT().VpnL2vpnsList(gomock.Any()).Times(0)
	mockVpnAPI.EXPECT().VpnL2vpnsCreate(gomock.Any()).Times(0)
	mockVpnAPI.EXPECT().VpnL2vpnsUpdate(gomock.Any(), gomock.Any()).Times(0)
	mockVpnAPI.EXPECT().VpnL2vpnsDestroy(gomock.Any(), gomock.Any()).Times(0)
	mockVpnListRequest.EXPECT().Name(gomock.Any()).Times(0)
	mockVpnListRequest.EXPECT().Limit(gomock.Any()).Times(0)
	mockVpnListRequest.EXPECT().Offset(gomock.Any()).Times(0)
	mockVpnListRequest.EXPECT().Execute().Times(0)
	mockVpnCreateRequest.EXPECT().WritableL2VPNRequest(gomock.Any()).Times(0)
	mockVpnCreateRequest.EXPECT().Execute().Times(0)
	mockVpnUpdateRequest.EXPECT().WritableL2VPNRequest(gomock.Any()).Times(0)
	mockVpnUpdateRequest.EXPECT().Execute().Times(0)
	mockVpnDestroyRequest.EXPECT().Execute().Times(0)

	mockVpnAPIClaim.EXPECT().VpnL2vpnsList(gomock.Any()).Times(0)
	mockVpnClaimListRequest.EXPECT().Limit(gomock.Any()).Times(0)
	mockVpnClaimListRequest.EXPECT().Offset(gomock.Any()).Times(0)
	mockVpnClaimListRequest.EXPECT().Execute().Times(0)
}
