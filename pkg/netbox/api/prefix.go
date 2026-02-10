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

	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	nclient "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

/*
ReserveOrUpdatePrefix creates or updates the prefix passed as parameter
*/
func (c *NetboxClientV4) ReserveOrUpdatePrefix(ctx context.Context, cLegacy *NetboxClient, prefix *models.Prefix) (*nclient.Prefix, error) {
	responsePrefix, err := c.GetPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	// create prefix since it doesn't exist
	if len(responsePrefix.Results) == 0 {
		return c.createPrefix(ctx, cLegacy, prefix)
	}

	prefixToUpdate := responsePrefix.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if prefix.Metadata != nil {
		if restorationHash, ok := prefix.Metadata.Custom[restorationHashKey]; ok {
			if prefixToUpdate.CustomFields != nil && prefixToUpdate.CustomFields[restorationHashKey] == restorationHash {
				//update ip address since it does exist and the restoration hash matches
				return c.updatePrefix(ctx, cLegacy, prefixToUpdate.Id, prefix)
			}
			return nil, fmt.Errorf("%w, assigned prefix %s", ErrRestorationHashMismatch, prefix.Prefix)
		}
	}

	//update ip address since it does exist
	prefixId := responsePrefix.Results[0].Id
	return c.updatePrefix(ctx, cLegacy, prefixId, prefix)
}

func (c *NetboxClientV4) GetPrefix(ctx context.Context, prefix *models.Prefix) (resp *nclient.PaginatedPrefixList, err error) {
	req := c.IpamAPI.IpamPrefixesList(ctx).
		Prefix([]string{prefix.Prefix})
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer func() {
			errClose := httpResp.Body.Close()
			err = errors.Join(err, errClose)
		}()
	}

	if err != nil {
		return nil, utils.NetboxError("failed to fetch prefix details", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(httpResp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch prefix details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to fetch prefix details: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *NetboxClientV4) createPrefix(ctx context.Context, cLegacy *NetboxClient, prefix *models.Prefix) (resp *nclient.Prefix, err error) {
	isLegacy, err := c.isLegacyNetBox(ctx)
	if err != nil {
		return nil, err
	}

	if isLegacy {

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
				tenantDetails, err := cLegacy.GetTenantDetails(prefix.Metadata.Tenant)
				if err != nil {
					return nil, err
				}
				desiredPrefix.Tenant = &tenantDetails.Id
			}
			if prefix.Metadata.Site != "" {
				siteDetails, err := cLegacy.GetSiteDetails(prefix.Metadata.Site)
				if err != nil {
					return nil, err
				}
				desiredPrefix.Site = &siteDetails.Id
			}
		}
		return cLegacy.createPrefixV3(desiredPrefix)
	}

	desiredPrefix := nclient.NewWritablePrefixRequest(prefix.Prefix)

	if prefix.Metadata != nil {
		desiredPrefix.SetComments(prefix.Metadata.Comments + warningComment)
		// Convert map[string]string to map[string]interface{}
		customFields := make(map[string]interface{}, len(prefix.Metadata.Custom))
		for k, v := range prefix.Metadata.Custom {
			customFields[k] = v
		}
		desiredPrefix.SetCustomFields(customFields)
		desiredPrefix.SetDescription(TruncateDescription(prefix.Metadata.Description))

		if prefix.Metadata.Tenant != "" {
			tenantDetails, err := cLegacy.GetTenantDetails(prefix.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredPrefix.SetTenant(nclient.Int32AsASNRangeRequestTenant(&tenantId))
		}

		if prefix.Metadata.Site != "" {
			siteDetails, err := cLegacy.GetSiteDetails(prefix.Metadata.Site)
			if err != nil {
				return nil, err
			}
			desiredPrefix.SetScopeType("dcim.site")
			desiredPrefix.SetScopeId(int32(siteDetails.Id))
		}
	}

	status, err := nclient.NewPatchedWritablePrefixRequestStatusFromValue("active")
	if err != nil {
		return nil, err
	}
	desiredPrefix.SetStatus(*status)
	return c.createPrefixV4(ctx, desiredPrefix)

}

func (c *NetboxClientV4) createPrefixV4(ctx context.Context, prefix *nclient.WritablePrefixRequest) (resp *nclient.Prefix, err error) {
	req := c.IpamAPI.IpamPrefixesCreate(ctx).WritablePrefixRequest(*prefix)
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer func() {
			errClose := httpResp.Body.Close()
			err = errors.Join(err, errClose)
		}()
	}

	if err != nil {
		return nil, utils.NetboxError("failed to reserve prefix", err)
	}

	if httpResp.StatusCode != http.StatusCreated {
		body, readErr := io.ReadAll(httpResp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch prefix details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to reserve prefix: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *NetboxClientV4) updatePrefix(ctx context.Context, cLegacy *NetboxClient, prefixId int32, prefix *models.Prefix) (resp *nclient.Prefix, err error) {
	isLegacy, err := c.isLegacyNetBox(ctx)
	if err != nil {
		return nil, err
	}

	if isLegacy {

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
		}

		if prefix.Metadata != nil && prefix.Metadata.Tenant != "" {
			tenantDetails, err := cLegacy.GetTenantDetails(prefix.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			desiredPrefix.Tenant = &tenantDetails.Id
		}

		if prefix.Metadata != nil && prefix.Metadata.Site != "" {
			siteDetails, err := cLegacy.GetSiteDetails(prefix.Metadata.Site)
			if err != nil {
				return nil, err
			}
			desiredPrefix.Site = &siteDetails.Id
		}
		return cLegacy.updatePrefixV3(int64(prefixId), desiredPrefix)
	}

	desiredPrefix := nclient.NewWritablePrefixRequest(prefix.Prefix)

	if prefix.Metadata != nil {
		desiredPrefix.SetComments(prefix.Metadata.Comments + warningComment)
		// Convert map[string]string to map[string]interface{}
		customFields := make(map[string]interface{}, len(prefix.Metadata.Custom))
		for k, v := range prefix.Metadata.Custom {
			customFields[k] = v
		}
		desiredPrefix.SetCustomFields(customFields)
		desiredPrefix.SetDescription(prefix.Metadata.Description)

		if prefix.Metadata.Tenant != "" {
			tenantDetails, err := cLegacy.GetTenantDetails(prefix.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredPrefix.SetTenant(nclient.Int32AsASNRangeRequestTenant(&tenantId))
		}

		if prefix.Metadata != nil && prefix.Metadata.Site != "" {
			siteDetails, err := cLegacy.GetSiteDetails(prefix.Metadata.Site)
			if err != nil {
				return nil, err
			}
			desiredPrefix.SetScopeType("dcim.site")
			desiredPrefix.SetScopeId(int32(siteDetails.Id))
		}
	}
	return c.updatePrefixV4(ctx, prefixId, desiredPrefix)

}

func (c *NetboxClientV4) updatePrefixV4(ctx context.Context, prefixId int32, prefix *nclient.WritablePrefixRequest) (resp *nclient.Prefix, err error) {
	req := c.IpamAPI.IpamPrefixesUpdate(ctx, prefixId).WritablePrefixRequest(*prefix)
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer func() {
			errClose := httpResp.Body.Close()
			err = errors.Join(err, errClose)
		}()
	}

	if err != nil {
		return nil, utils.NetboxError("failed to update prefix", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(httpResp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch prefix details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to update prefix: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *NetboxClientV4) DeletePrefix(ctx context.Context, prefixId int32) (err error) {
	req := c.IpamAPI.IpamPrefixesDestroy(ctx, prefixId)
	httpResp, err := req.Execute()

	if httpResp != nil {
		defer func() {
			errClose := httpResp.Body.Close()
			err = errors.Join(err, errClose)
		}()
	}

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return nil
		}
		return utils.NetboxError("failed to delete prefix from Netbox", err)
	}
	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusNotFound {
		body, readErr := io.ReadAll(httpResp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to fetch prefix details: unexpected status %d, and failed to read body %w", httpResp.StatusCode, readErr)
		}
		return fmt.Errorf("failed to delete prefix from Netbox: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return nil
}
