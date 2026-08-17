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
	v4client "github.com/netbox-community/go-netbox/v4"
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

type IpamIpRangesListRequest interface {
	StartAddress(startAddress []string) IpamIpRangesListRequest
	EndAddress(endAddress []string) IpamIpRangesListRequest
	Execute() (*v4client.PaginatedIPRangeList, *http.Response, error)
}

type IpamIpRangesCreateRequest interface {
	WritableIPRangeRequest(writableIPRangeRequest v4client.WritableIPRangeRequest) IpamIpRangesCreateRequest
	Execute() (*v4client.IPRange, *http.Response, error)
}

type IpamIpRangesUpdateRequest interface {
	WritableIPRangeRequest(writableIPRangeRequest v4client.WritableIPRangeRequest) IpamIpRangesUpdateRequest
	Execute() (*v4client.IPRange, *http.Response, error)
}

type IpamIpRangesDestroyRequest interface {
	Execute() (*http.Response, error)
}

type IpamPrefixesListRequest interface {
	Prefix(prefix []string) IpamPrefixesListRequest
	Execute() (*v4client.PaginatedPrefixList, *http.Response, error)
}

type IpamPrefixesCreateRequest interface {
	WritablePrefixRequest(writablePrefixRequest v4client.WritablePrefixRequest) IpamPrefixesCreateRequest
	Execute() (*v4client.Prefix, *http.Response, error)
}

type IpamPrefixesUpdateRequest interface {
	WritablePrefixRequest(writablePrefixRequest v4client.WritablePrefixRequest) IpamPrefixesUpdateRequest
	Execute() (*v4client.Prefix, *http.Response, error)
}

type IpamPrefixesDestroyRequest interface {
	Execute() (*http.Response, error)
}

type IpamAPI interface {
	IpamIpRangesList(ctx context.Context) IpamIpRangesListRequest
	IpamIpRangesCreate(ctx context.Context) IpamIpRangesCreateRequest
	IpamIpRangesUpdate(ctx context.Context, id int32) IpamIpRangesUpdateRequest
	IpamIpRangesDestroy(ctx context.Context, id int32) IpamIpRangesDestroyRequest
	IpamPrefixesList(ctx context.Context) IpamPrefixesListRequest
	IpamPrefixesCreate(ctx context.Context) IpamPrefixesCreateRequest
	IpamPrefixesUpdate(ctx context.Context, id int32) IpamPrefixesUpdateRequest
	IpamPrefixesDestroy(ctx context.Context, id int32) IpamPrefixesDestroyRequest
	IpamAsnsList(ctx context.Context) IpamAsnsListRequest
	IpamAsnsRetrieve(ctx context.Context, id int32) IpamAsnsRetrieveRequest
	IpamAsnsCreate(ctx context.Context) IpamAsnsCreateRequest
	IpamAsnsUpdate(ctx context.Context, id int32) IpamAsnsUpdateRequest
	IpamAsnsDestroy(ctx context.Context, id int32) IpamAsnsDestroyRequest
	IpamAsnRangesList(ctx context.Context) IpamAsnRangesListRequest
	IpamAsnRangesAvailableAsnsCreate(ctx context.Context, id int32) IpamAsnRangesAvailableAsnsCreateRequest
}

type APIStatusRetrieveRequest interface {
	Execute() (map[string]interface{}, *http.Response, error)
}

type StatusAPI interface {
	StatusRetrieve(ctx context.Context) APIStatusRetrieveRequest
}

// V4 ASN API Interfaces

type IpamAsnsListRequest interface {
	Asn(asn []int32) IpamAsnsListRequest
	Limit(limit int32) IpamAsnsListRequest
	Offset(offset int32) IpamAsnsListRequest
	Execute() (*v4client.PaginatedASNList, *http.Response, error)
}

type IpamAsnsRetrieveRequest interface {
	Execute() (*v4client.ASN, *http.Response, error)
}

type IpamAsnsCreateRequest interface {
	ASNRequest(aSNRequest v4client.ASNRequest) IpamAsnsCreateRequest
	Execute() (*v4client.ASN, *http.Response, error)
}

type IpamAsnsUpdateRequest interface {
	ASNRequest(aSNRequest v4client.ASNRequest) IpamAsnsUpdateRequest
	Execute() (*v4client.ASN, *http.Response, error)
}

type IpamAsnsDestroyRequest interface {
	Execute() (*http.Response, error)
}

type IpamAsnRangesListRequest interface {
	Name(name []string) IpamAsnRangesListRequest
	Limit(limit int32) IpamAsnRangesListRequest
	Offset(offset int32) IpamAsnRangesListRequest
	Execute() (*v4client.PaginatedASNRangeList, *http.Response, error)
}

type IpamAsnRangesAvailableAsnsCreateRequest interface {
	ASNRequest(aSNRequest []v4client.ASNRequest) IpamAsnRangesAvailableAsnsCreateRequest
	Execute() ([]v4client.ASN, *http.Response, error)
}
