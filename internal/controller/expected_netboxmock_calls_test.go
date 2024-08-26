/*
Copyright 2024.

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

func expectedIpAddressListWithIpAddressFilter(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseIPAddressList()}, nil
		}).MinTimes(1)
}

func expectedIpAddressListWithIpAddressFilterEmptyResult(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesListOK, error) {
			got := params.(*ipam.IpamIPAddressesListParams)
			diff := deep.Equal(got, ExpectedIpAddressListParamsWithIpAddressData)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesList, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesListOK{Payload: nil}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList (empty reslut) was called with expected input,\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseEmptyIPAddressList()}, nil
		}).Times(1)
}

func expectedIpAddressListWithHashFilterEmptyResult(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList (empty reslut) was called with expected input,\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseEmptyIPAddressList()}, nil
		}).Times(1)
}

func expectedIpAddressListWithHashFilter(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesList (empty reslut) was called with expected input,\n")
			return &ipam.IpamIPAddressesListOK{Payload: mockedResponseIPAddressList()}, nil
		}).Times(1)
}

func expectedPrefixesListWithPrefixFilter(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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
		}).Times(1)
}

func expectedPrefixesAvailableIpsList(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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
		}).Times(1)
}

func expectedIpAddressesDelete(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
	ipamMock.EXPECT().IpamIPAddressesDelete(gomock.Any(), nil).
		DoAndReturn(func(params interface{}, authInfo interface{}, opts ...interface{}) (*ipam.IpamIPAddressesDeleteNoContent, error) {
			got := params.(*ipam.IpamIPAddressesDeleteParams)
			diff := deep.Equal(got, ExpectedDeleteParams)
			if len(diff) > 0 {
				err := fmt.Errorf("netboxmock: unexpected call to ipam.IpamIPAddressesDelete, diff to expected params diff: %+v", diff)
				catchUnexpectedParams <- err
				return &ipam.IpamIPAddressesDeleteNoContent{}, err
			}
			fmt.Printf("NETBOXMOCK\t ipam.IpamIPAddressesDelete was called with expected input\n")
			return &ipam.IpamIPAddressesDeleteNoContent{}, nil
		}).Times(1)
}

func expectedIpamIPAddressesUpdate(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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

func expectedIpamIPAddressesUpdateFail(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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

func expectedIpamIPAddressesCreate(ipamMock *mock_interfaces.MockIpamInterface, catchUnexpectedParams chan error) {
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
		}).Times(1)
}

// -----------------------------
// Tenancy Mock Functions
// -----------------------------

func expectedTenancyTenancyTenantsList(tenancyMock *mock_interfaces.MockTenancyInterface, catchUnexpectedParams chan error) {
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
