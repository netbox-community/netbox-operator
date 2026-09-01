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
	"errors"
	"fmt"
	"math"
	"net/http"

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

const (
	// asnListPageSize is the number of objects requested per page when paginating
	// through the NetBox ASN and ASN Range list endpoints.
	asnListPageSize = 250

	// asnListMaxPages caps the number of pages fetched by the pagination helpers so
	// that an inconsistent NetBox response cannot make the operator loop forever.
	asnListMaxPages = 1000
)

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
	}

	// create ASN since it doesn't exist
	if asnToUpdate == nil {
		// Look up RIR from ASN range containing this ASN value
		rirId, err := c.getRirIdForAsn(ctx, asn.Asn)
		if err != nil {
			return nil, false, fmt.Errorf("failed to determine RIR for ASN %d: %w", asn.Asn, err)
		}
		desiredAsn.SetRir(v4client.Int32AsASNRequestRir(&rirId))
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

	// NetBox treats the ASN update as a full replacement and the generated request omits
	// `rir` when it is unset, which would clear the RIR. Carry over the RIR the ASN
	// already has, falling back to the RIR of the containing ASN Range.
	if rir := asnToUpdate.Rir.Get(); rir != nil {
		rirId := rir.Id
		desiredAsn.SetRir(v4client.Int32AsASNRequestRir(&rirId))
	} else {
		rirId, err := c.getRirIdForAsn(ctx, asn.Asn)
		if err != nil {
			return nil, false, fmt.Errorf("failed to determine RIR for ASN %d: %w", asn.Asn, err)
		}
		desiredAsn.SetRir(v4client.Int32AsASNRequestRir(&rirId))
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

	// The generated client's `asn=` filter only accepts int32, so it cannot express 32-bit
	// ASNs above math.MaxInt32. Fall back to a paginated scan for those.
	if asn.Asn > math.MaxInt32 {
		all, err := c.listAllAsns(ctx)
		if err != nil {
			return nil, err
		}
		for i := range all {
			if all[i].Asn == asn.Asn {
				return &all[i], nil
			}
		}
		return nil, nil
	}

	result, err := c.listAsnsPage(ctx, []int32{int32(asn.Asn)}, 0)
	if err != nil {
		return nil, err
	}
	if len(result.Results) == 0 {
		return nil, nil
	}
	return &result.Results[0], nil
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

// listAsnsPage fetches a single page of ASNs, optionally narrowed by the given asn filter.
func (c *NetboxCompositeClient) listAsnsPage(ctx context.Context, asnFilter []int32, offset int32) (list *v4client.PaginatedASNList, err error) {
	req := c.clientV4.IpamAPI.IpamAsnsList(ctx).Limit(asnListPageSize).Offset(offset)
	if len(asnFilter) > 0 {
		req = req.Asn(asnFilter)
	}

	result, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "fetch ASN details")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return result, nil
}

// listAllAsns pages through every ASN known to NetBox.
func (c *NetboxCompositeClient) listAllAsns(ctx context.Context) ([]v4client.ASN, error) {
	var all []v4client.ASN
	for page := 0; page < asnListMaxPages; page++ {
		result, err := c.listAsnsPage(ctx, nil, int32(len(all)))
		if err != nil {
			return nil, err
		}
		if len(result.Results) == 0 {
			return all, nil
		}
		all = append(all, result.Results...)
		if int32(len(all)) >= result.Count {
			return all, nil
		}
	}
	return nil, fmt.Errorf("failed to fetch ASN details: exceeded maximum of %d pages", asnListMaxPages)
}

// listAsnRangesPage fetches a single page of ASN Ranges, optionally narrowed by name.
func (c *NetboxCompositeClient) listAsnRangesPage(ctx context.Context, nameFilter []string, offset int32) (list *v4client.PaginatedASNRangeList, err error) {
	req := c.clientV4.IpamAPI.IpamAsnRangesList(ctx).Limit(asnListPageSize).Offset(offset)
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

// listAllAsnRanges pages through every ASN Range known to NetBox.
func (c *NetboxCompositeClient) listAllAsnRanges(ctx context.Context) ([]v4client.ASNRange, error) {
	var all []v4client.ASNRange
	for page := 0; page < asnListMaxPages; page++ {
		result, err := c.listAsnRangesPage(ctx, nil, int32(len(all)))
		if err != nil {
			return nil, err
		}
		if len(result.Results) == 0 {
			return all, nil
		}
		all = append(all, result.Results...)
		if int32(len(all)) >= result.Count {
			return all, nil
		}
	}
	return nil, fmt.Errorf("failed to fetch ASN Range details: exceeded maximum of %d pages", asnListMaxPages)
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

// getRirIdForAsn looks up ASN ranges to find which range contains the given ASN value
// and returns the RIR ID from that range. This is needed because NetBox requires an RIR
// when creating ASNs directly (as opposed to via the available-asns endpoint which inherits
// the RIR from the range).
func (c *NetboxCompositeClient) getRirIdForAsn(ctx context.Context, asnValue int64) (int32, error) {
	ranges, err := c.listAllAsnRanges(ctx)
	if err != nil {
		return 0, err
	}

	for _, r := range ranges {
		if asnValue >= r.Start && asnValue <= r.End {
			return r.Rir.Id, nil
		}
	}

	return 0, fmt.Errorf("no ASN range found containing ASN %d", asnValue)
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
