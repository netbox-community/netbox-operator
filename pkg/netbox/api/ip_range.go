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
	"io"
	"net/http"

	nclient "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (c *NetboxCompositeClient) ReserveOrUpdateIpRange(ctx context.Context, ipRange *models.IpRange) (*nclient.IPRange, error) {
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
			tenantDetails, err := c.getTenantDetails(ipRange.Metadata.Tenant)
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

func (c *NetboxCompositeClient) getIpRange(ctx context.Context, ipRange *models.IpRange) (resp *nclient.PaginatedIPRangeList, err error) {
	req := c.clientV4.IpamAPI.IpamIpRangesList(ctx).
		StartAddress([]string{ipRange.StartAddress}).
		EndAddress([]string{ipRange.EndAddress})
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
		return nil, fmt.Errorf("failed to fetch ip range details: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch ip range details: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to fetch ip range details: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return nil, utils.NetboxError("failed to fetch ip range details", err)
	}

	return resp, nil
}

func (c *NetboxCompositeClient) createIpRange(ctx context.Context, ipRange *nclient.WritableIPRangeRequest) (resp *nclient.IPRange, err error) {
	req := c.clientV4.IpamAPI.IpamIpRangesCreate(ctx).WritableIPRangeRequest(*ipRange)
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
		return nil, fmt.Errorf("failed to reserve ip range: %w", err)
	}

	if httpResp.StatusCode != http.StatusCreated {
		if readErr != nil {
			return nil, fmt.Errorf("failed to reserve ip range: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to reserve ip range: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return nil, utils.NetboxError("failed to reserve ip range", err)
	}

	return resp, nil
}

func (c *NetboxCompositeClient) updateIpRange(ctx context.Context, ipRangeId int32, ipRange *nclient.WritableIPRangeRequest) (resp *nclient.IPRange, err error) {
	req := c.clientV4.IpamAPI.IpamIpRangesUpdate(ctx, ipRangeId).WritableIPRangeRequest(*ipRange)
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
		return nil, fmt.Errorf("failed to update ip range: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		if readErr != nil {
			return nil, fmt.Errorf("failed to update ip range: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to update ip range: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return nil, utils.NetboxError("failed to update ip range", err)
	}

	return resp, nil
}

func (c *NetboxCompositeClient) DeleteIpRange(ctx context.Context, ipRangeId int64) (err error) {
	req := c.clientV4.IpamAPI.IpamIpRangesDestroy(ctx, int32(ipRangeId))
	httpResp, err := req.Execute()

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
		return fmt.Errorf("failed to delete ip range from Netbox: %w", err)
	}

	if httpResp.StatusCode == http.StatusNotFound {
		return nil
	}

	if httpResp.StatusCode != http.StatusNoContent {
		if readErr != nil {
			return fmt.Errorf("failed to delete ip range from Netbox: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return fmt.Errorf("failed to delete ip range from Netbox: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return utils.NetboxError("failed to delete ip range from Netbox", err)
	}

	return nil
}
