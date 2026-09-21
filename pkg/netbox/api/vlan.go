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

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (c *NetboxCompositeClient) ReserveOrUpdateVlan(ctx context.Context, vlan *models.Vlan, vlanV1 *netboxv1.Vlan) (resp *v4client.VLAN, isUpToDate bool, err error) {
	responseVlanList, err := c.getVlan(ctx, vlan)
	if err != nil {
		return nil, false, err
	}

	desiredVlan := v4client.NewWritableVLANRequest(vlan.Vid, vlan.Name)
	status := vlan.Status
	if status == "" {
		status = string(v4client.PATCHEDWRITABLEVLANREQUESTSTATUS_ACTIVE)
	}
	desiredVlan.SetStatus(v4client.PatchedWritableVLANRequestStatus(status))

	if vlan.Metadata != nil {
		desiredVlan.SetComments(vlan.Metadata.Comments + warningComment)
		// Convert map[string]string to map[string]interface{}
		customFields := make(map[string]interface{}, len(vlan.Metadata.Custom))
		for k, v := range vlan.Metadata.Custom {
			customFields[k] = v
		}
		desiredVlan.SetCustomFields(customFields)
		desiredVlan.SetDescription(TruncateDescription(vlan.Metadata.Description))
		if vlan.Metadata.Site != "" {
			siteDetails, err := c.getSiteDetails(vlan.Metadata.Site)
			if err != nil {
				return nil, false, err
			}
			siteId := int32(siteDetails.Id)
			desiredVlan.SetSite(v4client.Int32AsPatchedWritableVLANRequestSite(&siteId))
		}
		if vlan.Metadata.Tenant != "" {
			tenantDetails, err := c.getTenantDetails(vlan.Metadata.Tenant)
			if err != nil {
				return nil, false, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredVlan.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}
	}

	// create vlan since it doesn't exist
	if len(responseVlanList.Results) == 0 {
		resp, err := c.createVlan(ctx, desiredVlan)
		return resp, false, err
	}

	vlanToUpdate := &responseVlanList.Results[0]

	if !vlanToUpdate.LastUpdated.IsSet() {
		return nil, false, fmt.Errorf("last updated field is not set in Netbox for vlan %s", vlan.Name)
	}

	// if the desired vlan has a restoration hash
	// check that the vlan to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if vlan.Metadata != nil {
		if restorationHash, ok := vlan.Metadata.Custom[restorationHashKey]; ok {
			if vlanToUpdate.CustomFields != nil && vlanToUpdate.CustomFields[restorationHashKey] == restorationHash {
				if IsUpToDate(ctx, *vlanToUpdate.LastUpdated.Get(), vlanV1.Status.LastUpdated, vlanV1.Status.Conditions, vlanV1.Generation) {
					return nil, true, nil
				}

				//update vlan since it does exist and the restoration hash matches
				resp, err := c.updateVlan(ctx, vlanToUpdate.Id, desiredVlan)
				if err != nil {
					return nil, false, err
				}
				return resp, false, nil
			}
			return nil, false, fmt.Errorf("%w, assigned vlan vid %d", ErrRestorationHashMismatch, vlan.Vid)
		}
	}

	if IsUpToDate(ctx, *vlanToUpdate.LastUpdated.Get(), vlanV1.Status.LastUpdated, vlanV1.Status.Conditions, vlanV1.Generation) {
		return nil, true, nil
	}

	//update vlan since it does exist
	vlanId := responseVlanList.Results[0].Id
	resp, err = c.updateVlan(ctx, vlanId, desiredVlan)
	if err != nil {
		return nil, false, err
	}
	return resp, false, nil
}

// getVlan looks up the NetBox VLAN matching vlan by its unique (site, vid)
// pair — VIDs are only guaranteed unique within a site, so vid alone isn't
// enough to identify a single VLAN. When vlan has no site, the result is
// filtered to VLANs that also have no site, since NetBox's site filter has
// no "site is unset" query option.
func (c *NetboxCompositeClient) getVlan(ctx context.Context, vlan *models.Vlan) (*v4client.PaginatedVLANList, error) {
	site := ""
	if vlan.Metadata != nil {
		site = vlan.Metadata.Site
	}

	req := c.clientV4.IpamAPI.IpamVlansList(ctx).Vid([]int32{vlan.Vid})
	if site != "" {
		siteDetails, err := c.getSiteDetails(site)
		if err != nil {
			return nil, err
		}
		req = req.Site([]string{siteDetails.Slug})
	}

	resp, httpResp, err := req.Execute()

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
		return nil, fmt.Errorf("failed to fetch vlan details: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch vlan details: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to fetch vlan details: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return nil, utils.NetboxError("failed to fetch vlan details", err)
	}

	if site == "" {
		siteless := make([]v4client.VLAN, 0, len(resp.Results))
		for _, v := range resp.Results {
			if !v.Site.IsSet() || v.Site.Get() == nil {
				siteless = append(siteless, v)
			}
		}
		resp.Results = siteless
		resp.Count = int32(len(siteless))
	}

	return resp, nil
}

func (c *NetboxCompositeClient) createVlan(ctx context.Context, vlan *v4client.WritableVLANRequest) (resp *v4client.VLAN, err error) {
	req := c.clientV4.IpamAPI.IpamVlansCreate(ctx).WritableVLANRequest(*vlan)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "reserve vlan")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) updateVlan(ctx context.Context, vlanId int32, vlan *v4client.WritableVLANRequest) (resp *v4client.VLAN, err error) {
	req := c.clientV4.IpamAPI.IpamVlansUpdate(ctx, vlanId).WritableVLANRequest(*vlan)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update vlan")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) DeleteVlan(ctx context.Context, vlanId int32) (err error) {
	req := c.clientV4.IpamAPI.IpamVlansDestroy(ctx, vlanId)
	httpResp, execErr := req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusNoContent, "delete vlan from netbox")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return handleErr
	}

	return nil
}
