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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

var slugInvalidCharsRegex = regexp.MustCompile(`[^-a-zA-Z0-9_]+`)

// slugify converts name into a value that satisfies NetBox's slug field
// constraints (matches ^[-a-zA-Z0-9_]+$, max 100 characters).
func slugify(name string) string {
	slug := slugInvalidCharsRegex.ReplaceAllString(strings.TrimSpace(name), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 100 {
		slug = strings.Trim(slug[:100], "-")
	}
	if slug == "" {
		slug = "vlangroup"
	}
	return slug
}

func (c *NetboxCompositeClient) ReserveOrUpdateVlanGroup(ctx context.Context, vlanGroup *models.VlanGroup, vlanGroupV1 *netboxv1.VlanGroup) (resp *v4client.VLANGroup, isUpToDate bool, err error) {
	responseVlanGroupList, err := c.getVlanGroup(ctx, vlanGroup)
	if err != nil {
		return nil, false, err
	}

	desiredVlanGroup := v4client.NewVLANGroupRequest(vlanGroup.Name, slugify(vlanGroup.Name))

	if vlanGroup.VidRangeStart != 0 && vlanGroup.VidRangeEnd != 0 {
		// The go-netbox v4 client types VidRanges as [][][]int32, but NetBox's
		// API actually sends/expects [][]int32 (a list of [lower, upper]
		// pairs), so using the typed setter produces a payload NetBox rejects.
		// Set it via AdditionalProperties instead, which is serialized after
		// (and so overrides) the typed field. See getVlanGroupDecodeBugFallback
		// for the matching read-side workaround.
		if desiredVlanGroup.AdditionalProperties == nil {
			desiredVlanGroup.AdditionalProperties = map[string]interface{}{}
		}
		desiredVlanGroup.AdditionalProperties["vid_ranges"] = [][]int32{{vlanGroup.VidRangeStart, vlanGroup.VidRangeEnd}}
	}

	if vlanGroup.Metadata != nil {
		desiredVlanGroup.SetDescription(TruncateDescription(vlanGroup.Metadata.Description))
		// Convert map[string]string to map[string]interface{}
		customFields := make(map[string]interface{}, len(vlanGroup.Metadata.Custom))
		for k, v := range vlanGroup.Metadata.Custom {
			customFields[k] = v
		}
		desiredVlanGroup.SetCustomFields(customFields)
		if vlanGroup.Metadata.Site != "" {
			siteDetails, err := c.getSiteDetails(vlanGroup.Metadata.Site)
			if err != nil {
				return nil, false, err
			}
			desiredVlanGroup.SetScopeType("dcim.site")
			desiredVlanGroup.SetScopeId(int32(siteDetails.Id))
		}
		if vlanGroup.Metadata.Tenant != "" {
			tenantDetails, err := c.getTenantDetails(vlanGroup.Metadata.Tenant)
			if err != nil {
				return nil, false, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredVlanGroup.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}
	}

	// create vlan group since it doesn't exist
	if len(responseVlanGroupList.Results) == 0 {
		resp, err := c.createVlanGroup(ctx, desiredVlanGroup)
		return resp, false, err
	}

	vlanGroupToUpdate := &responseVlanGroupList.Results[0]

	if !vlanGroupToUpdate.LastUpdated.IsSet() {
		return nil, false, fmt.Errorf("last updated field is not set in Netbox for vlan group %s", vlanGroup.Name)
	}

	if IsUpToDate(ctx, *vlanGroupToUpdate.LastUpdated.Get(), vlanGroupV1.Status.LastUpdated, vlanGroupV1.Status.Conditions, vlanGroupV1.Generation) {
		return nil, true, nil
	}

	//update vlan group since it does exist
	resp, err = c.updateVlanGroup(ctx, vlanGroupToUpdate.Id, desiredVlanGroup)
	if err != nil {
		return nil, false, err
	}
	return resp, false, nil
}

// vlanGroupFallbackJSON decodes just the fields ReserveOrUpdateVlanGroup and
// the VlanGroup controller actually read from a NetBox VLAN Group response,
// using correctly-typed fields (ScopeId as *int32, not the SDK's mis-typed
// VidRanges). Used by vlanGroupDecodeBugFallback as a workaround for the
// go-netbox v4 client bug described there.
type vlanGroupFallbackJSON struct {
	Id          int32      `json:"id"`
	Slug        string     `json:"slug"`
	ScopeId     *int32     `json:"scope_id"`
	LastUpdated *time.Time `json:"last_updated"`
}

func (v vlanGroupFallbackJSON) toVLANGroup() v4client.VLANGroup {
	vg := v4client.VLANGroup{Id: v.Id, Slug: v.Slug}
	if v.ScopeId != nil {
		vg.ScopeId = *v4client.NewNullableInt32(v.ScopeId)
	}
	if v.LastUpdated != nil {
		vg.LastUpdated = *v4client.NewNullableTime(v.LastUpdated)
	}
	return vg
}

// vlanGroupDecodeBugFallback recovers from a go-netbox v4 client bug: NetBox
// returns "vid_ranges" as [][]int32, but the SDK's VLANGroup.VidRanges field
// is typed as [][][]int32, so decoding any VLAN Group response that has
// vid_ranges set fails client-side even though the HTTP call succeeded. When
// that happens this re-decodes the raw response body.
// Returns ok=false if err isn't this specific situation, in which case the caller
// should treat err normally.
func vlanGroupDecodeBugFallback(httpResp *http.Response, err error) (body []byte, ok bool) {
	if err == nil || httpResp == nil || httpResp.StatusCode >= 300 {
		return nil, false
	}
	apiErr, ok := err.(*v4client.GenericOpenAPIError)
	if !ok {
		return nil, false
	}
	return apiErr.Body(), true
}

// getVlanGroup looks up the NetBox VLAN Group matching vlanGroup by its
// unique (site, name) pair. NetBox only enforces name (and slug) uniqueness
// within a scope, so name alone isn't enough to identify a single group.
// When vlanGroup has no site, the result is filtered to groups that also
// have no scope, since NetBox's site filter has no "scope is unset" query
// option.
func (c *NetboxCompositeClient) getVlanGroup(ctx context.Context, vlanGroup *models.VlanGroup) (*v4client.PaginatedVLANGroupList, error) {
	site := ""
	if vlanGroup.Metadata != nil {
		site = vlanGroup.Metadata.Site
	}

	req := c.clientV4.IpamAPI.IpamVlanGroupsList(ctx).Name([]string{vlanGroup.Name})
	if site != "" {
		siteDetails, err := c.getSiteDetails(site)
		if err != nil {
			return nil, err
		}
		req = req.Site(int32(siteDetails.Id))
	}

	resp, httpResp, err := req.Execute()

	if rawBody, ok := vlanGroupDecodeBugFallback(httpResp, err); ok {
		var fallback struct {
			Count   int32                   `json:"count"`
			Results []vlanGroupFallbackJSON `json:"results"`
		}
		if jsonErr := json.Unmarshal(rawBody, &fallback); jsonErr == nil {
			results := make([]v4client.VLANGroup, len(fallback.Results))
			for i, r := range fallback.Results {
				results[i] = r.toVLANGroup()
			}
			resp, err = &v4client.PaginatedVLANGroupList{Count: fallback.Count, Results: results}, nil
		}
	}

	var body []byte
	var readErr error
	if httpResp != nil && httpResp.Body != nil {
		defer func() {
			errClose := httpResp.Body.Close()
			err = errors.Join(err, errClose)
		}()
		body, readErr = io.ReadAll(httpResp.Body)
	}

	if httpResp == nil {
		return nil, fmt.Errorf("failed to fetch vlan group details: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch vlan group details: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to fetch vlan group details: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return nil, utils.NetboxError("failed to fetch vlan group details", err)
	}

	if site == "" {
		scopeless := make([]v4client.VLANGroup, 0, len(resp.Results))
		for _, v := range resp.Results {
			if !v.ScopeId.IsSet() || v.ScopeId.Get() == nil {
				scopeless = append(scopeless, v)
			}
		}
		resp.Results = scopeless
		resp.Count = int32(len(scopeless))
	}

	return resp, nil
}

func (c *NetboxCompositeClient) createVlanGroup(ctx context.Context, vlanGroup *v4client.VLANGroupRequest) (resp *v4client.VLANGroup, err error) {
	req := c.clientV4.IpamAPI.IpamVlanGroupsCreate(ctx).VLANGroupRequest(*vlanGroup)
	resp, httpResp, execErr := req.Execute()

	if rawBody, ok := vlanGroupDecodeBugFallback(httpResp, execErr); ok {
		var fallback vlanGroupFallbackJSON
		if jsonErr := json.Unmarshal(rawBody, &fallback); jsonErr == nil {
			vg := fallback.toVLANGroup()
			resp, execErr = &vg, nil
		}
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "reserve vlan group")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) updateVlanGroup(ctx context.Context, vlanGroupId int32, vlanGroup *v4client.VLANGroupRequest) (resp *v4client.VLANGroup, err error) {
	req := c.clientV4.IpamAPI.IpamVlanGroupsUpdate(ctx, vlanGroupId).VLANGroupRequest(*vlanGroup)
	resp, httpResp, execErr := req.Execute()

	if rawBody, ok := vlanGroupDecodeBugFallback(httpResp, execErr); ok {
		var fallback vlanGroupFallbackJSON
		if jsonErr := json.Unmarshal(rawBody, &fallback); jsonErr == nil {
			vg := fallback.toVLANGroup()
			resp, execErr = &vg, nil
		}
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update vlan group")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) DeleteVlanGroup(ctx context.Context, vlanGroupId int32) (err error) {
	req := c.clientV4.IpamAPI.IpamVlanGroupsDestroy(ctx, vlanGroupId)
	httpResp, execErr := req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusNoContent, "delete vlan group from netbox")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return handleErr
	}

	return nil
}
