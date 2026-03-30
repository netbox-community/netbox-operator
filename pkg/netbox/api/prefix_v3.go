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
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

// to ensure compatibility with older NetBox versions the CreatePrefix and UpdatePrefix
// functions for the v3 client are still required

func (c *NetboxClientV3) createPrefixV3(prefix *netboxModels.WritablePrefix) (resp *v4client.Prefix, skipsUpdate bool, err error) {
	requestCreatePrefix := ipam.
		NewIpamPrefixesCreateParams().
		WithDefaults().
		WithData(prefix)
	responseCreatePrefix, err := c.Ipam.
		IpamPrefixesCreate(requestCreatePrefix, nil)
	if err != nil {
		return nil, true, utils.NetboxError("failed to create Prefix", err)
	}
	prefixPayload := responseCreatePrefix.Payload

	nclientPrefix := &v4client.Prefix{
		Id:          int32(prefixPayload.ID),
		Prefix:      *prefixPayload.Prefix,
		Description: &prefix.Description,
	}

	return nclientPrefix, false, nil
}

func (c *NetboxClientV3) updatePrefixV3(prefixId int64, prefix *netboxModels.WritablePrefix) (resp *v4client.Prefix, skipsUpdate bool, err error) {
	requestUpdatePrefix := ipam.NewIpamPrefixesUpdateParams().
		WithDefaults().
		WithData(prefix).
		WithID(prefixId)
	responseUpdatePrefix, err := c.Ipam.IpamPrefixesUpdate(requestUpdatePrefix, nil)
	if err != nil {
		return nil, true, utils.NetboxError("failed to update Prefix", err)
	}
	prefixPayload := responseUpdatePrefix.Payload

	nclientPrefix := &v4client.Prefix{
		Id:          int32(prefixPayload.ID),
		Prefix:      *prefixPayload.Prefix,
		Description: &prefix.Description,
	}

	return nclientPrefix, false, nil
}

func (c *NetboxCompositeClient) buildWritablePrefixRequestV3(prefix *models.Prefix) (*netboxModels.WritablePrefix, error) {
	desiredPrefix := &netboxModels.WritablePrefix{
		Prefix:       &prefix.Prefix,
		Comments:     prefix.Metadata.Comments + warningComment,
		CustomFields: prefix.Metadata.Custom,
		Description:  prefix.Metadata.Description + warningComment,
		Status:       "active",
	}
	if prefix.Metadata != nil {
		desiredPrefix.CustomFields = prefix.Metadata.Custom
		desiredPrefix.Comments = prefix.Metadata.Comments + warningComment
		desiredPrefix.Description = TruncateDescription(prefix.Metadata.Description)

		if prefix.Metadata.Tenant != "" {
			tenantDetails, err := c.getTenantDetails(prefix.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			desiredPrefix.Tenant = &tenantDetails.Id
		}

		if prefix.Metadata.Site != "" {
			siteDetails, err := c.getSiteDetails(prefix.Metadata.Site)
			if err != nil {
				return nil, err
			}
			desiredPrefix.Site = &siteDetails.Id
		}
	}
	return desiredPrefix, nil
}
