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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

// allocatePrefix creates a child prefix of the given parent in a single atomic NetBox call
// (POST /api/ipam/prefixes/{id}/available-prefixes/) and returns the allocated prefix.
// The generated clients cannot express this request: they always serialize the required
// "prefix" field and have no "prefix_length" field. It works for both legacy and current NetBox,
// the caller is responsible for building a body matching the NetBox version.
func (c *NetboxClientV4) allocatePrefix(ctx context.Context, parentPrefixId int32, body map[string]any) (prefix string, err error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	cfg := c.client.GetConfig()
	url := fmt.Sprintf("%s://%s/api/ipam/prefixes/%d/available-prefixes/", cfg.Scheme, cfg.Host, parentPrefixId)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	for k, v := range cfg.DefaultHeader {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	httpResp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to allocate prefix: %w", err)
	}
	defer func() { err = errors.Join(err, httpResp.Body.Close()) }()

	respBody, readErr := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode == http.StatusConflict {
		// NetBox answers 409 if the parent has no space left for the requested prefix length
		return "", fmt.Errorf("%w: %s", ErrNoPrefixMatchsSizeCriteria, string(respBody))
	}
	if httpResp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to allocate prefix: status %d, body: %s", httpResp.StatusCode, string(respBody))
	}
	if readErr != nil {
		return "", fmt.Errorf("failed to read allocated prefix response: %w", readErr)
	}

	// NetBox returns a single object when a single object was posted
	var allocated struct {
		Prefix string `json:"prefix"`
	}
	if err := json.Unmarshal(respBody, &allocated); err != nil {
		return "", fmt.Errorf("failed to parse allocated prefix response: %w", err)
	}
	if allocated.Prefix == "" {
		return "", errors.New("NetBox returned an empty prefix for the allocation")
	}
	return allocated.Prefix, nil
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
