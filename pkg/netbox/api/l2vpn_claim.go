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

// l2vpnListPageSize is the page size used when paginating through NetBox's
// L2VPN list endpoint. NetBox has no "available identifiers" endpoint for
// L2VPN the way it does for prefixes/IP ranges, so restoration-by-hash lookup
// and free-identifier search both have to list and scan client-side.
const l2vpnListPageSize = int32(100)

// forEachL2VPN pages through NetBox's entire L2VPN list and invokes visit for
// every result. visit returns false to stop iterating early.
func (c *NetboxCompositeClient) forEachL2VPN(ctx context.Context, visit func(l2vpn *v4client.L2VPN) bool) error {
	var offset int32
	for {
		req := c.clientV4.VpnAPI.VpnL2vpnsList(ctx).Limit(l2vpnListPageSize).Offset(offset)
		resp, httpResp, err := req.Execute()

		if httpResp != nil && httpResp.Body != nil {
			_, _ = io.ReadAll(httpResp.Body)
			closeErr := httpResp.Body.Close()
			err = errors.Join(err, closeErr)
		}
		if httpResp == nil {
			return fmt.Errorf("failed to list l2vpns: %w", err)
		}
		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to list l2vpns: status %d", httpResp.StatusCode)
		}
		if err != nil {
			return fmt.Errorf("failed to list l2vpns: %w", err)
		}

		for i := range resp.Results {
			if !visit(&resp.Results[i]) {
				return nil
			}
		}

		// A page shorter than requested means there's nothing left to fetch
		if int32(len(resp.Results)) < l2vpnListPageSize {
			return nil
		}

		offset += int32(len(resp.Results))
	}
}

// RestoreExistingL2VPNByHash searches for an existing NetBox L2VPN whose
// restoration hash custom field matches hash. Returns nil if none is found.
func (c *NetboxCompositeClient) RestoreExistingL2VPNByHash(ctx context.Context, hash string) (*models.L2VPN, error) {
	hashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName

	var found *v4client.L2VPN
	err := c.forEachL2VPN(ctx, func(l2vpn *v4client.L2VPN) bool {
		if len(l2vpn.CustomFields) == 0 {
			return true
		}

		if v, ok := l2vpn.CustomFields[hashKey]; ok && v == hash {
			found = l2vpn
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

	identifier := int64(0)
	if found.Identifier.IsSet() && found.Identifier.Get() != nil {
		identifier = *found.Identifier.Get()
	}

	return &models.L2VPN{
		Name:       found.Name,
		Slug:       found.Slug,
		Identifier: identifier,
		Id:         int64(found.Id),
	}, nil
}

// GetAvailableL2VPNIdentifierByClaim searches for a free VNI in the range
// requested by claim, by listing L2VPNs of the claim's type already using an
// identifier within [claim.IdentifierRangeStart, claim.IdentifierRangeEnd]
// and returning the first gap.
func (c *NetboxCompositeClient) GetAvailableL2VPNIdentifierByClaim(ctx context.Context, claim *models.L2VPNClaim) (*models.L2VPN, error) {
	if claim.Metadata != nil && claim.Metadata.Tenant != "" {
		if _, err := c.getTenantDetails(claim.Metadata.Tenant); err != nil {
			return nil, err
		}
	}

	var used []int64
	err := c.forEachL2VPN(ctx, func(l2vpn *v4client.L2VPN) bool {
		if !l2vpn.Identifier.IsSet() || l2vpn.Identifier.Get() == nil {
			return true
		}
		id := *l2vpn.Identifier.Get()
		if id >= claim.IdentifierRangeStart && id <= claim.IdentifierRangeEnd {
			used = append(used, id)
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(used)

	next := claim.IdentifierRangeStart
	for _, id := range used {
		if id > next {
			break
		}
		if id == next {
			next++
		}
	}

	if next > claim.IdentifierRangeEnd {
		return nil, ErrL2VPNRangeExhausted
	}

	return &models.L2VPN{
		Identifier: next,
	}, nil
}
