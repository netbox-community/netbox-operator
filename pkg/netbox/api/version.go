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

	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
	"golang.org/x/mod/semver"
)

func (c *NetboxClientV4) GetNetBoxVersion(ctx context.Context) (version string, err error) {

	req := c.StatusAPI.StatusRetrieve(ctx)
	resp, httpResp, err := req.Execute()

	if httpResp != nil {
		defer func() {
			errClose := httpResp.Body.Close()
			err = errors.Join(err, errClose)
		}()
	}

	version = resp["netbox-version"].(string)

	if err != nil {
		return "", utils.NetboxError("failed to fetch netbox version", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to fetch netbox version: unexpected status %d, and failed to read body %w", httpResp.StatusCode, err)
		}
		return "", fmt.Errorf("failed to fetch netbox version: unexpected status %d, body: %s", httpResp.StatusCode, string(body))
	}

	return version, nil
}

func isLegacyVersion(version string) bool {
	v := version
	if v != "" && v[0] != 'v' {
		v = "v" + v
	}
	// v4+ uses scope; v3 uses site
	return semver.Compare(v, "v4.2.0") < 0
}

func (c *NetboxClientV4) isLegacyNetBox(ctx context.Context) (bool, error) {
	version, err := c.GetNetBoxVersion(ctx)
	if err != nil {
		return false, err
	}
	return isLegacyVersion(version), nil
}
