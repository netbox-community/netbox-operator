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
	"net/http"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	nclient "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (r *NetboxClient) ReserveOrUpdateIpRange(ipRange *models.IpRange) (*netboxModels.IPRange, error) {
	responseIpRange, err := r.GetIpRange(ipRange)
	if err != nil {
		return nil, err
	}

	desiredIpRange := &netboxModels.WritableIPRange{
		StartAddress: &ipRange.StartAddress,
		EndAddress:   &ipRange.EndAddress,
		Comments:     ipRange.Metadata.Comments + warningComment,
		CustomFields: ipRange.Metadata.Custom,
		Description:  ipRange.Metadata.Description,
		Status:       "active",
	}

	if ipRange.Metadata != nil {
		desiredIpRange.CustomFields = ipRange.Metadata.Custom
		desiredIpRange.Comments = ipRange.Metadata.Comments + warningComment
		desiredIpRange.Description = TruncateDescription(ipRange.Metadata.Description)
	}

	if ipRange.Metadata != nil && ipRange.Metadata.Tenant != "" {
		tenantDetails, err := r.GetTenantDetails(ipRange.Metadata.Tenant)
		if err != nil {
			return nil, err
		}
		desiredIpRange.Tenant = &tenantDetails.Id
	}

	// create ip range since it doesn't exist
	if len(responseIpRange.Payload.Results) == 0 {
		return r.CreateIpRange(desiredIpRange)
	}

	ipRangeToUpdate := responseIpRange.Payload.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if ipRange.Metadata != nil {
		if restorationHash, ok := ipRange.Metadata.Custom[restorationHashKey]; ok {
			if ipRangeToUpdate.CustomFields != nil && ipRangeToUpdate.CustomFields.(map[string]interface{})[restorationHashKey] == restorationHash {
				//update ip address since it does exist and the restoration hash matches
				return r.UpdateIpRange(ipRangeToUpdate.ID, desiredIpRange)
			}
			return nil, fmt.Errorf("%w, assigned ip range %s-%s", ErrRestorationHashMismatch, ipRange.StartAddress, ipRange.EndAddress)
		}
	}

	//update ip range since it does exist
	ipRangeId := responseIpRange.Payload.Results[0].ID
	return r.UpdateIpRange(ipRangeId, desiredIpRange)
}

func (r *NetboxClient) GetIpRange(ipRange *models.IpRange) (*ipam.IpamIPRangesListOK, error) {

	requestIpRange := ipam.
		NewIpamIPRangesListParams().
		WithStartAddress(&ipRange.StartAddress).
		WithEndAddress(&ipRange.EndAddress)
	responseIpRange, err := r.Ipam.IpamIPRangesList(requestIpRange, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to fetch IpRange details", err)
	}

	return responseIpRange, err
}

func (r *NetboxClient) CreateIpRange(ipRange *netboxModels.WritableIPRange) (*netboxModels.IPRange, error) {
	requestCreateIpRange := ipam.
		NewIpamIPRangesCreateParams().
		WithDefaults().
		WithData(ipRange)
	responseCreateIpRange, err := r.Ipam.
		IpamIPRangesCreate(requestCreateIpRange, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to reserve IP Range", err)
	}
	return responseCreateIpRange.Payload, nil
}

func (r *NetboxClient) UpdateIpRange(ipRangeId int64, ipRange *netboxModels.WritableIPRange) (*netboxModels.IPRange, error) {
	requestUpdateIpRange := ipam.
		NewIpamIPRangesUpdateParams().
		WithDefaults().
		WithData(ipRange).
		WithID(ipRangeId)
	responseUpdateIp, err := r.Ipam.IpamIPRangesUpdate(requestUpdateIpRange, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to update IP Range", err)
	}
	return responseUpdateIp.Payload, nil
}

func (r *NetboxClient) DeleteIpRange(ipRangeId int64) error {
	requestDeleteIpRange := ipam.NewIpamIPRangesDeleteParams().WithID(ipRangeId)
	_, err := r.Ipam.IpamIPRangesDelete(requestDeleteIpRange, nil)
	if err != nil {
		switch typedErr := err.(type) {
		case *ipam.IpamIPRangesDeleteDefault:
			if typedErr.IsCode(http.StatusNotFound) {
				return nil
			}
			return utils.NetboxError("Failed to delete ip range from Netbox", err)
		default:
			return utils.NetboxError("Failed to delete ip range from Netbox", err)
		}
	}
	return nil
}

func ReserveOrUpdateIpRange(ctx context.Context, cLegacy *NetboxClient, c *nclient.APIClient, ipRange *models.IpRange) (*nclient.IPRange, error) {
	responseIpRangeList, err := getIpRange(ctx, c, ipRange)
	if err != nil {
		return nil, err
	}

	desiredIpRange := nclient.NewWritableIPRangeRequest(ipRange.StartAddress, ipRange.EndAddress)
	desiredIpRange.SetStatus("active")

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
			tenantID := int32(tenantDetails.Id)
			desiredIpRange.SetTenant(nclient.Int32AsASNRangeRequestTenant(&tenantID))
		}
	}

	// create ip range since it doesn't exist
	if len(responseIpRangeList.Results) == 0 {
		return createIpRange(ctx, c, desiredIpRange)
	}

	ipRangeToUpdate := responseIpRangeList.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if ipRange.Metadata != nil {
		if restorationHash, ok := ipRange.Metadata.Custom[restorationHashKey]; ok {
			if ipRangeToUpdate.CustomFields != nil && ipRangeToUpdate.CustomFields[restorationHashKey] == restorationHash {
				//update ip address since it does exist and the restoration hash matches
				return updateIpRange(ctx, c, ipRangeToUpdate.Id, desiredIpRange)
			}
			return nil, fmt.Errorf("%w, assigned ip range %s-%s", ErrRestorationHashMismatch, ipRange.StartAddress, ipRange.EndAddress)
		}
	}

	//update ip range since it does exist
	ipRangeId := responseIpRangeList.Results[0].Id
	return updateIpRange(ctx, c, ipRangeId, desiredIpRange)
}

func getIpRange(ctx context.Context, c *nclient.APIClient, ipRange *models.IpRange) (*nclient.PaginatedIPRangeList, error) {
	req := c.IpamAPI.IpamIpRangesList(ctx).
		StartAddress([]string{ipRange.StartAddress}).
		EndAddress([]string{ipRange.StartAddress})
	resp, httpResp, err := req.Execute()
	if err != nil {
		return nil, utils.NetboxError("failed to fetch IpRange details", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, utils.NetboxError("failed to fetch IpRange details", errors.New(httpResp.Status))
	}

	return resp, err
}

func createIpRange(ctx context.Context, c *nclient.APIClient, ipRange *nclient.WritableIPRangeRequest) (*nclient.IPRange, error) {
	req := c.IpamAPI.IpamIpRangesCreate(ctx).WritableIPRangeRequest(*ipRange)
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer httpResp.Body.Close()
	}

	if err != nil {
		return nil, utils.NetboxError("failed to reserve IP Range", err)
	}

	if httpResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to reserve IP Range: unexpected status %d, body: %s", httpResp.StatusCode, httpResp.Body)
	}

	return resp, nil
}

func updateIpRange(ctx context.Context, c *nclient.APIClient, ipRangeId int32, ipRange *nclient.WritableIPRangeRequest) (*nclient.IPRange, error) {
	req := c.IpamAPI.IpamIpRangesUpdate(ctx, ipRangeId).WritableIPRangeRequest(*ipRange)
	resp, httpResp, err := req.Execute()
	if err != nil {
		return nil, utils.NetboxError("failed to update IP Range", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, utils.NetboxError("failed to update IP Range", errors.New(httpResp.Status))
	}

	return resp, nil
}

func DeleteIpRange(ctx context.Context, c *nclient.APIClient, ipRangeId int64) error {
	req := c.IpamAPI.IpamIpRangesDestroy(ctx, int32(ipRangeId))
	httpResp, err := req.Execute()
	if err != nil {
		return utils.NetboxError("failed to delete ip range from Netbox", err)
	}
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNotFound {
		return utils.NetboxError("failed to delete ip range from Netbox", errors.New(httpResp.Status))
	}

	return nil
}
