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
	"regexp"
	"strings"

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"

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
		slug = "l2vpn"
	}
	return slug
}

func (c *NetboxCompositeClient) ReserveOrUpdateL2VPN(ctx context.Context, l2vpn *models.L2VPN, l2vpnV1 *netboxv1.L2VPN) (resp *v4client.L2VPN, isUpToDate bool, err error) {
	responseL2VPNList, err := c.getL2VPN(ctx, l2vpn)
	if err != nil {
		return nil, false, err
	}

	desiredL2VPN := v4client.NewWritableL2VPNRequest(l2vpn.Name, slugify(l2vpn.Name), v4client.BriefL2VPNTypeValue(l2vpn.Type))
	desiredL2VPN.SetIdentifier(l2vpn.Identifier)
	desiredL2VPN.SetStatus(v4client.L2VPNSTATUSVALUE_ACTIVE)

	if l2vpn.Metadata != nil {
		desiredL2VPN.SetComments(l2vpn.Metadata.Comments + warningComment)
		// Convert map[string]string to map[string]interface{}
		customFields := make(map[string]interface{}, len(l2vpn.Metadata.Custom))
		for k, v := range l2vpn.Metadata.Custom {
			customFields[k] = v
		}
		desiredL2VPN.SetCustomFields(customFields)
		desiredL2VPN.SetDescription(TruncateDescription(l2vpn.Metadata.Description))
		if l2vpn.Metadata.Tenant != "" {
			tenantDetails, err := c.getTenantDetails(l2vpn.Metadata.Tenant)
			if err != nil {
				return nil, false, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredL2VPN.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}
	}

	// create l2vpn since it doesn't exist
	if len(responseL2VPNList.Results) == 0 {
		resp, err := c.createL2VPN(ctx, desiredL2VPN)
		return resp, false, err
	}

	l2vpnToUpdate := &responseL2VPNList.Results[0]

	if !l2vpnToUpdate.LastUpdated.IsSet() {
		return nil, false, fmt.Errorf("last updated field is not set in Netbox for l2vpn %s", l2vpn.Name)
	}

	// if the desired l2vpn has a restoration hash
	// check that the l2vpn to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if l2vpn.Metadata != nil {
		if restorationHash, ok := l2vpn.Metadata.Custom[restorationHashKey]; ok {
			if l2vpnToUpdate.CustomFields != nil && l2vpnToUpdate.CustomFields[restorationHashKey] == restorationHash {
				if IsUpToDate(ctx, *l2vpnToUpdate.LastUpdated.Get(), l2vpnV1.Status.LastUpdated, l2vpnV1.Status.Conditions, l2vpnV1.Generation) {
					return nil, true, nil
				}

				//update l2vpn since it does exist and the restoration hash matches
				resp, err := c.updateL2VPN(ctx, l2vpnToUpdate.Id, desiredL2VPN)
				if err != nil {
					return nil, false, err
				}
				return resp, false, nil
			}
			return nil, false, fmt.Errorf("%w, assigned l2vpn identifier %d", ErrRestorationHashMismatch, l2vpn.Identifier)
		}
	}

	if IsUpToDate(ctx, *l2vpnToUpdate.LastUpdated.Get(), l2vpnV1.Status.LastUpdated, l2vpnV1.Status.Conditions, l2vpnV1.Generation) {
		return nil, true, nil
	}

	//update l2vpn since it does exist
	l2vpnId := responseL2VPNList.Results[0].Id
	resp, err = c.updateL2VPN(ctx, l2vpnId, desiredL2VPN)
	if err != nil {
		return nil, false, err
	}
	return resp, false, nil
}

func (c *NetboxCompositeClient) getL2VPN(ctx context.Context, l2vpn *models.L2VPN) (*v4client.PaginatedL2VPNList, error) {
	req := c.clientV4.VpnAPI.VpnL2vpnsList(ctx).
		Name([]string{l2vpn.Name})
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
		return nil, fmt.Errorf("failed to fetch l2vpn details: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch l2vpn details: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to fetch l2vpn details: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return nil, utils.NetboxError("failed to fetch l2vpn details", err)
	}

	return resp, nil
}

func (c *NetboxCompositeClient) createL2VPN(ctx context.Context, l2vpn *v4client.WritableL2VPNRequest) (resp *v4client.L2VPN, err error) {
	req := c.clientV4.VpnAPI.VpnL2vpnsCreate(ctx).WritableL2VPNRequest(*l2vpn)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "reserve l2vpn")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) updateL2VPN(ctx context.Context, l2vpnId int32, l2vpn *v4client.WritableL2VPNRequest) (resp *v4client.L2VPN, err error) {
	req := c.clientV4.VpnAPI.VpnL2vpnsUpdate(ctx, l2vpnId).WritableL2VPNRequest(*l2vpn)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update l2vpn")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) DeleteL2VPN(ctx context.Context, l2vpnId int32) (err error) {
	req := c.clientV4.VpnAPI.VpnL2vpnsDestroy(ctx, l2vpnId)
	httpResp, execErr := req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusNoContent, "delete l2vpn from netbox")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return handleErr
	}

	return nil
}
