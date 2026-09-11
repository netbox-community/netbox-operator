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
	"context"
	"errors"
	"net/http"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (c *NetboxCompositeClient) getRirDetailsByName(ctx context.Context, name string) (rir *models.Rir, err error) {
	result, httpResp, execErr := c.clientV4.IpamAPI.IpamRirsList(ctx).
		Name([]string{name}).Limit(listPageSize).Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "fetch RIR details")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	if len(result.Results) == 0 {
		return nil, utils.NetboxNotFoundError("RIR '" + name + "'")
	}

	return &models.Rir{
		Id:   int64(result.Results[0].Id),
		Name: result.Results[0].Name,
		Slug: result.Results[0].Slug,
	}, nil
}
