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

/*
ReserveOrUpdatePrefix creates or updates the prefix passed as parameter
*/
func (r *NetboxClient) ReserveOrUpdatePrefix(prefix *models.Prefix) (*netboxModels.Prefix, error) {
	responsePrefix, err := r.GetPrefix(prefix)
	if err != nil {
		return nil, err
	}

	desiredPrefix := &netboxModels.WritablePrefix{
		Prefix:       &prefix.Prefix,
		Comments:     prefix.Metadata.Comments + warningComment,
		CustomFields: prefix.Metadata.Custom,
		Description:  prefix.Metadata.Description,
		Status:       "active",
	}

	if prefix.Metadata != nil {
		desiredPrefix.CustomFields = prefix.Metadata.Custom
		desiredPrefix.Comments = prefix.Metadata.Comments + warningComment
	}

	if prefix.Metadata != nil && prefix.Metadata.Tenant != "" {
		tenantDetails, err := r.GetTenantDetails(prefix.Metadata.Tenant)
		if err != nil {
			return nil, err
		}
		desiredPrefix.Tenant = &tenantDetails.Id
	}

	if prefix.Metadata != nil && prefix.Metadata.Site != "" {
		siteDetails, err := r.GetSiteDetails(prefix.Metadata.Site)
		if err != nil {
			return nil, err
		}
		desiredPrefix.Site = &siteDetails.Id
	}

	// create prefix since it doesn't exist
	if len(responsePrefix.Payload.Results) == 0 {
		return r.CreatePrefix(desiredPrefix)
	}

	prefixToUpdate := responsePrefix.Payload.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if prefix.Metadata != nil {
		if restorationHash, ok := prefix.Metadata.Custom[restorationHashKey]; ok {
			if prefixToUpdate.CustomFields != nil && prefixToUpdate.CustomFields.(map[string]interface{})[restorationHashKey] == restorationHash {
				//update ip address since it does exist and the restoration hash matches
				return r.UpdatePrefix(prefixToUpdate.ID, desiredPrefix)
			}
			return nil, fmt.Errorf("%w, assigned prefix %s", ErrRestorationHashMismatch, prefix.Prefix)
		}
	}

	//update ip address since it does exist
	prefixId := responsePrefix.Payload.Results[0].ID
	return r.UpdatePrefix(prefixId, desiredPrefix)
}

func (r *NetboxClient) GetPrefix(prefix *models.Prefix) (*ipam.IpamPrefixesListOK, error) {

	requestPrefix := ipam.
		NewIpamPrefixesListParams().
		WithPrefix(&prefix.Prefix)
	responsePrefix, err := r.Ipam.IpamPrefixesList(requestPrefix, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to fetch Prefix details", err)
	}

	return responsePrefix, err
}

func (r *NetboxClient) CreatePrefix(prefix *netboxModels.WritablePrefix) (*netboxModels.Prefix, error) {
	requestCreatePrefix := ipam.
		NewIpamPrefixesCreateParams().
		WithDefaults().
		WithData(prefix)
	responseCreatePrefix, err := r.Ipam.
		IpamPrefixesCreate(requestCreatePrefix, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to create Prefix", err)
	}
	return responseCreatePrefix.Payload, nil
}

func (r *NetboxClient) UpdatePrefix(prefixId int64, prefix *netboxModels.WritablePrefix) (*netboxModels.Prefix, error) {
	requestUpdatePrefix := ipam.NewIpamPrefixesUpdateParams().
		WithDefaults().
		WithData(prefix).
		WithID(prefixId)
	responseUpdatePrefix, err := r.Ipam.IpamPrefixesUpdate(requestUpdatePrefix, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to update Prefix", err)
	}
	return responseUpdatePrefix.Payload, nil
}

func (r *NetboxClient) DeletePrefix(prefixId int64) error {
	requestDeletePrefix := ipam.NewIpamPrefixesDeleteParams().WithID(prefixId)
	_, err := r.Ipam.IpamPrefixesDelete(requestDeletePrefix, nil)
	if err != nil {
		switch typedErr := err.(type) {
		case *ipam.IpamPrefixesDeleteDefault:
			if typedErr.IsCode(http.StatusNotFound) {
				return nil
			}
			return utils.NetboxError("Failed to delete prefix from Netbox", err)
		default:
			return utils.NetboxError("Failed to delete prefix from Netbox", err)
		}
	}
	return nil
}
