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
	"fmt"
	"math"
	"net/http"

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

func (c *NetboxCompositeClient) ReserveOrUpdateRouteTarget(ctx context.Context, routeTarget *models.RouteTarget, routeTargetV1 *netboxv1.RouteTarget) (resp *v4client.RouteTarget, isUpToDate bool, err error) {
	responseRouteTargetList, err := c.getRouteTarget(ctx, routeTarget.Name)
	if err != nil {
		return nil, false, err
	}

	desiredRouteTarget := v4client.NewRouteTargetRequest(routeTarget.Name)

	if routeTarget.Metadata != nil {
		desiredRouteTarget.SetComments(routeTarget.Metadata.Comments + warningComment)
		// Convert map[string]string to map[string]interface{}
		customFields := make(map[string]interface{}, len(routeTarget.Metadata.Custom))
		for k, v := range routeTarget.Metadata.Custom {
			customFields[k] = v
		}
		desiredRouteTarget.SetCustomFields(customFields)
		desiredRouteTarget.SetDescription(TruncateDescription(routeTarget.Metadata.Description))
		if routeTarget.Metadata.Tenant != "" {
			tenantDetails, err := c.getTenantDetails(routeTarget.Metadata.Tenant)
			if err != nil {
				return nil, false, err
			}
			tenantId := int32(tenantDetails.Id)
			desiredRouteTarget.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}
	}

	// create route target since it doesn't exist
	if len(responseRouteTargetList.Results) == 0 {
		resp, err := c.createRouteTarget(ctx, desiredRouteTarget)
		return resp, false, err
	}

	routeTargetToUpdate := &responseRouteTargetList.Results[0]

	if !routeTargetToUpdate.LastUpdated.IsSet() {
		return nil, false, fmt.Errorf("last updated field is not set in Netbox for route target %s", routeTarget.Name)
	}

	// if the desired route target has a restoration hash
	// check that the route target to update has the same restoration hash
	restorationHashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if routeTarget.Metadata != nil {
		if restorationHash, ok := routeTarget.Metadata.Custom[restorationHashKey]; ok {
			if routeTargetToUpdate.CustomFields != nil && routeTargetToUpdate.CustomFields[restorationHashKey] == restorationHash {
				if IsUpToDate(ctx, *routeTargetToUpdate.LastUpdated.Get(), routeTargetV1.Status.LastUpdated, routeTargetV1.Status.Conditions, routeTargetV1.Generation) {
					return nil, true, nil
				}

				//update route target since it does exist and the restoration hash matches
				resp, err := c.updateRouteTarget(ctx, routeTargetToUpdate.Id, desiredRouteTarget)
				if err != nil {
					return nil, false, err
				}
				return resp, false, nil
			}
			return nil, false, fmt.Errorf("%w, assigned route target %s", ErrRestorationHashMismatch, routeTarget.Name)
		}
	}

	if IsUpToDate(ctx, *routeTargetToUpdate.LastUpdated.Get(), routeTargetV1.Status.LastUpdated, routeTargetV1.Status.Conditions, routeTargetV1.Generation) {
		return nil, true, nil
	}

	//update route target since it does exist
	resp, err = c.updateRouteTarget(ctx, routeTargetToUpdate.Id, desiredRouteTarget)
	if err != nil {
		return nil, false, err
	}
	return resp, false, nil
}

func (c *NetboxCompositeClient) getRouteTarget(ctx context.Context, name string) (resp *v4client.PaginatedRouteTargetList, err error) {
	req := c.clientV4.IpamAPI.IpamRouteTargetsList(ctx).Name([]string{name})
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "fetch route target details")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) createRouteTarget(ctx context.Context, routeTarget *v4client.RouteTargetRequest) (resp *v4client.RouteTarget, err error) {
	req := c.clientV4.IpamAPI.IpamRouteTargetsCreate(ctx).RouteTargetRequest(*routeTarget)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "reserve route target")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) updateRouteTarget(ctx context.Context, routeTargetId int32, routeTarget *v4client.RouteTargetRequest) (resp *v4client.RouteTarget, err error) {
	req := c.clientV4.IpamAPI.IpamRouteTargetsUpdate(ctx, routeTargetId).RouteTargetRequest(*routeTarget)
	resp, httpResp, execErr := req.Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusOK, "update route target")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return nil, handleErr
	}

	return resp, nil
}

func (c *NetboxCompositeClient) DeleteRouteTarget(ctx context.Context, routeTargetId int64) (err error) {
	if routeTargetId > math.MaxInt32 {
		return fmt.Errorf("deletion of route targets with id's larger than %d is not supported", math.MaxInt32)
	}

	req := c.clientV4.IpamAPI.IpamRouteTargetsDestroy(ctx, int32(routeTargetId))
	httpResp, execErr := req.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil
	}

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusNoContent, "delete route target from netbox")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		return handleErr
	}

	return nil
}
