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

package api

import (
	"context"
	"net/http"

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/netbox/interfaces"
)

// Adapter implementations for v4 API request objects

// ipamIpRangesListRequestAdapter adapts the v4 list request to the interface
type ipamIpRangesListRequestAdapter struct {
	req v4client.ApiIpamIpRangesListRequest
}

func (a *ipamIpRangesListRequestAdapter) StartAddress(startAddress []string) interfaces.IpamIpRangesListRequest {
	a.req = a.req.StartAddress(startAddress)
	return a
}

func (a *ipamIpRangesListRequestAdapter) EndAddress(endAddress []string) interfaces.IpamIpRangesListRequest {
	a.req = a.req.EndAddress(endAddress)
	return a
}

func (a *ipamIpRangesListRequestAdapter) Execute() (*v4client.PaginatedIPRangeList, *http.Response, error) {
	return a.req.Execute()
}

// ipamIpRangesCreateRequestAdapter adapts the v4 create request to the interface
type ipamIpRangesCreateRequestAdapter struct {
	req v4client.ApiIpamIpRangesCreateRequest
}

func (a *ipamIpRangesCreateRequestAdapter) WritableIPRangeRequest(writableIPRangeRequest v4client.WritableIPRangeRequest) interfaces.IpamIpRangesCreateRequest {
	a.req = a.req.WritableIPRangeRequest(writableIPRangeRequest)
	return a
}

func (a *ipamIpRangesCreateRequestAdapter) Execute() (*v4client.IPRange, *http.Response, error) {
	return a.req.Execute()
}

// ipamIpRangesUpdateRequestAdapter adapts the v4 update request to the interface
type ipamIpRangesUpdateRequestAdapter struct {
	req v4client.ApiIpamIpRangesUpdateRequest
}

func (a *ipamIpRangesUpdateRequestAdapter) WritableIPRangeRequest(writableIPRangeRequest v4client.WritableIPRangeRequest) interfaces.IpamIpRangesUpdateRequest {
	a.req = a.req.WritableIPRangeRequest(writableIPRangeRequest)
	return a
}

func (a *ipamIpRangesUpdateRequestAdapter) Execute() (*v4client.IPRange, *http.Response, error) {
	return a.req.Execute()
}

// ipamIpRangesDestroyRequestAdapter adapts the v4 destroy request to the interface
type ipamIpRangesDestroyRequestAdapter struct {
	req v4client.ApiIpamIpRangesDestroyRequest
}

func (a *ipamIpRangesDestroyRequestAdapter) Execute() (*http.Response, error) {
	return a.req.Execute()
}

// ipamV4APIAdapter adapts the v4 IpamAPI to the interface
type ipamV4APIAdapter struct {
	api v4client.IpamAPI
}

func (a *ipamV4APIAdapter) IpamIpRangesList(ctx context.Context) interfaces.IpamIpRangesListRequest {
	return &ipamIpRangesListRequestAdapter{req: a.api.IpamIpRangesList(ctx)}
}

func (a *ipamV4APIAdapter) IpamIpRangesCreate(ctx context.Context) interfaces.IpamIpRangesCreateRequest {
	return &ipamIpRangesCreateRequestAdapter{req: a.api.IpamIpRangesCreate(ctx)}
}

func (a *ipamV4APIAdapter) IpamIpRangesUpdate(ctx context.Context, id int32) interfaces.IpamIpRangesUpdateRequest {
	return &ipamIpRangesUpdateRequestAdapter{req: a.api.IpamIpRangesUpdate(ctx, id)}
}

func (a *ipamV4APIAdapter) IpamIpRangesDestroy(ctx context.Context, id int32) interfaces.IpamIpRangesDestroyRequest {
	return &ipamIpRangesDestroyRequestAdapter{req: a.api.IpamIpRangesDestroy(ctx, id)}
}

// ipamPrefixesListRequestAdapter adapts the v4 list request to the interface
type ipamPrefixesListRequestAdapter struct {
	req v4client.ApiIpamPrefixesListRequest
}

func (a *ipamPrefixesListRequestAdapter) Prefix(prefix []string) interfaces.IpamPrefixesListRequest {
	a.req = a.req.Prefix(prefix)
	return a
}

func (a *ipamPrefixesListRequestAdapter) Execute() (*v4client.PaginatedPrefixList, *http.Response, error) {
	return a.req.Execute()
}

// ipamPrefixesCreateRequestAdapter adapts the v4 create request to the interface
type ipamPrefixesCreateRequestAdapter struct {
	req v4client.ApiIpamPrefixesCreateRequest
}

func (a *ipamPrefixesCreateRequestAdapter) WritablePrefixRequest(writablePrefixRequest v4client.WritablePrefixRequest) interfaces.IpamPrefixesCreateRequest {
	a.req = a.req.WritablePrefixRequest(writablePrefixRequest)
	return a
}

func (a *ipamPrefixesCreateRequestAdapter) Execute() (*v4client.Prefix, *http.Response, error) {
	return a.req.Execute()
}

// ipamPrefixesUpdateRequestAdapter adapts the v4 update request to the interface
type ipamPrefixesUpdateRequestAdapter struct {
	req v4client.ApiIpamPrefixesUpdateRequest
}

func (a *ipamPrefixesUpdateRequestAdapter) WritablePrefixRequest(writablePrefixRequest v4client.WritablePrefixRequest) interfaces.IpamPrefixesUpdateRequest {
	a.req = a.req.WritablePrefixRequest(writablePrefixRequest)
	return a
}

func (a *ipamPrefixesUpdateRequestAdapter) Execute() (*v4client.Prefix, *http.Response, error) {
	return a.req.Execute()
}

// ipamPrefixesDestroyRequestAdapter adapts the v4 destroy request to the interface
type ipamPrefixesDestroyRequestAdapter struct {
	req v4client.ApiIpamPrefixesDestroyRequest
}

func (a *ipamPrefixesDestroyRequestAdapter) Execute() (*http.Response, error) {
	return a.req.Execute()
}

func (a *ipamV4APIAdapter) IpamPrefixesList(ctx context.Context) interfaces.IpamPrefixesListRequest {
	return &ipamPrefixesListRequestAdapter{req: a.api.IpamPrefixesList(ctx)}
}

func (a *ipamV4APIAdapter) IpamPrefixesCreate(ctx context.Context) interfaces.IpamPrefixesCreateRequest {
	return &ipamPrefixesCreateRequestAdapter{req: a.api.IpamPrefixesCreate(ctx)}
}

func (a *ipamV4APIAdapter) IpamPrefixesUpdate(ctx context.Context, id int32) interfaces.IpamPrefixesUpdateRequest {
	return &ipamPrefixesUpdateRequestAdapter{req: a.api.IpamPrefixesUpdate(ctx, id)}
}

func (a *ipamV4APIAdapter) IpamPrefixesDestroy(ctx context.Context, id int32) interfaces.IpamPrefixesDestroyRequest {
	return &ipamPrefixesDestroyRequestAdapter{req: a.api.IpamPrefixesDestroy(ctx, id)}
}

// vpnL2vpnsListRequestAdapter adapts the v4 list request to the interface
type vpnL2vpnsListRequestAdapter struct {
	req v4client.ApiVpnL2vpnsListRequest
}

func (a *vpnL2vpnsListRequestAdapter) Name(name []string) interfaces.VpnL2vpnsListRequest {
	a.req = a.req.Name(name)
	return a
}

func (a *vpnL2vpnsListRequestAdapter) Type_(type_ []string) interfaces.VpnL2vpnsListRequest {
	a.req = a.req.Type_(type_)
	return a
}

func (a *vpnL2vpnsListRequestAdapter) Identifier(identifier []int32) interfaces.VpnL2vpnsListRequest {
	a.req = a.req.Identifier(identifier)
	return a
}

func (a *vpnL2vpnsListRequestAdapter) IdentifierGte(identifierGte []int32) interfaces.VpnL2vpnsListRequest {
	a.req = a.req.IdentifierGte(identifierGte)
	return a
}

func (a *vpnL2vpnsListRequestAdapter) IdentifierLte(identifierLte []int32) interfaces.VpnL2vpnsListRequest {
	a.req = a.req.IdentifierLte(identifierLte)
	return a
}

func (a *vpnL2vpnsListRequestAdapter) Limit(limit int32) interfaces.VpnL2vpnsListRequest {
	a.req = a.req.Limit(limit)
	return a
}

func (a *vpnL2vpnsListRequestAdapter) Offset(offset int32) interfaces.VpnL2vpnsListRequest {
	a.req = a.req.Offset(offset)
	return a
}

func (a *vpnL2vpnsListRequestAdapter) Execute() (*v4client.PaginatedL2VPNList, *http.Response, error) {
	return a.req.Execute()
}

// vpnL2vpnsCreateRequestAdapter adapts the v4 create request to the interface
type vpnL2vpnsCreateRequestAdapter struct {
	req v4client.ApiVpnL2vpnsCreateRequest
}

func (a *vpnL2vpnsCreateRequestAdapter) WritableL2VPNRequest(writableL2VPNRequest v4client.WritableL2VPNRequest) interfaces.VpnL2vpnsCreateRequest {
	a.req = a.req.WritableL2VPNRequest(writableL2VPNRequest)
	return a
}

func (a *vpnL2vpnsCreateRequestAdapter) Execute() (*v4client.L2VPN, *http.Response, error) {
	return a.req.Execute()
}

// vpnL2vpnsUpdateRequestAdapter adapts the v4 update request to the interface
type vpnL2vpnsUpdateRequestAdapter struct {
	req v4client.ApiVpnL2vpnsUpdateRequest
}

func (a *vpnL2vpnsUpdateRequestAdapter) WritableL2VPNRequest(writableL2VPNRequest v4client.WritableL2VPNRequest) interfaces.VpnL2vpnsUpdateRequest {
	a.req = a.req.WritableL2VPNRequest(writableL2VPNRequest)
	return a
}

func (a *vpnL2vpnsUpdateRequestAdapter) Execute() (*v4client.L2VPN, *http.Response, error) {
	return a.req.Execute()
}

// vpnL2vpnsDestroyRequestAdapter adapts the v4 destroy request to the interface
type vpnL2vpnsDestroyRequestAdapter struct {
	req v4client.ApiVpnL2vpnsDestroyRequest
}

func (a *vpnL2vpnsDestroyRequestAdapter) Execute() (*http.Response, error) {
	return a.req.Execute()
}

// vpnV4APIAdapter adapts the v4 VpnAPI to the interface
type vpnV4APIAdapter struct {
	api v4client.VpnAPI
}

func (a *vpnV4APIAdapter) VpnL2vpnsList(ctx context.Context) interfaces.VpnL2vpnsListRequest {
	return &vpnL2vpnsListRequestAdapter{req: a.api.VpnL2vpnsList(ctx)}
}

func (a *vpnV4APIAdapter) VpnL2vpnsCreate(ctx context.Context) interfaces.VpnL2vpnsCreateRequest {
	return &vpnL2vpnsCreateRequestAdapter{req: a.api.VpnL2vpnsCreate(ctx)}
}

func (a *vpnV4APIAdapter) VpnL2vpnsUpdate(ctx context.Context, id int32) interfaces.VpnL2vpnsUpdateRequest {
	return &vpnL2vpnsUpdateRequestAdapter{req: a.api.VpnL2vpnsUpdate(ctx, id)}
}

func (a *vpnV4APIAdapter) VpnL2vpnsDestroy(ctx context.Context, id int32) interfaces.VpnL2vpnsDestroyRequest {
	return &vpnL2vpnsDestroyRequestAdapter{req: a.api.VpnL2vpnsDestroy(ctx, id)}
}

type statusRetrieveRequestAdapter struct {
	req v4client.ApiStatusRetrieveRequest
}

func (a *statusRetrieveRequestAdapter) Execute() (map[string]any, *http.Response, error) {
	return a.req.Execute()
}

// statusV4APIAdapter adapts the v4 StatusAPI to the interface
type statusV4APIAdapter struct {
	api v4client.StatusAPI
}

func (a *statusV4APIAdapter) StatusRetrieve(ctx context.Context) interfaces.APIStatusRetrieveRequest {
	return &statusRetrieveRequestAdapter{req: a.api.StatusRetrieve(ctx)}
}
