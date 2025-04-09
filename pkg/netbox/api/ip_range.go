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
	"net/http"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"

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

	if ipRange.Metadata.Tenant != "" {
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
