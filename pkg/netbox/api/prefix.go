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

/*
ReserveOrUpdatePrefix creates or updates the prefix passed as parameter
*/
func (c *NetboxCompositeClient) ReserveOrUpdatePrefix(ctx context.Context, prefix *models.Prefix, prefixV1 *netboxv1.Prefix) (resp *v4client.Prefix, isUpToDate bool, err error) {
	responsePrefix, err := c.getPrefix(ctx, prefix)
	if err != nil {
		return nil, false, err
	}

	// create prefix since it doesn't exist
	if len(responsePrefix.Results) == 0 {
		resp, err := c.createPrefix(ctx, prefix)
		return resp, false, err
	}

	prefixToUpdate := &responsePrefix.Results[0]

	if !prefixToUpdate.LastUpdated.IsSet() {
		return nil, false, fmt.Errorf("last updated field is not set in Netbox for prefix %s", prefix.Prefix)
	}

	// if the desired prefix has a restoration hash
	// check that the prefix to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if prefix.Metadata != nil {
		if restorationHash, ok := prefix.Metadata.Custom[restorationHashKey]; ok {
			if prefixToUpdate.CustomFields != nil && prefixToUpdate.CustomFields[restorationHashKey] == restorationHash {
				if IsUpToDate(*prefixToUpdate.LastUpdated.Get(), prefixV1.Status.LastUpdated, prefixV1.Status.Conditions, prefixV1.Generation) {
					return nil, true, nil
				}

				//update prefix since it does exist and the restoration hash matches
				resp, err := c.updatePrefix(ctx, prefixToUpdate.Id, prefix)
				if err != nil {
					return nil, false, err
				}
				return resp, false, nil
			}

			return nil, false, fmt.Errorf("%w, assigned prefix %s", ErrRestorationHashMismatch, prefix.Prefix)
		}
	}

	if IsUpToDate(*prefixToUpdate.LastUpdated.Get(), prefixV1.Status.LastUpdated, prefixV1.Status.Conditions, prefixV1.Generation) {
		return nil, true, nil
	}

	//update prefix since it does exist
	resp, err = c.updatePrefix(ctx, prefixToUpdate.Id, prefix)
	if err != nil {
		return nil, false, err
	}
	return resp, false, nil
}

func (c *NetboxCompositeClient) getPrefix(ctx context.Context, prefix *models.Prefix) (*v4client.PaginatedPrefixList, error) {
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

func (c *NetboxCompositeClient) createPrefix(ctx context.Context, prefix *models.Prefix) (*v4client.Prefix, error) {
	isLegacy, err := c.clientV4.isLegacyNetBox(ctx)
	if err != nil {
		return nil, err
	}

	if isLegacy {
		desiredPrefix, err := c.buildWritablePrefixRequestV3(prefix)
		if err != nil {
			return nil, err
		}

		return c.clientV3.createPrefixV3(desiredPrefix)
	}

	desiredPrefix, err := c.writablePrefixRequestV4(prefix)
	if err != nil {
		return nil, err
	}
	status, err := v4client.NewPatchedWritablePrefixRequestStatusFromValue("active")
	if err != nil {
		return nil, err
	}
	desiredPrefix.SetStatus(*status)
	return c.clientV4.createPrefixV4(ctx, desiredPrefix)
}

func (c *NetboxCompositeClient) updatePrefix(ctx context.Context, prefixId int32, prefix *models.Prefix) (resp *v4client.Prefix, err error) {
	isLegacy, err := c.clientV4.isLegacyNetBox(ctx)
	if err != nil {
		return nil, err
	}

	if isLegacy {
		desiredPrefix, err := c.buildWritablePrefixRequestV3(prefix)
		if err != nil {
			return nil, err
		}

		return c.clientV3.updatePrefixV3(int64(prefixId), desiredPrefix)
	}

	desiredPrefix, err := c.writablePrefixRequestV4(prefix)
	if err != nil {
		return nil, err
	}

	return c.clientV4.updatePrefixV4(ctx, prefixId, desiredPrefix)
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
