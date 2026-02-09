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

package api

import (
	"context"
	"net/http"

	nclient "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/netbox/interfaces"
)

// Adapter implementations for v4 API request objects

// ipamIpRangesListRequestAdapter adapts the v4 list request to the interface
type ipamIpRangesListRequestAdapter struct {
	req nclient.ApiIpamIpRangesListRequest
}

func (a *ipamIpRangesListRequestAdapter) StartAddress(startAddress []string) interfaces.IpamIpRangesListRequest {
	a.req = a.req.StartAddress(startAddress)
	return a
}

func (a *ipamIpRangesListRequestAdapter) EndAddress(endAddress []string) interfaces.IpamIpRangesListRequest {
	a.req = a.req.EndAddress(endAddress)
	return a
}

func (a *ipamIpRangesListRequestAdapter) Execute() (*nclient.PaginatedIPRangeList, *http.Response, error) {
	return a.req.Execute()
}

// ipamIpRangesCreateRequestAdapter adapts the v4 create request to the interface
type ipamIpRangesCreateRequestAdapter struct {
	req nclient.ApiIpamIpRangesCreateRequest
}

func (a *ipamIpRangesCreateRequestAdapter) WritableIPRangeRequest(writableIPRangeRequest nclient.WritableIPRangeRequest) interfaces.IpamIpRangesCreateRequest {
	a.req = a.req.WritableIPRangeRequest(writableIPRangeRequest)
	return a
}

func (a *ipamIpRangesCreateRequestAdapter) Execute() (*nclient.IPRange, *http.Response, error) {
	return a.req.Execute()
}

// ipamIpRangesUpdateRequestAdapter adapts the v4 update request to the interface
type ipamIpRangesUpdateRequestAdapter struct {
	req nclient.ApiIpamIpRangesUpdateRequest
}

func (a *ipamIpRangesUpdateRequestAdapter) WritableIPRangeRequest(writableIPRangeRequest nclient.WritableIPRangeRequest) interfaces.IpamIpRangesUpdateRequest {
	a.req = a.req.WritableIPRangeRequest(writableIPRangeRequest)
	return a
}

func (a *ipamIpRangesUpdateRequestAdapter) Execute() (*nclient.IPRange, *http.Response, error) {
	return a.req.Execute()
}

// ipamIpRangesDestroyRequestAdapter adapts the v4 destroy request to the interface
type ipamIpRangesDestroyRequestAdapter struct {
	req nclient.ApiIpamIpRangesDestroyRequest
}

func (a *ipamIpRangesDestroyRequestAdapter) Execute() (*http.Response, error) {
	return a.req.Execute()
}

// ipamV4APIAdapter adapts the v4 IpamAPI to the interface
type ipamV4APIAdapter struct {
	api nclient.IpamAPI
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
	req nclient.ApiIpamPrefixesListRequest
}

func (a *ipamPrefixesListRequestAdapter) Prefix(prefix []string) interfaces.IpamPrefixesListRequest {
	a.req = a.req.Prefix(prefix)
	return a
}

func (a *ipamPrefixesListRequestAdapter) Execute() (*nclient.PaginatedPrefixList, *http.Response, error) {
	return a.req.Execute()
}

// ipamPrefixesCreateRequestAdapter adapts the v4 create request to the interface
type ipamPrefixesCreateRequestAdapter struct {
	req nclient.ApiIpamPrefixesCreateRequest
}

func (a *ipamPrefixesCreateRequestAdapter) WritablePrefixRequest(writablePrefixRequest nclient.WritablePrefixRequest) interfaces.IpamPrefixesCreateRequest {
	a.req = a.req.WritablePrefixRequest(writablePrefixRequest)
	return a
}

func (a *ipamPrefixesCreateRequestAdapter) Execute() (*nclient.Prefix, *http.Response, error) {
	return a.req.Execute()
}

// ipamPrefixesUpdateRequestAdapter adapts the v4 update request to the interface
type ipamPrefixesUpdateRequestAdapter struct {
	req nclient.ApiIpamPrefixesUpdateRequest
}

func (a *ipamPrefixesUpdateRequestAdapter) WritablePrefixRequest(writablePrefixRequest nclient.WritablePrefixRequest) interfaces.IpamPrefixesUpdateRequest {
	a.req = a.req.WritablePrefixRequest(writablePrefixRequest)
	return a
}

func (a *ipamPrefixesUpdateRequestAdapter) Execute() (*nclient.Prefix, *http.Response, error) {
	return a.req.Execute()
}

// ipamPrefixesDestroyRequestAdapter adapts the v4 destroy request to the interface
type ipamPrefixesDestroyRequestAdapter struct {
	req nclient.ApiIpamPrefixesDestroyRequest
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
	req nclient.ApiStatusRetrieveRequest
}

func (a *statusRetrieveRequestAdapter) Execute() (map[string]any, *http.Response, error) {
	return a.req.Execute()
}

// statusV4APIAdapter adapts the v4 StatusAPI to the interface
type statusV4APIAdapter struct {
	api nclient.StatusAPI
}

func (a *statusV4APIAdapter) StatusRetrieve(ctx context.Context) interfaces.APIStatusRetrieveRequest {
	return &statusRetrieveRequestAdapter{req: a.api.StatusRetrieve(ctx)}
}
