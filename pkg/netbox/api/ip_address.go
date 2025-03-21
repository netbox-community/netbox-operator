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
	"fmt"
	"net/http"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (r *NetboxClient) ReserveOrUpdateIpAddress(ipAddress *models.IPAddress) (*netboxModels.IPAddress, error) {
	responseIpAddress, err := r.GetIpAddress(ipAddress)
	if err != nil {
		return nil, err
	}

	desiredIPAddress := &netboxModels.WritableIPAddress{
		Address:     &ipAddress.IpAddress,
		Description: TruncateDescription(""),
		Status:      "active",
	}

	if ipAddress.Metadata != nil {
		desiredIPAddress.CustomFields = ipAddress.Metadata.Custom
		desiredIPAddress.Comments = ipAddress.Metadata.Comments + warningComment
		desiredIPAddress.Description = TruncateDescription(ipAddress.Metadata.Description)
	}

	if ipAddress.Metadata != nil && ipAddress.Metadata.Tenant != "" {
		tenantDetails, err := r.GetTenantDetails(ipAddress.Metadata.Tenant)
		if err != nil {
			return nil, err
		}
		desiredIPAddress.Tenant = &tenantDetails.Id
	}

	// create ip address since it doesn't exist
	if len(responseIpAddress.Payload.Results) == 0 {
		return r.CreateIpAddress(desiredIPAddress)
	}

	ipToUpdate := responseIpAddress.Payload.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if ipAddress.Metadata != nil {
		if restorationHash, ok := ipAddress.Metadata.Custom[restorationHashKey]; ok {
			if ipToUpdate.CustomFields != nil && ipToUpdate.CustomFields.(map[string]interface{})[restorationHashKey] == restorationHash {
				//update ip address since it does exist and the restoration hash matches
				return r.UpdateIpAddress(ipToUpdate.ID, desiredIPAddress)
			}
			return nil, fmt.Errorf("%w, assigned ip address %s", ErrRestorationHashMismatch, ipAddress.IpAddress)
		}
	}

	ipAddressId := responseIpAddress.Payload.Results[0].ID
	return r.UpdateIpAddress(ipAddressId, desiredIPAddress)
}

func (r *NetboxClient) GetIpAddress(ipAddress *models.IPAddress) (*ipam.IpamIPAddressesListOK, error) {

	requestIpAddress := ipam.
		NewIpamIPAddressesListParams().
		WithAddress(&ipAddress.IpAddress)
	responseIpAddress, err := r.Ipam.IpamIPAddressesList(requestIpAddress, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to fetch IpAddress details", err)
	}

	return responseIpAddress, err
}

func (r *NetboxClient) CreateIpAddress(ipAddress *netboxModels.WritableIPAddress) (*netboxModels.IPAddress, error) {
	requestCreateIp := ipam.
		NewIpamIPAddressesCreateParams().
		WithDefaults().
		WithData(ipAddress)
	responseCreateIp, err := r.Ipam.
		IpamIPAddressesCreate(requestCreateIp, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to reserve IP Address", err)
	}
	return responseCreateIp.Payload, nil
}

func (r *NetboxClient) UpdateIpAddress(ipAddressId int64, ipAddress *netboxModels.WritableIPAddress) (*netboxModels.IPAddress, error) {
	requestUpdateIp := ipam.
		NewIpamIPAddressesUpdateParams().
		WithDefaults().
		WithData(ipAddress).
		WithID(ipAddressId)
	responseUpdateIp, err := r.Ipam.IpamIPAddressesUpdate(requestUpdateIp, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to update IP Address", err)
	}
	return responseUpdateIp.Payload, nil
}

func (r *NetboxClient) DeleteIpAddress(ipAddressId int64) error {
	requestDeleteIp := ipam.NewIpamIPAddressesDeleteParams().WithID(ipAddressId)
	_, err := r.Ipam.IpamIPAddressesDelete(requestDeleteIp, nil)
	if err != nil {
		switch typedErr := err.(type) {
		case *ipam.IpamIPAddressesDeleteDefault:
			if typedErr.IsCode(http.StatusNotFound) {
				return nil
			}
			return utils.NetboxError("Failed to delete ip address from Netbox", err)
		default:
			return utils.NetboxError("Failed to delete ip address from Netbox", err)
		}
	}
	return nil
}
