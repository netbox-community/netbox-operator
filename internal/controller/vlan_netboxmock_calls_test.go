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
// IpamAPI mock functions (VlanReconciler side: mockVlanIpamAPI/mockVlanListRequest/
// mockVlanCreateRequest/mockVlanUpdateRequest/mockVlanDestroyRequest)
// -----------------------------

func mockVlanIpamAPIList(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlansList(gomock.Any()).Return(mockVlanListRequest).MinTimes(1)
}

func mockVlanIpamAPICreate(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlansCreate(gomock.Any()).Return(mockVlanCreateRequest).MinTimes(1)
}

func mockVlanIpamAPIUpdate(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlansUpdate(gomock.Any(), vlanId).Return(mockVlanUpdateRequest).MinTimes(1)
}

func mockVlanIpamAPIDestroy(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlansDestroy(gomock.Any(), vlanId).Return(mockVlanDestroyRequest).MinTimes(1)
}

// mockVlanIpamAPIDestroyZeroId matches the cleanup delete call issued when the
// CR is deleted after NetBox reservation never succeeded (o.Status.VlanId==0).
func mockVlanIpamAPIDestroyZeroId(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlansDestroy(gomock.Any(), int32(0)).Return(mockVlanDestroyRequest).MinTimes(1)
}

func mockVlanListRequestByName(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Name([]string{vlanName}).Return(mockVlanListRequest).MinTimes(1)
}

func mockVlanListRequestExecuteEmpty(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanListEmpty(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanListRequestExecuteExistingNoHash(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanListExisting(nil), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanListRequestExecuteExistingWithMismatchedHash(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanListExisting(vlanCustomFieldsWithHashMismatchNetboxFmt), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

// mockVlanListRequestByClaimName matches the name-based lookup VlanReconciler
// issues for a child Vlan CR created by a VlanClaim (CR name == claim name).
func mockVlanListRequestByClaimName(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Name([]string{vlanClaimName}).Return(mockVlanListRequest).MinTimes(1)
}

// mockVlanListRequestExecuteClaimExistingWithHash returns a mock for
// VlanReconciler's name-based lookup finding a pre-existing NetBox object
// (named after the claim) carrying the given restoration hash/vid.
func mockVlanListRequestExecuteClaimExistingWithHash(hash string, vid int32) func(*mock_interfaces.MockIpamVlansListRequest, chan error) {
	return func(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
		lastUpdated := vlanLastUpdated
		existing := v4client.VLAN{
			Id:   vlanId,
			Name: vlanClaimName,
			Vid:  vid,
			CustomFields: map[string]interface{}{
				config.GetOperatorConfig().NetboxRestorationHashFieldName: hash,
			},
		}
		existing.LastUpdated = *v4client.NewNullableTime(&lastUpdated)
		reqMock.EXPECT().Execute().
			Return(&v4client.PaginatedVLANList{Count: 1, Results: []v4client.VLAN{existing}}, &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
			MinTimes(1)
	}
}

func mockVlanCreateRequestBuild(reqMock *mock_interfaces.MockIpamVlansCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().WritableVLANRequest(gomock.Any()).Return(mockVlanCreateRequest).MinTimes(1)
}

func mockVlanCreateRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamVlansCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanResponse(), &http.Response{StatusCode: 201, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanCreateRequestExecuteFail(reqMock *mock_interfaces.MockIpamVlansCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return((*v4client.VLAN)(nil), &http.Response{StatusCode: 500, Body: http.NoBody}, fmt.Errorf("mock error in netbox")).
		MinTimes(1)
}

func mockVlanUpdateRequestBuild(reqMock *mock_interfaces.MockIpamVlansUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().WritableVLANRequest(gomock.Any()).Return(mockVlanUpdateRequest).MinTimes(1)
}

func mockVlanUpdateRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamVlansUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanResponse(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanDestroyRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamVlansDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanDestroyRequestExecuteNotFound(reqMock *mock_interfaces.MockIpamVlansDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil).
		MinTimes(1)
}

// -----------------------------
// IpamAPI mock functions (VlanClaimReconciler side: mockVlanClaimIpamAPI/
// mockVlanClaimListRequest). Only ever exercises IpamVlansList via
// forEachVlan (Limit/Offset paging), never Name/Create/Update/Destroy.
// -----------------------------

func mockVlanIpamAPIClaimList(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlansList(gomock.Any()).Return(mockVlanClaimListRequest).MinTimes(1)
}

func mockVlanClaimListRequestPaging(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Limit(gomock.Any()).Return(mockVlanClaimListRequest).MinTimes(1)
	reqMock.EXPECT().Offset(gomock.Any()).Return(mockVlanClaimListRequest).MinTimes(1)
}

func mockVlanClaimListRequestExecuteEmpty(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanListEmpty(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanClaimListRequestExecuteWithMatchingHash(reqMock *mock_interfaces.MockIpamVlansListRequest, catchUnexpectedParams chan error) {
	hash := generateVlanRestorationHash(defaultVlanClaimCRWithRange())
	reqMock.EXPECT().Execute().
		Return(mockedVlanListWithHash(hash, vlanRestoredVid), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

// -----------------------------
// Reset mock functions
// -----------------------------

func resetVlanMockFunctions() {
	mockVlanIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Times(0)
	mockVlanIpamAPI.EXPECT().IpamVlansCreate(gomock.Any()).Times(0)
	mockVlanIpamAPI.EXPECT().IpamVlansUpdate(gomock.Any(), gomock.Any()).Times(0)
	mockVlanIpamAPI.EXPECT().IpamVlansDestroy(gomock.Any(), gomock.Any()).Times(0)
	mockVlanListRequest.EXPECT().Name(gomock.Any()).Times(0)
	mockVlanListRequest.EXPECT().Site(gomock.Any()).Times(0)
	mockVlanListRequest.EXPECT().Limit(gomock.Any()).Times(0)
	mockVlanListRequest.EXPECT().Offset(gomock.Any()).Times(0)
	mockVlanListRequest.EXPECT().Execute().Times(0)
	mockVlanCreateRequest.EXPECT().WritableVLANRequest(gomock.Any()).Times(0)
	mockVlanCreateRequest.EXPECT().Execute().Times(0)
	mockVlanUpdateRequest.EXPECT().WritableVLANRequest(gomock.Any()).Times(0)
	mockVlanUpdateRequest.EXPECT().Execute().Times(0)
	mockVlanDestroyRequest.EXPECT().Execute().Times(0)

	mockVlanClaimIpamAPI.EXPECT().IpamVlansList(gomock.Any()).Times(0)
	mockVlanClaimListRequest.EXPECT().Site(gomock.Any()).Times(0)
	mockVlanClaimListRequest.EXPECT().Limit(gomock.Any()).Times(0)
	mockVlanClaimListRequest.EXPECT().Offset(gomock.Any()).Times(0)
	mockVlanClaimListRequest.EXPECT().Execute().Times(0)
}
