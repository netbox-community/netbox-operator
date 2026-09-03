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

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

// vlanListPageSize is the page size used when paginating through NetBox's
// VLAN list endpoint.
const vlanListPageSize = int32(100)

// forEachVlan pages through NetBox's VLAN list, scoped to siteSlug when
// non-empty, and invokes visit for every result.
func (c *NetboxCompositeClient) forEachVlan(ctx context.Context, siteSlug string, visit func(vlan *v4client.VLAN) bool) error {
	var offset int32
	for {
		req := c.clientV4.IpamAPI.IpamVlansList(ctx).Limit(vlanListPageSize).Offset(offset)
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
			return fmt.Errorf("failed to list vlans: %w", err)
		}
		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to list vlans: status %d, body: %s", httpResp.StatusCode, string(body))
		}
		if err != nil {
			return fmt.Errorf("failed to list vlans: %w", err)
		}

		for i := range resp.Results {
			if !visit(&resp.Results[i]) {
				return nil
			}
		}

		offset += int32(len(resp.Results))
		if offset >= resp.Count || len(resp.Results) == 0 {
			return nil
		}
	}
}

// RestoreExistingVlanByHash searches for an existing NetBox VLAN whose
// restoration hash custom field matches hash. Returns nil if none is found.
// The restoration hash is globally unique, so this scans all VLANs regardless
// of site.
func (c *NetboxCompositeClient) RestoreExistingVlanByHash(ctx context.Context, hash string) (*models.Vlan, error) {
	hashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName

	var found *v4client.VLAN
	err := c.forEachVlan(ctx, "", func(vlan *v4client.VLAN) bool {
		if vlan.CustomFields == nil {
			return true
		}
		if v, ok := vlan.CustomFields[hashKey]; ok && v == hash {
			found = vlan
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, nil
	}

	return &models.Vlan{
		Name: found.Name,
		Vid:  found.Vid,
		Id:   int64(found.Id),
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

	var used []int32
	err := c.forEachVlan(ctx, siteSlug, func(vlan *v4client.VLAN) bool {
		vid := vlan.GetVid()
		if vid >= claim.VidRangeStart && vid <= claim.VidRangeEnd {
			used = append(used, vid)
		}
		return true
	})
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
