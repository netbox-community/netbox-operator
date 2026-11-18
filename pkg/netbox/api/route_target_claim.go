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
	"strconv"
	"strings"

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

// routeTargetListPageSize is the page size used when paginating through
// NetBox's Route Target list endpoint. NetBox has no "available route
// targets" endpoint the way it does for prefixes/IP ranges, so
// restoration-by-hash lookup and free-route-target search both have to list
// and scan client-side.
const routeTargetListPageSize = int32(100)

// RouteTargetName renders an ASN-based route target value in the
// "ASN:number" notation NetBox uses as the Route Target's name.
func RouteTargetName(asn int64, assignedNumber int64) string {
	return fmt.Sprintf("%d:%d", asn, assignedNumber)
}

// routeTargetAsnPrefix is the name prefix shared by all route targets of
// asn. The trailing colon keeps e.g. ASN 6500 from matching "65000:1".
func routeTargetAsnPrefix(asn int64) string {
	return fmt.Sprintf("%d:", asn)
}

// forEachRouteTargetOfAsn pages through all NetBox Route Targets whose name
// starts with "<asn>:" and invokes visit for every result. visit returns
// false to stop iterating early.
func (c *NetboxCompositeClient) forEachRouteTargetOfAsn(ctx context.Context, asn int64, visit func(routeTarget *v4client.RouteTarget) bool) error {
	var offset int32
	for {
		req := c.clientV4.IpamAPI.IpamRouteTargetsList(ctx).
			NameIsw([]string{routeTargetAsnPrefix(asn)}).
			Limit(routeTargetListPageSize).
			Offset(offset)
		resp, httpResp, err := req.Execute()

		if httpResp != nil && httpResp.Body != nil {
			_, _ = io.ReadAll(httpResp.Body)
			closeErr := httpResp.Body.Close()
			err = errors.Join(err, closeErr)
		}
		if httpResp == nil {
			return fmt.Errorf("failed to list route targets: %w", err)
		}
		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to list route targets: status %d", httpResp.StatusCode)
		}
		if err != nil {
			return fmt.Errorf("failed to list route targets: %w", err)
		}

		for i := range resp.Results {
			if !visit(&resp.Results[i]) {
				return nil
			}
		}

		// A page shorter than requested means there's nothing left to fetch
		if int32(len(resp.Results)) < routeTargetListPageSize {
			return nil
		}

		offset += int32(len(resp.Results))
	}
}

// RestoreExistingRouteTargetByHash searches the route targets of asn for one
// whose restoration hash custom field matches hash. Returns nil if none is
// found. The restoration hash covers the claim's ASN, so a match can only
// ever be a route target of that ASN.
func (c *NetboxCompositeClient) RestoreExistingRouteTargetByHash(ctx context.Context, asn int64, hash string) (*models.RouteTarget, error) {
	hashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName

	var found *v4client.RouteTarget
	err := c.forEachRouteTargetOfAsn(ctx, asn, func(routeTarget *v4client.RouteTarget) bool {
		if len(routeTarget.CustomFields) == 0 {
			return true
		}

		if v, ok := routeTarget.CustomFields[hashKey]; ok && v == hash {
			found = routeTarget
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

	return &models.RouteTarget{
		Name: found.Name,
		Id:   int64(found.Id),
	}, nil
}

// GetAvailableRouteTargetByClaim searches for a free route target in the
// range requested by claim, by listing the route targets of the claim's ASN
// already using an assigned number within
// [claim.AssignedNumberRangeStart, claim.AssignedNumberRangeEnd] and
// returning the first gap.
func (c *NetboxCompositeClient) GetAvailableRouteTargetByClaim(ctx context.Context, claim *models.RouteTargetClaim) (*models.RouteTarget, error) {
	if claim.Metadata != nil && claim.Metadata.Tenant != "" {
		if _, err := c.getTenantDetails(claim.Metadata.Tenant); err != nil {
			return nil, err
		}
	}

	prefix := routeTargetAsnPrefix(claim.Asn)

	var used []int64
	err := c.forEachRouteTargetOfAsn(ctx, claim.Asn, func(routeTarget *v4client.RouteTarget) bool {
		suffix, ok := strings.CutPrefix(routeTarget.Name, prefix)
		if !ok {
			return true
		}
		// skip names that aren't plain ASN:number route targets
		n, err := strconv.ParseInt(suffix, 10, 64)
		if err != nil {
			return true
		}
		if n >= claim.AssignedNumberRangeStart && n <= claim.AssignedNumberRangeEnd {
			used = append(used, n)
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(used)

	next := claim.AssignedNumberRangeStart
	for _, n := range used {
		if n > next {
			break
		}
		if n == next {
			next++
		}
	}

	if next > claim.AssignedNumberRangeEnd {
		return nil, ErrRouteTargetRangeExhausted
	}

	return &models.RouteTarget{
		Name: RouteTargetName(claim.Asn, next),
	}, nil
}
