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
	apismeta "k8s.io/apimachinery/pkg/api/meta"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

/*
ReserveOrUpdatePrefix creates or updates the prefix passed as parameter
*/
func (c *NetboxCompositeClient) ReserveOrUpdatePrefix(ctx context.Context, prefix *models.Prefix, prefixV1 *netboxv1.Prefix) (*v4client.Prefix, error) {
	responsePrefix, err := c.getPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	// create prefix since it doesn't exist
	if len(responsePrefix.Results) == 0 {
		return c.createPrefix(ctx, prefix)
	}

	prefixToUpdate := &responsePrefix.Results[0]

	// if the desired ip address has a restoration hash
	// check that the ip address to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if prefix.Metadata != nil {
		if restorationHash, ok := prefix.Metadata.Custom[restorationHashKey]; ok {
			if prefixToUpdate.CustomFields != nil && prefixToUpdate.CustomFields[restorationHashKey] == restorationHash {
				// compare the LastUpdated of IPAddress in NetBox with the LastUpdated in the K8s status
				sameLastUpdated := (!prefixToUpdate.LastUpdated.IsSet()) == (prefixV1.Status.LastUpdated == nil) &&
					(!prefixToUpdate.LastUpdated.IsSet() || prefixV1.Status.LastUpdated.Time.Equal(*prefixToUpdate.LastUpdated.Get()))

				// and the observed generation of the ready condition
				readyCondition := apismeta.FindStatusCondition(prefixV1.Status.Conditions, "Ready")
				sameGeneration := readyCondition != nil && readyCondition.Status == "True" && readyCondition.ObservedGeneration == prefixV1.Generation
				if sameLastUpdated && sameGeneration {
					return nil, nil
				}

				//update ip address since it does exist and the restoration hash matches
				return c.updatePrefix(ctx, prefixToUpdate, prefix)
			}
			return nil, fmt.Errorf("%w, assigned prefix %s", ErrRestorationHashMismatch, prefix.Prefix)
		}
	}

	// compare the LastUpdated of IPAddress in NetBox with the LastUpdated in the K8s status
	sameLastUpdated := (!prefixToUpdate.LastUpdated.IsSet()) == (prefixV1.Status.LastUpdated == nil) &&
		(!prefixToUpdate.LastUpdated.IsSet() || prefixV1.Status.LastUpdated.Time.Equal(*prefixToUpdate.LastUpdated.Get()))

	// and the observed generation of the ready condition
	readyCondition := apismeta.FindStatusCondition(prefixV1.Status.Conditions, "Ready")
	sameGeneration := readyCondition != nil && readyCondition.Status == "True" && readyCondition.ObservedGeneration == prefixV1.Generation
	if sameLastUpdated && sameGeneration {
		return nil, nil
	}

	//update ip address since it does exist
	return c.updatePrefix(ctx, prefixToUpdate, prefix)
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

func (c *NetboxCompositeClient) updatePrefix(ctx context.Context, prefixToUpdate *v4client.Prefix, prefix *models.Prefix) (resp *v4client.Prefix, err error) {
	isLegacy, err := c.clientV4.isLegacyNetBox(ctx)
	if err != nil {
		return nil, err
	}

	if isLegacy {
		desiredPrefix, err := c.buildWritablePrefixRequestV3(prefix)
		if err != nil {
			return nil, err
		}

		// needsUpdate := utils.NeedsUpdate(
		// 	prefixToUpdate,
		// 	desiredPrefix,
		// 	func(current *v4client.Prefix, desired *netboxModels.WritablePrefix) bool {
		// 		return current.GetDescription() != desired.Description
		// 	},
		// 	func(current *v4client.Prefix, desired *netboxModels.WritablePrefix) bool {
		// 		return current.GetComments() != desired.Comments
		// 	},
		// 	func(current *v4client.Prefix, desired *netboxModels.WritablePrefix) bool {
		// 		return string(current.Status.GetValue()) != desired.Status
		// 	},
		// 	func(current *v4client.Prefix, desired *netboxModels.WritablePrefix) bool {
		// 		return utils.CompareCustomFields(
		// 			current.GetCustomFields(),
		// 			utils.NormalizeCustomFields(desired.CustomFields),
		// 		)
		// 	},
		// 	func(current *v4client.Prefix, desired *netboxModels.WritablePrefix) bool {
		// 		return current.GetScopeId() != int32(*desired.Site)
		// 	},
		// )

		// if !needsUpdate {
		// 	return prefixToUpdate, nil
		// }

		return c.clientV3.updatePrefixV3(int64(prefixToUpdate.Id), desiredPrefix)
	}

	desiredPrefix, err := c.writablePrefixRequestV4(prefix)
	if err != nil {
		return nil, err
	}

	// needsUpdate := utils.NeedsUpdate(
	// 	prefixToUpdate,
	// 	desiredPrefix,
	// 	func(current *v4client.Prefix, desired *v4client.WritablePrefixRequest) bool {
	// 		return current.GetDescription() != desired.GetDescription()
	// 	},
	// 	func(current *v4client.Prefix, desired *v4client.WritablePrefixRequest) bool {
	// 		return current.GetComments() != desired.GetComments()
	// 	},
	// 	func(current *v4client.Prefix, desired *v4client.WritablePrefixRequest) bool {
	// 		return string(*current.Status.Value) != string(*desired.Status)
	// 	},
	// 	func(current *v4client.Prefix, desired *v4client.WritablePrefixRequest) bool {
	// 		return utils.CompareCustomFields(
	// 			current.GetCustomFields(),
	// 			desired.GetCustomFields(),
	// 		)
	// 	},
	// 	func(current *v4client.Prefix, desired *v4client.WritablePrefixRequest) bool {
	// 		return current.GetScopeType() != desired.GetScopeType() || current.GetScopeId() != desired.GetScopeId()
	// 	},
	// )

	// if !needsUpdate {
	// 	return prefixToUpdate, nil
	// }
	return c.clientV4.updatePrefixV4(ctx, prefixToUpdate.Id, desiredPrefix)
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
