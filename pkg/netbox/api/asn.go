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
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"

	"github.com/go-openapi/runtime"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

const (
	// listPageSize is the number of objects requested from the NetBox ASN Range and
	// RIR list endpoints.
	listPageSize = 250
)

// newAsnListQuery builds the query for the v3 ASN list endpoint. It always requests the
// brief representation, as the full one cannot be decoded: the v3 model types `rir` as an
// integer, whereas NetBox 4 returns a nested object. Note that the returned option
// replaces the typed list parameters, so every filter has to be passed through here.
func newAsnListQuery(netBoxFields map[string]string, customFields []CustomFieldEntry) func(co *runtime.ClientOperation) {
	query := map[string]string{"brief": "true"}
	maps.Copy(query, netBoxFields)
	return newQueryFilterOperation(query, customFields)
}

func (c *NetboxCompositeClient) ReserveOrUpdateAsn(ctx context.Context, asn *models.ASN, asnV1 *netboxv1.Asn) (resp *v4client.ASN, isUpToDate bool, err error) {
	asnToUpdate, err := c.getAsn(ctx, asn, asnV1.Status.AsnId)
	if err != nil {
		return nil, false, err
	}

	desiredAsn := v4client.ASNRequest{
		Asn:          asn.Asn,
		CustomFields: make(map[string]interface{}),
	}

	if asn.Metadata != nil {
		for k, v := range asn.Metadata.Custom {
			desiredAsn.CustomFields[k] = v
		}
		desc := TruncateDescription(asn.Metadata.Description)
		desiredAsn.Description = &desc
		comments := asn.Metadata.Comments + warningComment
		desiredAsn.Comments = &comments

		if asn.Metadata.Tenant != "" {
			tenantDetails, err := c.getTenantDetails(asn.Metadata.Tenant)
			if err != nil {
				return nil, false, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredAsn.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}

		// NetBox treats an ASN update as a full replacement and drops the RIR when the
		// request omits it, so the RIR has to be set on the update path as well.
		if asn.Metadata.Rir != "" {
			rirDetails, err := c.getRirDetailsByName(ctx, asn.Metadata.Rir)
			if err != nil {
				return nil, false, err
			}
			rirId := int32(rirDetails.Id)
			desiredAsn.SetRir(v4client.Int32AsASNRequestRir(&rirId))
		}
	}

	// create ASN since it doesn't exist
	if asnToUpdate == nil {
		resp, err := c.createAsn(ctx, desiredAsn)
		return resp, false, err
	}

	if asnToUpdate.LastUpdated.Get() == nil || asnToUpdate.LastUpdated.Get().IsZero() {
		return nil, false, fmt.Errorf("last updated field is not set in Netbox for ASN %d", asn.Asn)
	}

	netboxLastUpdated := *asnToUpdate.LastUpdated.Get()

	// If the desired ASN carries a restoration hash, the ASN in NetBox must carry exactly
	// the same one. An ASN without a hash, or with a different hash, belongs to somebody
	// else and must never be adopted.
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if asn.Metadata != nil {
		if restorationHash, ok := asn.Metadata.Custom[restorationHashKey]; ok {
			cfHash, cfOk := asnToUpdate.CustomFields[restorationHashKey]
			if !cfOk || cfHash == nil || cfHash == "" || cfHash != restorationHash {
				return nil, false, fmt.Errorf("%w, assigned ASN %d", ErrRestorationHashMismatch, asn.Asn)
			}
		}
	}

	if IsUpToDate(ctx, netboxLastUpdated, asnV1.Status.LastUpdated, asnV1.Status.Conditions, asnV1.Generation) {
		return asnToUpdate, true, nil
	}

	resp, err = c.updateAsn(ctx, asnToUpdate.Id, desiredAsn)
	if err != nil {
		return nil, false, err
	}
	return resp, false, nil
}

// getAsn returns the ASN in NetBox matching the given model, or nil if it does not exist.
func (c *NetboxCompositeClient) getAsn(ctx context.Context, asn *models.ASN, netboxAsnId int64) (*v4client.ASN, error) {
	// Once the ASN has been reconciled at least once we know its NetBox object id and can
	// look it up directly, avoiding the `asn=` filter entirely.
	if netboxAsnId != 0 {
		return c.retrieveAsn(ctx, int32(netboxAsnId))
	}

	// The v4 client types the `asn=` filter as int32, which cannot express 32-bit ASNs
	// above math.MaxInt32, so the lookup by value goes through the v3 client instead.
	asnValue := strconv.FormatInt(asn.Asn, 10)
	list, err := c.clientV3.Ipam.IpamAsnsList(
		ipam.NewIpamAsnsListParams().WithContext(ctx),
		nil,
		newAsnListQuery(map[string]string{"asn": asnValue}, nil),
	)
	if err != nil {
		return nil, err
	}
	if len(list.Payload.Results) == 0 {
		return nil, nil
	}

	return c.retrieveAsn(ctx, int32(list.Payload.Results[0].ID))
}

// retrieveAsn fetches a single ASN by its NetBox object id, returning nil if it is gone.
func (c *NetboxCompositeClient) retrieveAsn(ctx context.Context, asnId int32) (resp *v4client.ASN, err error) {
	result, httpResp, execErr := c.clientV4.IpamAPI.IpamAsnsRetrieve(ctx, asnId).Execute()

	if httpResp != nil && httpResp.Body != nil {
		defer func() { err = errors.Join(err, httpResp.Body.Close()) }()
	}

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if _, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "fetch ASN details"); handleErr != nil {
		return nil, handleErr
	}

	return result, nil
}

// listAsnRanges fetches ASN Ranges, optionally narrowed by name.
func (c *NetboxCompositeClient) listAsnRanges(ctx context.Context, nameFilter []string) (list *v4client.PaginatedASNRangeList, err error) {
	req := c.clientV4.IpamAPI.IpamAsnRangesList(ctx).Limit(listPageSize)
	if len(nameFilter) > 0 {
		req = req.Name(nameFilter)
	}

	result, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "fetch ASN Range details")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return result, nil
}

func (c *NetboxCompositeClient) createAsn(ctx context.Context, asn v4client.ASNRequest) (resp *v4client.ASN, err error) {
	result, httpResp, execErr := c.clientV4.IpamAPI.IpamAsnsCreate(ctx).ASNRequest(asn).Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "create ASN")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return result, nil
}

func (c *NetboxCompositeClient) updateAsn(ctx context.Context, asnId int32, asn v4client.ASNRequest) (resp *v4client.ASN, err error) {
	result, httpResp, execErr := c.clientV4.IpamAPI.IpamAsnsUpdate(ctx, asnId).ASNRequest(asn).Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update ASN")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return result, nil
}

func (c *NetboxCompositeClient) DeleteAsn(ctx context.Context, asnId int64) (err error) {
	httpResp, execErr := c.clientV4.IpamAPI.IpamAsnsDestroy(ctx, int32(asnId)).Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		if httpResp.Body != nil {
			return httpResp.Body.Close()
		}
		return nil
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusNoContent, "delete ASN from netbox")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return handleErr
	}

	return nil
}
