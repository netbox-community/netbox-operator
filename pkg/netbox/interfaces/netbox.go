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

package interfaces

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	nclient "github.com/netbox-community/go-netbox/v4"
)

type IpamInterface interface {
	IpamIPAddressesList(params *ipam.IpamIPAddressesListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPAddressesListOK, error)
	IpamIPAddressesCreate(params *ipam.IpamIPAddressesCreateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPAddressesCreateCreated, error)
	IpamIPAddressesUpdate(params *ipam.IpamIPAddressesUpdateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPAddressesUpdateOK, error)
	IpamIPAddressesDelete(params *ipam.IpamIPAddressesDeleteParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPAddressesDeleteNoContent, error)
	IpamPrefixesAvailableIpsList(params *ipam.IpamPrefixesAvailableIpsListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamPrefixesAvailableIpsListOK, error)

	IpamPrefixesList(params *ipam.IpamPrefixesListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamPrefixesListOK, error)
	IpamPrefixesCreate(params *ipam.IpamPrefixesCreateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamPrefixesCreateCreated, error)
	IpamPrefixesUpdate(params *ipam.IpamPrefixesUpdateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamPrefixesUpdateOK, error)
	IpamPrefixesDelete(params *ipam.IpamPrefixesDeleteParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamPrefixesDeleteNoContent, error)
	IpamPrefixesAvailablePrefixesList(params *ipam.IpamPrefixesAvailablePrefixesListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamPrefixesAvailablePrefixesListOK, error)

	IpamIPRangesList(params *ipam.IpamIPRangesListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPRangesListOK, error)
	IpamIPRangesCreate(params *ipam.IpamIPRangesCreateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPRangesCreateCreated, error)
	IpamIPRangesUpdate(params *ipam.IpamIPRangesUpdateParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPRangesUpdateOK, error)
	IpamIPRangesDelete(params *ipam.IpamIPRangesDeleteParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPRangesDeleteNoContent, error)
	IpamIPRangesAvailableIpsList(params *ipam.IpamIPRangesAvailableIpsListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ipam.ClientOption) (*ipam.IpamIPRangesAvailableIpsListOK, error)
}

type TenancyInterface interface {
	TenancyTenantsList(params *tenancy.TenancyTenantsListParams, authInfo runtime.ClientAuthInfoWriter, opts ...tenancy.ClientOption) (*tenancy.TenancyTenantsListOK, error)
}

type ExtrasInterface interface {
	ExtrasCustomFieldsList(params *extras.ExtrasCustomFieldsListParams, authInfo runtime.ClientAuthInfoWriter, opts ...extras.ClientOption) (*extras.ExtrasCustomFieldsListOK, error)
}

type DcimInterface interface {
	DcimSitesList(params *dcim.DcimSitesListParams, authInfo runtime.ClientAuthInfoWriter, opts ...dcim.ClientOption) (*dcim.DcimSitesListOK, error)
}

// V4 API Interfaces - Request Objects

// IpamIpRangesListRequest represents the fluent API for listing IP ranges
type IpamIpRangesListRequest interface {
	StartAddress(startAddress []string) IpamIpRangesListRequest
	EndAddress(endAddress []string) IpamIpRangesListRequest
	Execute() (*nclient.PaginatedIPRangeList, *http.Response, error)
}

// IpamIpRangesCreateRequest represents the fluent API for creating IP ranges
type IpamIpRangesCreateRequest interface {
	WritableIPRangeRequest(writableIPRangeRequest nclient.WritableIPRangeRequest) IpamIpRangesCreateRequest
	Execute() (*nclient.IPRange, *http.Response, error)
}

// IpamIpRangesUpdateRequest represents the fluent API for updating IP ranges
type IpamIpRangesUpdateRequest interface {
	WritableIPRangeRequest(writableIPRangeRequest nclient.WritableIPRangeRequest) IpamIpRangesUpdateRequest
	Execute() (*nclient.IPRange, *http.Response, error)
}

// IpamIpRangesDestroyRequest represents the fluent API for deleting IP ranges
type IpamIpRangesDestroyRequest interface {
	Execute() (*http.Response, error)
}

type IpamPrefixesListRequest interface {
	Prefix(prefix []string) IpamPrefixesListRequest
	Execute() (*nclient.PaginatedPrefixList, *http.Response, error)
}

// IpamPrefixesCreateRequest represents the fluent API for creating IP ranges
type IpamPrefixesCreateRequest interface {
	WritablePrefixRequest(writablePrefixRequest nclient.WritablePrefixRequest) IpamPrefixesCreateRequest
	Execute() (*nclient.Prefix, *http.Response, error)
}

// IpamPrefixesUpdateRequest represents the fluent API for updating IP ranges
type IpamPrefixesUpdateRequest interface {
	WritablePrefixRequest(writablePrefixRequest nclient.WritablePrefixRequest) IpamPrefixesUpdateRequest
	Execute() (*nclient.Prefix, *http.Response, error)
}

// IpamPrefixesDestroyRequest represents the fluent API for deleting IP ranges
type IpamPrefixesDestroyRequest interface {
	Execute() (*http.Response, error)
}

// IpamV4API represents the v4 API interface for IPAM operations
type IpamAPI interface {
	IpamIpRangesList(ctx context.Context) IpamIpRangesListRequest
	IpamIpRangesCreate(ctx context.Context) IpamIpRangesCreateRequest
	IpamIpRangesUpdate(ctx context.Context, id int32) IpamIpRangesUpdateRequest
	IpamIpRangesDestroy(ctx context.Context, id int32) IpamIpRangesDestroyRequest
	IpamPrefixesList(ctx context.Context) IpamPrefixesListRequest
	IpamPrefixesCreate(ctx context.Context) IpamPrefixesCreateRequest
	IpamPrefixesUpdate(ctx context.Context, id int32) IpamPrefixesUpdateRequest
	IpamPrefixesDestroy(ctx context.Context, id int32) IpamPrefixesDestroyRequest
}

// IpamPrefixesDestroyRequest represents the fluent API for deleting IP ranges
type ApiStatusRetrieveRequest interface {
	Execute() (map[string]interface{}, *http.Response, error)
}

type StatusAPI interface {
	StatusRetrieve(ctx context.Context) ApiStatusRetrieveRequest
}
