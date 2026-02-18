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
	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

/*
ReserveOrUpdatePrefix creates or updates the prefix passed as parameter
*/
func (c *NetboxCompositeClient) ReserveOrUpdatePrefix(ctx context.Context, prefix *models.Prefix) (*v4client.Prefix, error) {
	responsePrefix, err := c.getPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	// create prefix since it doesn't exist
	if len(responsePrefix.Results) == 0 {
		return c.createPrefix(ctx, prefix)
	}

	prefixToUpdate := responsePrefix.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if prefix.Metadata != nil {
		if restorationHash, ok := prefix.Metadata.Custom[restorationHashKey]; ok {
			if prefixToUpdate.CustomFields != nil && prefixToUpdate.CustomFields[restorationHashKey] == restorationHash {
				//update ip address since it does exist and the restoration hash matches
				return c.updatePrefix(ctx, prefixToUpdate.Id, prefix)
			}
			return nil, fmt.Errorf("%w, assigned prefix %s", ErrRestorationHashMismatch, prefix.Prefix)
		}
	}

	//update ip address since it does exist
	prefixId := responsePrefix.Results[0].Id
	return c.updatePrefix(ctx, prefixId, prefix)
}

func (c *NetboxCompositeClient) getPrefix(ctx context.Context, prefix *models.Prefix) (resp *v4client.PaginatedPrefixList, err error) {
	req := c.clientV4.IpamAPI.IpamPrefixesList(ctx).
		Prefix([]string{prefix.Prefix})
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
		return nil, fmt.Errorf("failed to fetch prefix details: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		if readErr != nil {
			return nil, fmt.Errorf("failed to fetch prefix details: status %d; read body: %w", httpResp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to fetch prefix details: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	if err != nil {
		return nil, utils.NetboxError("failed to fetch prefix details", err)
	}

	return resp, nil
}

func (c *NetboxCompositeClient) createPrefix(ctx context.Context, prefix *models.Prefix) (resp *v4client.Prefix, err error) {
	isLegacy, err := c.clientV4.isLegacyNetBox(ctx)
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
		return c.clientV3.createPrefixV3(desiredPrefix)
	}

	desiredPrefix := v4client.NewWritablePrefixRequest(prefix.Prefix)

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
			tenantDetails, err := c.getTenantDetails(prefix.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredPrefix.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}

		if prefix.Metadata.Site != "" {
			siteDetails, err := c.getSiteDetails(prefix.Metadata.Site)
			if err != nil {
				return nil, err
			}
			desiredPrefix.SetScopeType("dcim.site")
			desiredPrefix.SetScopeId(int32(siteDetails.Id))
		}
	}

	status, err := v4client.NewPatchedWritablePrefixRequestStatusFromValue("active")
	if err != nil {
		return nil, err
	}
	desiredPrefix.SetStatus(*status)
	return c.clientV4.createPrefixV4(ctx, desiredPrefix)

}

func (c *NetboxClientV4) createPrefixV4(ctx context.Context, prefix *v4client.WritablePrefixRequest) (resp *v4client.Prefix, err error) {
	req := c.IpamAPI.IpamPrefixesCreate(ctx).WritablePrefixRequest(*prefix)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "create prefix")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) updatePrefix(ctx context.Context, prefixId int32, prefix *models.Prefix) (resp *v4client.Prefix, err error) {
	isLegacy, err := c.clientV4.isLegacyNetBox(ctx)
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
		return c.clientV3.updatePrefixV3(int64(prefixId), desiredPrefix)
	}

	desiredPrefix := v4client.NewWritablePrefixRequest(prefix.Prefix)

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
			tenantDetails, err := c.getTenantDetails(prefix.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredPrefix.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}

		if prefix.Metadata.Site != "" {
			siteDetails, err := c.getSiteDetails(prefix.Metadata.Site)
			if err != nil {
				return nil, err
			}
			desiredPrefix.SetScopeType("dcim.site")
			desiredPrefix.SetScopeId(int32(siteDetails.Id))
		}
	}
	return c.clientV4.updatePrefixV4(ctx, prefixId, desiredPrefix)

}

func (c *NetboxClientV4) updatePrefixV4(ctx context.Context, prefixId int32, prefix *v4client.WritablePrefixRequest) (resp *v4client.Prefix, err error) {
	req := c.IpamAPI.IpamPrefixesUpdate(ctx, prefixId).WritablePrefixRequest(*prefix)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update prefix")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) DeletePrefix(ctx context.Context, prefixId int32) (err error) {
	req := c.clientV4.IpamAPI.IpamPrefixesDestroy(ctx, prefixId)
	httpResp, execErr := req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusNoContent, "delete prefix")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return handleErr
	}

	return nil
}
