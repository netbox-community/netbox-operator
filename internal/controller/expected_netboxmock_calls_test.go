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

package controller

import (
	"fmt"

	"github.com/go-test/deep"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"go.uber.org/mock/gomock"
)

// -----------------------------
// IPAM Mock Functions
// -----------------------------

func mockIpAddressListWithIpAddressFilter(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesListOK, error) {
			got := params.(*ipam.IpamIPAddressesListParams)
			diff := deep.Equal(got, ExpectedIpAddressListParamsWithIpAddressData)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList was called with expected input\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseIPAddressListWithHash(customFieldsWithHash)}, nil
		}).MinTimes(1)
}

func mockIpAddressListWithIpAddressFilterEmptyResult(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesListOK, error) {
			got := params.(*ipam.IpamIPAddressesListParams)
			diff := deep.Equal(got, ExpectedIpAddressListParamsWithIpAddressData)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList (empty result) was called with expected input,\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseEmptyIPAddressList()}, nil
		}).MinTimes(1)
}

func mockIpAddressListWithHashFilterEmptyResult(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesListOK, error) {
			got := params.(*ipam.IpamIPAddressesListParams)
			diff := deep.Equal(got, ExpectedIpAddressListParams)
			// skip check for the 3rd input parameter as it is a method, method is a non comparable type
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList (empty result) was called with expected input,\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseEmptyIPAddressList()}, nil
		}).MinTimes(1)
}

func mockIpAddressListWithHashFilter(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesListOK, error) {
			got := params.(*ipam.IpamIPAddressesListParams)
			diff := deep.Equal(got, ExpectedIpAddressListParams)
			// skip check for the 3rd input parameter as it is a method, method is a non comparable type
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList (empty result) was called with expected input,\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseIPAddressList()}, nil
		}).MinTimes(1)
}

func mockIpAddressListWithHashFilterMismatch(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesListOK, error) {
			got := params.(*ipam.IpamIPAddressesListParams)
			diff := deep.Equal(got, ExpectedIpAddressListParamsWithIpAddressData)
			// skip check for the 3rd input parameter as it is a method, method is a non comparable type
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList (empty result) was called with expected input,\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseIPAddressListWithHash(customFieldsWithHashMismatch)}, nil
		}).MinTimes(1)
}

func mockPrefixesListWithPrefixFilter(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamPrefixesList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamPrefixesListOK, error) {
			got := params.(*ipam.IpamPrefixesListParams)
			diff := deep.Equal(got, ExpectedPrefixListParams)
			// skip check for the 3rd input parameter as it is a method, method is a non comparable type
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamPrefixesList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamPrefixesListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamPrefixesList was called with expected input,\n")
			return &ipam.IpamPrefixesListOK{Payload: mockedResponsePrefixList()}, nil
		}).MinTimes(1)
}

func mockPrefixesAvailableIpsList(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamPrefixesAvailableIpsList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamPrefixesAvailableIpsListOK, error) {
			got := params.(*ipam.IpamPrefixesAvailableIpsListParams)
			diff := deep.Equal(got, ExpectedPrefixesAvailableIpsListParams)
			// skip check for the 3rd input parameter as it is a method, method is a non comparable type
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamPrefixesAvailableIpsList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamPrefixesAvailableIpsListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamPrefixesAvailableIpsList was called with expected input,\n")
			return &ipam.IpamPrefixesAvailableIpsListOK{Payload: mockedResponseExpectedAvailableIpAddress()}, nil
		}).MinTimes(1)
}

func mockIpAddressesDelete(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesDelete(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesDeleteNoContent, error) {
			got := params.(*ipam.IpamIPAddressesDeleteParams)
			diff := deep.Equal(got, ExpectedDeleteParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesDelete, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesDeleteNoContent{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesDelete was called with mock input\n")
			return &ipam.IpamIPAddressesDeleteNoContent{}, nil
		}).MinTimes(1)
}

func mockIpAddressesDeleteFail(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesDelete(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesDeleteNoContent, error) {
			got := params.(*ipam.IpamIPAddressesDeleteParams)
			diff := deep.Equal(got, ExpectedDeleteFailParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesDelete, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesDeleteNoContent{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesDelete was called with mock input\n")
			return nil, ipam.NewIpamIPAddressesDeleteDefault(404)
		}).MinTimes(1)
}

func mockIpamIPAddressesUpdate(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesUpdateOK, error) {
			got := params.(*ipam.IpamIPAddressesUpdateParams)
			diff := deep.Equal(got, ExpectedIpAddressUpdateParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesUpdate, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesUpdateOK{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesUpdate was called with expected input\n")
			return &ipam.IpamIPAddressesUpdateOK{Payload: mockedResponseIPAddress()}, nil
		}).MinTimes(1)
}

func mockIpamIPAddressesUpdateWithHash(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesUpdateOK, error) {
			got := params.(*ipam.IpamIPAddressesUpdateParams)
			diff := deep.Equal(got, ExpectedIpAddressUpdateWithHashParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesUpdate (with hash), diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesUpdateOK{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesUpdate (with hash) was called with mock input\n")
			return &ipam.IpamIPAddressesUpdateOK{Payload: mockedResponseIPAddress()}, nil
		}).MinTimes(1)
}

func mockIpamIPAddressesUpdateFail(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesUpdateOK, error) {
			got := params.(*ipam.IpamIPAddressesUpdateParams)
			diff := deep.Equal(got, ExpectedIpAddressUpdateParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesUpdate, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesUpdateOK{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesUpdate was called with expected input\n")
			return &ipam.IpamIPAddressesUpdateOK{Payload: nil}, fmt.Errorf("ipam.IpamIpAddressesUpdate: mock error in netbox")
		}).MinTimes(1)
}

func mockIpamIPAddressesCreate(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesCreate(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesCreateCreated, error) {
			got := params.(*ipam.IpamIPAddressesCreateParams)
			diff := deep.Equal(got, ExpectedIpAddressesCreateParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesCreate, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesCreateCreated{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesCreate was called with expected input\n")
			return &ipam.IpamIPAddressesCreateCreated{Payload: mockedResponseIPAddress()}, nil
		}).MinTimes(1)
}

func mockIpamIPAddressesCreateWithHash(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesCreate(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesCreateCreated, error) {
			got := params.(*ipam.IpamIPAddressesCreateParams)
			diff := deep.Equal(got, ExpectedIpAddressesCreateWithHashParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesCreate (with hash), diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesCreateCreated{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesCreate (with hash) was called with expected input\n")
			return &ipam.IpamIPAddressesCreateCreated{Payload: mockedResponseIPAddress()}, nil
		}).MinTimes(1)
}

// -----------------------------
// Tenancy Mock Functions
// -----------------------------

func mockTenancyTenancyTenantsList(tenancyMock *mock_interfaces.MockTenancyInterface, catchUnexpectedParams chan error) {
	tenancyMock.EXPECT().TenancyTenantsList(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*tenancy.TenancyTenantsListOK, error) {
			got := params.(*tenancy.TenancyTenantsListParams)
			diff := deep.Equal(got, ExpectedTenantsListParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to tenancy.TenancyTenantsList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &tenancy.TenancyTenantsListOK{}, err
			}
			fmt.Printf("NETBOXMOCK\t tenancy.TenancyTenantsList was called with expected input\n")
			return &tenancy.TenancyTenantsListOK{Payload: mockedResponseTenancyTenantsList()}, nil
		}).MinTimes(1)
}

// -----------------------------
// Reset Mock Functions
// -----------------------------

func resetMockFunctions(ipamMockA *mock_interfaces.MockIpamInterface, ipamMockB *mock_interfaces.MockIpamInterface, tenancyMock *mock_interfaces.MockTenancyInterface) {
	ipamMockA.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any()).Times(0)
	ipamMockA.EXPECT().IpamIPAddressesUpdate(gomock.Any(), gomock.Any(), nil).Times(0)
	ipamMockA.EXPECT().IpamPrefixesList(gomock.Any(), gomock.Any()).Times(0)
	ipamMockA.EXPECT().IpamPrefixesAvailableIpsList(gomock.Any(), gomock.Any()).Times(0)
	ipamMockA.EXPECT().IpamIPAddressesDelete(gomock.Any(), nil).Times(0)
	ipamMockA.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Times(0)
	ipamMockA.EXPECT().IpamIPAddressesCreate(gomock.Any(), nil).Times(0)
	ipamMockB.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any()).Times(0)
	ipamMockB.EXPECT().IpamIPAddressesUpdate(gomock.Any(), gomock.Any(), nil).Times(0)
	ipamMockB.EXPECT().IpamPrefixesList(gomock.Any(), gomock.Any()).Times(0)
	ipamMockB.EXPECT().IpamPrefixesAvailableIpsList(gomock.Any(), gomock.Any()).Times(0)
	ipamMockB.EXPECT().IpamIPAddressesDelete(gomock.Any(), nil).Times(0)
	ipamMockB.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).Times(0)
	ipamMockB.EXPECT().IpamIPAddressesCreate(gomock.Any(), nil).Times(0)
	tenancyMock.EXPECT().TenancyTenantsList(gomock.Any(), nil).Times(0)
}
