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

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (c *NetboxCompositeClient) ReserveOrUpdateIpRange(ctx context.Context, ipRange *models.IpRange, ipRangeV1 *netboxv1.IpRange) (resp *v4client.IPRange, isUpToDate bool, err error) {
	responseIpRangeList, err := c.getIpRange(ctx, ipRange)
	if err != nil {
		return nil, true, err
	}

	desiredIpRange := v4client.NewWritableIPRangeRequest(ipRange.StartAddress, ipRange.EndAddress)
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
				return nil, true, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredIpRange.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}
	}

	// create ip range since it doesn't exist
	if len(responseIpRangeList.Results) == 0 {
		resp, err := c.createIpRange(ctx, desiredIpRange)
		return resp, false, err
	}

	ipRangeToUpdate := &responseIpRangeList.Results[0]

	if !ipRangeToUpdate.LastUpdated.IsSet() {
		return nil, true, fmt.Errorf("last updated field is not set in Netbox for ip range %s-%s", ipRange.StartAddress, ipRange.EndAddress)
	}

	// if the desired ip range has a restoration hash
	// check that the ip range to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if ipRange.Metadata != nil {
		if restorationHash, ok := ipRange.Metadata.Custom[restorationHashKey]; ok {
			if ipRangeToUpdate.CustomFields != nil && ipRangeToUpdate.CustomFields[restorationHashKey] == restorationHash {
				if IsUpToDate(ipRangeToUpdate.LastUpdated.Get(), ipRangeV1.Status.LastUpdated, ipRangeV1.Status.Conditions, ipRangeV1.Generation) {
					return nil, true, nil
				}

				//update ip range since it does exist and the restoration hash matches
				return c.updateIpRange(ctx, ipRangeToUpdate.Id, desiredIpRange)
			}
			return nil, true, fmt.Errorf("%w, assigned ip range %s-%s", ErrRestorationHashMismatch, ipRange.StartAddress, ipRange.EndAddress)
		}
	}

	if IsUpToDate(ipRangeToUpdate.LastUpdated.Get(), ipRangeV1.Status.LastUpdated, ipRangeV1.Status.Conditions, ipRangeV1.Generation) {
		return nil, true, nil
	}

	//update ip range since it does exist
	ipRangeId := responseIpRangeList.Results[0].Id
	return c.updateIpRange(ctx, ipRangeId, desiredIpRange)
}

func (c *NetboxCompositeClient) getIpRange(ctx context.Context, ipRange *models.IpRange) (resp *v4client.PaginatedIPRangeList, err error) {
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

func (c *NetboxCompositeClient) createIpRange(ctx context.Context, ipRange *v4client.WritableIPRangeRequest) (resp *v4client.IPRange, err error) {
	req := c.clientV4.IpamAPI.IpamIpRangesCreate(ctx).WritableIPRangeRequest(*ipRange)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "reserve ip range")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) updateIpRange(ctx context.Context, ipRangeId int32, ipRange *v4client.WritableIPRangeRequest) (resp *v4client.IPRange, isUpToDate bool, err error) {
	req := c.clientV4.IpamAPI.IpamIpRangesUpdate(ctx, ipRangeId).WritableIPRangeRequest(*ipRange)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update ip range")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, true, handleErr
	}

	return resp, false, nil
}

func (c *NetboxCompositeClient) DeleteIpRange(ctx context.Context, ipRangeId int32) (err error) {
	req := c.clientV4.IpamAPI.IpamIpRangesDestroy(ctx, ipRangeId)
	httpResp, execErr := req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusNoContent, "delete ip range from netbox")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return handleErr
	}

	return nil
}
