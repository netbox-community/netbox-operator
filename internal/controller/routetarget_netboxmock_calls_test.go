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
	"go.uber.org/mock/gomock"
)

// -----------------------------
// IpamAPI mock functions (RouteTargetReconciler side: mockIpamAPIRouteTarget/
// mockRouteTargetListRequest/mockRouteTargetCreateRequest/
// mockRouteTargetUpdateRequest/mockRouteTargetDestroyRequest)
// -----------------------------

func mockRouteTargetAPIList(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockRouteTargetListRequest).MinTimes(1)
}

func mockRouteTargetAPICreate(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamRouteTargetsCreate(gomock.Any()).Return(mockRouteTargetCreateRequest).MinTimes(1)
}

func mockRouteTargetAPIUpdate(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamRouteTargetsUpdate(gomock.Any(), routeTargetId).Return(mockRouteTargetUpdateRequest).MinTimes(1)
}

func mockRouteTargetAPIDestroy(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamRouteTargetsDestroy(gomock.Any(), routeTargetId).Return(mockRouteTargetDestroyRequest).MinTimes(1)
}

// mockRouteTargetAPIDestroyZeroId matches the cleanup delete call issued when
// the CR is deleted after NetBox reservation never succeeded
// (o.Status.RouteTargetId==0).
func mockRouteTargetAPIDestroyZeroId(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamRouteTargetsDestroy(gomock.Any(), int32(0)).Return(mockRouteTargetDestroyRequest).MinTimes(1)
}

// mockRouteTargetListRequestByName matches the name-based lookup
// RouteTargetReconciler issues for the route target value, which varies by
// scenario (standalone, exact claim, range-assigned, restored).
func mockRouteTargetListRequestByName(name string) func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error) {
	return func(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
		reqMock.EXPECT().Name([]string{name}).Return(mockRouteTargetListRequest).MinTimes(1)
	}
}

func mockRouteTargetListRequestExecuteEmpty(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedRouteTargetListEmpty(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockRouteTargetListRequestExecuteExistingNoHash(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedRouteTargetListExisting(nil), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockRouteTargetListRequestExecuteExistingWithMismatchedHash(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedRouteTargetListExisting(routeTargetCustomFieldsWithHashMismatchNetboxFmt), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

// mockRouteTargetListRequestExecuteClaimExistingWithHash returns a mock for
// RouteTargetReconciler's name-based lookup finding a pre-existing NetBox
// object carrying the given restoration hash.
func mockRouteTargetListRequestExecuteClaimExistingWithHash(hash string, name string) func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error) {
	return func(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
		reqMock.EXPECT().Execute().
			Return(mockedRouteTargetListWithHash(hash, name), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
			MinTimes(1)
	}
}

func mockRouteTargetCreateRequestBuild(reqMock *mock_interfaces.MockIpamRouteTargetsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().RouteTargetRequest(gomock.Any()).Return(mockRouteTargetCreateRequest).MinTimes(1)
}

func mockRouteTargetCreateRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamRouteTargetsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedRouteTargetResponse(), &http.Response{StatusCode: 201, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockRouteTargetCreateRequestExecuteFail(reqMock *mock_interfaces.MockIpamRouteTargetsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return((*v4client.RouteTarget)(nil), &http.Response{StatusCode: 500, Body: http.NoBody}, fmt.Errorf("mock error in netbox")).
		MinTimes(1)
}

func mockRouteTargetUpdateRequestBuild(reqMock *mock_interfaces.MockIpamRouteTargetsUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().RouteTargetRequest(gomock.Any()).Return(mockRouteTargetUpdateRequest).MinTimes(1)
}

func mockRouteTargetUpdateRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamRouteTargetsUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedRouteTargetResponse(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockRouteTargetDestroyRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamRouteTargetsDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockRouteTargetDestroyRequestExecuteNotFound(reqMock *mock_interfaces.MockIpamRouteTargetsDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil).
		MinTimes(1)
}

// -----------------------------
// IpamAPI mock functions (RouteTargetClaimReconciler side:
// mockIpamAPIRouteTargetClaim/mockRouteTargetClaimListRequest). Only ever
// exercises IpamRouteTargetsList via forEachRouteTargetOfAsn (NameIsw +
// Limit/Offset paging), never Name/Create/Update/Destroy.
// -----------------------------

func mockRouteTargetAPIClaimList(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamRouteTargetsList(gomock.Any()).Return(mockRouteTargetClaimListRequest).MinTimes(1)
}

func mockRouteTargetClaimListRequestPaging(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().NameIsw([]string{fmt.Sprintf("%d:", routeTargetAsn)}).Return(mockRouteTargetClaimListRequest).MinTimes(1)
	reqMock.EXPECT().Limit(gomock.Any()).Return(mockRouteTargetClaimListRequest).MinTimes(1)
	reqMock.EXPECT().Offset(gomock.Any()).Return(mockRouteTargetClaimListRequest).MinTimes(1)
}

func mockRouteTargetClaimListRequestExecuteEmpty(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedRouteTargetListEmpty(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockRouteTargetClaimListRequestExecuteWithMatchingHash(reqMock *mock_interfaces.MockIpamRouteTargetsListRequest, catchUnexpectedParams chan error) {
	hash := generateRouteTargetRestorationHash(defaultRouteTargetClaimCRWithRange())
	reqMock.EXPECT().Execute().
		Return(mockedRouteTargetListWithHash(hash, routeTargetRestoredValue), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

// -----------------------------
// Reset mock functions
// -----------------------------

func resetRouteTargetMockFunctions() {
	mockIpamAPIRouteTarget.EXPECT().IpamRouteTargetsList(gomock.Any()).Times(0)
	mockIpamAPIRouteTarget.EXPECT().IpamRouteTargetsCreate(gomock.Any()).Times(0)
	mockIpamAPIRouteTarget.EXPECT().IpamRouteTargetsUpdate(gomock.Any(), gomock.Any()).Times(0)
	mockIpamAPIRouteTarget.EXPECT().IpamRouteTargetsDestroy(gomock.Any(), gomock.Any()).Times(0)
	mockRouteTargetListRequest.EXPECT().Name(gomock.Any()).Times(0)
	mockRouteTargetListRequest.EXPECT().NameIsw(gomock.Any()).Times(0)
	mockRouteTargetListRequest.EXPECT().Limit(gomock.Any()).Times(0)
	mockRouteTargetListRequest.EXPECT().Offset(gomock.Any()).Times(0)
	mockRouteTargetListRequest.EXPECT().Execute().Times(0)
	mockRouteTargetCreateRequest.EXPECT().RouteTargetRequest(gomock.Any()).Times(0)
	mockRouteTargetCreateRequest.EXPECT().Execute().Times(0)
	mockRouteTargetUpdateRequest.EXPECT().RouteTargetRequest(gomock.Any()).Times(0)
	mockRouteTargetUpdateRequest.EXPECT().Execute().Times(0)
	mockRouteTargetDestroyRequest.EXPECT().Execute().Times(0)

	mockIpamAPIRouteTargetClaim.EXPECT().IpamRouteTargetsList(gomock.Any()).Times(0)
	mockRouteTargetClaimListRequest.EXPECT().NameIsw(gomock.Any()).Times(0)
	mockRouteTargetClaimListRequest.EXPECT().Limit(gomock.Any()).Times(0)
	mockRouteTargetClaimListRequest.EXPECT().Offset(gomock.Any()).Times(0)
	mockRouteTargetClaimListRequest.EXPECT().Execute().Times(0)
}
