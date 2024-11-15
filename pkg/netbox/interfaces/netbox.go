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
	"github.com/go-openapi/runtime"
	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
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
