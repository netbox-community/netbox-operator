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
// IpamAPI mock functions (VlanGroupReconciler side: mockVlanGroupIpamAPI/
// mockVlanGroupListRequest/mockVlanGroupCreateRequest/mockVlanGroupUpdateRequest/
// mockVlanGroupDestroyRequest)
// -----------------------------

func mockVlanGroupIpamAPIList(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlanGroupsList(gomock.Any()).Return(mockVlanGroupListRequest).MinTimes(1)
}

func mockVlanGroupIpamAPICreate(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlanGroupsCreate(gomock.Any()).Return(mockVlanGroupCreateRequest).MinTimes(1)
}

func mockVlanGroupIpamAPIUpdate(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlanGroupsUpdate(gomock.Any(), vlanGroupId).Return(mockVlanGroupUpdateRequest).MinTimes(1)
}

func mockVlanGroupIpamAPIDestroy(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlanGroupsDestroy(gomock.Any(), vlanGroupId).Return(mockVlanGroupDestroyRequest).MinTimes(1)
}

// mockVlanGroupIpamAPIDestroyZeroId matches the cleanup delete call issued
// when the CR is deleted after NetBox reservation never succeeded
// (o.Status.VlanGroupId==0).
func mockVlanGroupIpamAPIDestroyZeroId(ipamAPIMock *mock_interfaces.MockIpamAPI, catchUnexpectedParams chan error) {
	ipamAPIMock.EXPECT().IpamVlanGroupsDestroy(gomock.Any(), int32(0)).Return(mockVlanGroupDestroyRequest).MinTimes(1)
}

func mockVlanGroupListRequestByName(reqMock *mock_interfaces.MockIpamVlanGroupsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Name([]string{vlanGroupName}).Return(mockVlanGroupListRequest).MinTimes(1)
}

func mockVlanGroupListRequestExecuteEmpty(reqMock *mock_interfaces.MockIpamVlanGroupsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanGroupListEmpty(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanGroupListRequestExecuteExisting(reqMock *mock_interfaces.MockIpamVlanGroupsListRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanGroupListExisting(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanGroupCreateRequestBuild(reqMock *mock_interfaces.MockIpamVlanGroupsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().VLANGroupRequest(gomock.Any()).Return(mockVlanGroupCreateRequest).MinTimes(1)
}

func mockVlanGroupCreateRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamVlanGroupsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanGroupResponse(), &http.Response{StatusCode: 201, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanGroupCreateRequestExecuteFail(reqMock *mock_interfaces.MockIpamVlanGroupsCreateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return((*v4client.VLANGroup)(nil), &http.Response{StatusCode: 500, Body: http.NoBody}, fmt.Errorf("mock error in netbox")).
		MinTimes(1)
}

func mockVlanGroupUpdateRequestBuild(reqMock *mock_interfaces.MockIpamVlanGroupsUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().VLANGroupRequest(gomock.Any()).Return(mockVlanGroupUpdateRequest).MinTimes(1)
}

func mockVlanGroupUpdateRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamVlanGroupsUpdateRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(mockedVlanGroupResponse(), &http.Response{StatusCode: 200, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanGroupDestroyRequestExecuteSuccess(reqMock *mock_interfaces.MockIpamVlanGroupsDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil).
		MinTimes(1)
}

func mockVlanGroupDestroyRequestExecuteNotFound(reqMock *mock_interfaces.MockIpamVlanGroupsDestroyRequest, catchUnexpectedParams chan error) {
	reqMock.EXPECT().Execute().
		Return(&http.Response{StatusCode: 404, Body: http.NoBody}, nil).
		MinTimes(1)
}

// -----------------------------
// Reset mock functions
// -----------------------------

func resetVlanGroupMockFunctions() {
	mockVlanGroupIpamAPI.EXPECT().IpamVlanGroupsList(gomock.Any()).Times(0)
	mockVlanGroupIpamAPI.EXPECT().IpamVlanGroupsCreate(gomock.Any()).Times(0)
	mockVlanGroupIpamAPI.EXPECT().IpamVlanGroupsUpdate(gomock.Any(), gomock.Any()).Times(0)
	mockVlanGroupIpamAPI.EXPECT().IpamVlanGroupsDestroy(gomock.Any(), gomock.Any()).Times(0)
	mockVlanGroupListRequest.EXPECT().Name(gomock.Any()).Times(0)
	mockVlanGroupListRequest.EXPECT().Execute().Times(0)
	mockVlanGroupCreateRequest.EXPECT().VLANGroupRequest(gomock.Any()).Times(0)
	mockVlanGroupCreateRequest.EXPECT().Execute().Times(0)
	mockVlanGroupUpdateRequest.EXPECT().VLANGroupRequest(gomock.Any()).Times(0)
	mockVlanGroupUpdateRequest.EXPECT().Execute().Times(0)
	mockVlanGroupDestroyRequest.EXPECT().Execute().Times(0)
}
