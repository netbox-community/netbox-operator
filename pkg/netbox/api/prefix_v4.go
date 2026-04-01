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
	"net/http"

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

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

func (c *NetboxClientV4) updatePrefixV4(ctx context.Context, prefixId int32, prefix *v4client.WritablePrefixRequest) (resp *v4client.Prefix, isUpToDate bool, err error) {
	req := c.IpamAPI.IpamPrefixesUpdate(ctx, prefixId).WritablePrefixRequest(*prefix)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update prefix")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, true, handleErr
	}

	return resp, false, nil
}

func (c *NetboxCompositeClient) writablePrefixRequestV4(prefix *models.Prefix) (*v4client.WritablePrefixRequest, error) {
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
	return desiredPrefix, nil
}
