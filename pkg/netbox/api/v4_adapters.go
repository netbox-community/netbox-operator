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

// ASN v4 adapters

type ipamAsnsListRequestAdapter struct {
	req v4client.ApiIpamAsnsListRequest
}

func (a *ipamAsnsListRequestAdapter) Asn(asn []int32) interfaces.IpamAsnsListRequest {
	a.req = a.req.Asn(asn)
	return a
}

func (a *ipamAsnsListRequestAdapter) Limit(limit int32) interfaces.IpamAsnsListRequest {
	a.req = a.req.Limit(limit)
	return a
}

func (a *ipamAsnsListRequestAdapter) Offset(offset int32) interfaces.IpamAsnsListRequest {
	a.req = a.req.Offset(offset)
	return a
}

func (a *ipamAsnsListRequestAdapter) Execute() (*v4client.PaginatedASNList, *http.Response, error) {
	return a.req.Execute()
}

type ipamAsnsRetrieveRequestAdapter struct {
	req v4client.ApiIpamAsnsRetrieveRequest
}

func (a *ipamAsnsRetrieveRequestAdapter) Execute() (*v4client.ASN, *http.Response, error) {
	return a.req.Execute()
}

type ipamAsnsCreateRequestAdapter struct {
	req v4client.ApiIpamAsnsCreateRequest
}

func (a *ipamAsnsCreateRequestAdapter) ASNRequest(aSNRequest v4client.ASNRequest) interfaces.IpamAsnsCreateRequest {
	a.req = a.req.ASNRequest(aSNRequest)
	return a
}

func (a *ipamAsnsCreateRequestAdapter) Execute() (*v4client.ASN, *http.Response, error) {
	return a.req.Execute()
}

type ipamAsnsUpdateRequestAdapter struct {
	req v4client.ApiIpamAsnsUpdateRequest
}

func (a *ipamAsnsUpdateRequestAdapter) ASNRequest(aSNRequest v4client.ASNRequest) interfaces.IpamAsnsUpdateRequest {
	a.req = a.req.ASNRequest(aSNRequest)
	return a
}

func (a *ipamAsnsUpdateRequestAdapter) Execute() (*v4client.ASN, *http.Response, error) {
	return a.req.Execute()
}

type ipamAsnsDestroyRequestAdapter struct {
	req v4client.ApiIpamAsnsDestroyRequest
}

func (a *ipamAsnsDestroyRequestAdapter) Execute() (*http.Response, error) {
	return a.req.Execute()
}

type ipamAsnRangesListRequestAdapter struct {
	req v4client.ApiIpamAsnRangesListRequest
}

func (a *ipamAsnRangesListRequestAdapter) Name(name []string) interfaces.IpamAsnRangesListRequest {
	a.req = a.req.Name(name)
	return a
}

func (a *ipamAsnRangesListRequestAdapter) Limit(limit int32) interfaces.IpamAsnRangesListRequest {
	a.req = a.req.Limit(limit)
	return a
}

func (a *ipamAsnRangesListRequestAdapter) Offset(offset int32) interfaces.IpamAsnRangesListRequest {
	a.req = a.req.Offset(offset)
	return a
}

func (a *ipamAsnRangesListRequestAdapter) Execute() (*v4client.PaginatedASNRangeList, *http.Response, error) {
	return a.req.Execute()
}

type ipamAsnRangesAvailableAsnsCreateRequestAdapter struct {
	req v4client.ApiIpamAsnRangesAvailableAsnsCreateRequest
}

func (a *ipamAsnRangesAvailableAsnsCreateRequestAdapter) ASNRequest(aSNRequest []v4client.ASNRequest) interfaces.IpamAsnRangesAvailableAsnsCreateRequest {
	a.req = a.req.ASNRequest(aSNRequest)
	return a
}

func (a *ipamAsnRangesAvailableAsnsCreateRequestAdapter) Execute() ([]v4client.ASN, *http.Response, error) {
	return a.req.Execute()
}

func (a *ipamV4APIAdapter) IpamAsnsList(ctx context.Context) interfaces.IpamAsnsListRequest {
	return &ipamAsnsListRequestAdapter{req: a.api.IpamAsnsList(ctx)}
}

func (a *ipamV4APIAdapter) IpamAsnsRetrieve(ctx context.Context, id int32) interfaces.IpamAsnsRetrieveRequest {
	return &ipamAsnsRetrieveRequestAdapter{req: a.api.IpamAsnsRetrieve(ctx, id)}
}

func (a *ipamV4APIAdapter) IpamAsnsCreate(ctx context.Context) interfaces.IpamAsnsCreateRequest {
	return &ipamAsnsCreateRequestAdapter{req: a.api.IpamAsnsCreate(ctx)}
}

func (a *ipamV4APIAdapter) IpamAsnsUpdate(ctx context.Context, id int32) interfaces.IpamAsnsUpdateRequest {
	return &ipamAsnsUpdateRequestAdapter{req: a.api.IpamAsnsUpdate(ctx, id)}
}

func (a *ipamV4APIAdapter) IpamAsnsDestroy(ctx context.Context, id int32) interfaces.IpamAsnsDestroyRequest {
	return &ipamAsnsDestroyRequestAdapter{req: a.api.IpamAsnsDestroy(ctx, id)}
}

func (a *ipamV4APIAdapter) IpamAsnRangesList(ctx context.Context) interfaces.IpamAsnRangesListRequest {
	return &ipamAsnRangesListRequestAdapter{req: a.api.IpamAsnRangesList(ctx)}
}

func (a *ipamV4APIAdapter) IpamAsnRangesAvailableAsnsCreate(ctx context.Context, id int32) interfaces.IpamAsnRangesAvailableAsnsCreateRequest {
	return &ipamAsnRangesAvailableAsnsCreateRequestAdapter{req: a.api.IpamAsnRangesAvailableAsnsCreate(ctx, id)}
}
