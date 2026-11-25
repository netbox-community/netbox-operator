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
	"io"
	"net/http"
	"slices"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

// vlanListPageSize is the page size used when paginating through NetBox's
// VLAN list endpoint.
const vlanListPageSize = int32(100)

// listUsedVids pages through NetBox's VLAN list, filtered to VIDs within
// [vidRangeStart, vidRangeEnd] and scoped to siteSlug when non-empty, and
// returns the VIDs found.
func (c *NetboxCompositeClient) listUsedVids(ctx context.Context, siteSlug string, vidRangeStart, vidRangeEnd int32) ([]int32, error) {
	var used []int32
	var offset int32
	for {
		req := c.clientV4.IpamAPI.IpamVlansList(ctx).
			VidGte([]int32{vidRangeStart}).
			VidLte([]int32{vidRangeEnd}).
			Limit(vlanListPageSize).
			Offset(offset)
		if siteSlug != "" {
			req = req.Site([]string{siteSlug})
		}
		resp, httpResp, err := req.Execute()

		var body []byte
		if httpResp != nil && httpResp.Body != nil {
			var readErr error
			body, readErr = io.ReadAll(httpResp.Body)
			closeErr := httpResp.Body.Close()
			err = errors.Join(err, closeErr, readErr)
		}
		if httpResp == nil {
			return nil, fmt.Errorf("failed to list vlans: %w", err)
		}
		if httpResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to list vlans: status %d, body: %s", httpResp.StatusCode, string(body))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list vlans: %w", err)
		}

		for i := range resp.Results {
			used = append(used, resp.Results[i].GetVid())
		}

		offset += int32(len(resp.Results))
		if offset >= resp.Count || len(resp.Results) == 0 {
			return used, nil
		}
	}
}

// RestoreExistingVlanByHash searches for an existing NetBox VLAN whose
// restoration hash custom field matches hash. Returns nil if none is found.
// The go-netbox v4 client does not support custom field filters, so the v3
// client is used for this request.
func (c *NetboxCompositeClient) RestoreExistingVlanByHash(ctx context.Context, hash string) (*models.Vlan, error) {
	customVlanSearch := newQueryFilterOperation(nil, []CustomFieldEntry{
		{
			key:   config.GetOperatorConfig().NetboxRestorationHashFieldName,
			value: hash,
		},
	})
	list, err := c.clientV3.Ipam.IpamVlansList(ipam.NewIpamVlansListParams().WithContext(ctx), nil, customVlanSearch)
	if err != nil {
		return nil, err
	}

	if len(list.Payload.Results) == 0 {
		return nil, nil
	}

	// We should not have more than 1 result...
	if len(list.Payload.Results) != 1 {
		return nil, fmt.Errorf("incorrect number of restoration results, number of results: %v", len(list.Payload.Results))
	}
	res := list.Payload.Results[0]
	if res.Vid == nil {
		return nil, errors.New("vlan vid in netbox is nil")
	}
	if res.Name == nil {
		return nil, errors.New("vlan name in netbox is nil")
	}

	return &models.Vlan{
		Name: *res.Name,
		Vid:  int32(*res.Vid),
		Id:   res.ID,
	}, nil
}

// GetAvailableVlanByClaim searches for a free VID in the range requested by
// claim, scoped to claim.Metadata.Site, by listing VLANs already using a VID
// within [claim.VidRangeStart, claim.VidRangeEnd] and returning the first gap.
func (c *NetboxCompositeClient) GetAvailableVlanByClaim(ctx context.Context, claim *models.VlanClaim) (*models.Vlan, error) {
	site := ""
	if claim.Metadata != nil {
		site = claim.Metadata.Site
	}

	if claim.Metadata != nil && claim.Metadata.Tenant != "" {
		if _, err := c.getTenantDetails(claim.Metadata.Tenant); err != nil {
			return nil, err
		}
	}
	siteSlug := ""
	if site != "" {
		siteDetails, err := c.getSiteDetails(site)
		if err != nil {
			return nil, err
		}
		siteSlug = siteDetails.Slug
	}

	used, err := c.listUsedVids(ctx, siteSlug, claim.VidRangeStart, claim.VidRangeEnd)
	if err != nil {
		return nil, err
	}

	slices.Sort(used)

	next := claim.VidRangeStart
	for _, vid := range used {
		if vid > next {
			break
		}
		if vid == next {
			next++
		}
	}

	if next > claim.VidRangeEnd {
		return nil, ErrVlanRangeExhausted
	}

	return &models.Vlan{
		Vid: next,
	}, nil
}
