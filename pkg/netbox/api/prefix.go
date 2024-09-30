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
	"strings"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"

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
		Description:  prefix.Metadata.Description + warningComment,
		Status:       "active",
	}

	if prefix.Metadata.Tenant != "" {
		tenantDetails, err := r.GetTenantDetails(prefix.Metadata.Tenant)
		if err != nil {
			return nil, err
		}
		desiredPrefix.Tenant = &tenantDetails.Id
	}

	// create prefix since it doesn't exist
	if len(responsePrefix.Payload.Results) == 0 {
		return r.CreatePrefix(desiredPrefix)
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
	// assuming id starts from 1 when Prefix is created so 0 means that Prefix doesn't exist
	if prefixId == 0 {
		return nil
	}

	requestDeletePrefix := ipam.NewIpamPrefixesDeleteParams().WithID(prefixId)
	_, err := r.Ipam.IpamPrefixesDelete(requestDeletePrefix, nil)
	if err != nil {
		if strings.Contains(err.Error(), "No Prefix matches the given query.") {
			return utils.NetboxNotFoundError("Prefix")
		}
		return utils.NetboxError("failed to delete Prefix", err)
	}
	return nil
}
