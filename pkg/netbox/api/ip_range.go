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
	"fmt"
	"io"
	"net/http"

	nclient "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (c *NetboxClientV4) ReserveOrUpdateIpRange(ctx context.Context, cLegacy *NetboxClient, ipRange *models.IpRange) (*nclient.IPRange, error) {
	responseIpRangeList, err := c.getIpRange(ctx, ipRange)
	if err != nil {
		return nil, err
	}

	desiredIpRange := nclient.NewWritableIPRangeRequest(ipRange.StartAddress, ipRange.EndAddress)
	desiredIpRange.SetStatus("active")
	desiredIpRange.SetMarkPopulated(true)

	if ipRange.Metadata != nil {
		desiredIpRange.SetComments(ipRange.Metadata.Comments + warningComment)
		// Convert map[string]string to map[string]interface{}
		customFields := make(map[string]interface{}, len(ipRange.Metadata.Custom))
		for k, v := range ipRange.Metadata.Custom {
			customFields[k] = v
		}
		desiredIpRange.SetCustomFields(customFields)
		desiredIpRange.SetDescription(ipRange.Metadata.Description)
		if ipRange.Metadata.Tenant != "" {
			tenantDetails, err := cLegacy.GetTenantDetails(ipRange.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredIpRange.SetTenant(nclient.Int32AsASNRangeRequestTenant(&tenantId))
		}
	}

	// create ip range since it doesn't exist
	if len(responseIpRangeList.Results) == 0 {
		return c.createIpRange(ctx, desiredIpRange)
	}

	ipRangeToUpdate := responseIpRangeList.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if ipRange.Metadata != nil {
		if restorationHash, ok := ipRange.Metadata.Custom[restorationHashKey]; ok {
			if ipRangeToUpdate.CustomFields != nil && ipRangeToUpdate.CustomFields[restorationHashKey] == restorationHash {
				//update ip address since it does exist and the restoration hash matches
				return c.updateIpRange(ctx, ipRangeToUpdate.Id, desiredIpRange)
			}
			return nil, fmt.Errorf("%w, assigned ip range %s-%s", ErrRestorationHashMismatch, ipRange.StartAddress, ipRange.EndAddress)
		}
	}

	//update ip range since it does exist
	ipRangeId := responseIpRangeList.Results[0].Id
	return c.updateIpRange(ctx, ipRangeId, desiredIpRange)
}

func (c *NetboxClientV4) getIpRange(ctx context.Context, ipRange *models.IpRange) (*nclient.PaginatedIPRangeList, error) {
	req := c.IpamAPI.IpamIpRangesList(ctx).
		StartAddress([]string{ipRange.StartAddress}).
		EndAddress([]string{ipRange.EndAddress})
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer httpResp.Body.Close()
	}

	if err != nil {
		return nil, utils.NetboxError("failed to fetch ip range details", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch ip range details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to fetch ip range details: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *NetboxClientV4) createIpRange(ctx context.Context, ipRange *nclient.WritableIPRangeRequest) (*nclient.IPRange, error) {
	req := c.IpamAPI.IpamIpRangesCreate(ctx).WritableIPRangeRequest(*ipRange)
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer httpResp.Body.Close()
	}

	if err != nil {
		return nil, utils.NetboxError("failed to reserve ip range", err)
	}

	if httpResp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch ip range details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to reserve ip range: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *NetboxClientV4) updateIpRange(ctx context.Context, ipRangeId int32, ipRange *nclient.WritableIPRangeRequest) (*nclient.IPRange, error) {
	req := c.IpamAPI.IpamIpRangesUpdate(ctx, ipRangeId).WritableIPRangeRequest(*ipRange)
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer httpResp.Body.Close()
	}

	if err != nil {
		return nil, utils.NetboxError("failed to update IP Range", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch IpRange details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to update IP Range: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *NetboxClientV4) DeleteIpRange(ctx context.Context, ipRangeId int64) error {
	req := c.IpamAPI.IpamIpRangesDestroy(ctx, int32(ipRangeId))
	httpResp, err := req.Execute()

	if httpResp != nil {
		defer httpResp.Body.Close()
	}

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return nil
		}
		return utils.NetboxError("failed to delete ip range from Netbox", err)
	}
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNotFound {
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return fmt.Errorf("failed to fetch IpRange details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, err)
		}
		return fmt.Errorf("failed to delete ip range from Netbox: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return nil
}
