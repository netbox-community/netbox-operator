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
	"net/http"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

var ErrAsnRangeExhausted = errors.New("ASN range exhausted")

func (c *NetboxCompositeClient) RestoreExistingAsnByHash(ctx context.Context, hash string) (*models.ASN, error) {
	customAsnSearch := newAsnListQuery(nil, []CustomFieldEntry{
		{
			key:   config.GetOperatorConfig().NetboxRestorationHashFieldName,
			value: hash,
		},
	})
	list, err := c.clientV3.Ipam.IpamAsnsList(ipam.NewIpamAsnsListParams().WithContext(ctx), nil, customAsnSearch)
	if err != nil {
		return nil, err
	}

	if len(list.Payload.Results) == 0 {
		return nil, nil
	}

	// We should not have more than 1 result...
	if len(list.Payload.Results) != 1 {
		return nil, fmt.Errorf("incorrect number of restoration results, number of results: %v", len(list.Payload.Results))
	}
	res := list.Payload.Results[0]
	if res.Asn == nil {
		return nil, errors.New("asn in netbox is nil")
	}

	return &models.ASN{
		Asn: *res.Asn,
		Id:  res.ID,
	}, nil
}

// ReserveAvailableAsnByClaim allocates an ASN from the specified ASN Range. Unlike the
// IP address and prefix equivalents this already creates the object in NetBox.
func (c *NetboxCompositeClient) ReserveAvailableAsnByClaim(ctx context.Context, asnClaim *models.ASNClaim) (asn *models.ASN, err error) {
	// Claim an available ASN from the range by POSTing to available-asns. The custom
	// fields (in particular the restoration hash) and the tenant have to be set as part of
	// this request: NetBox allocates and persists the ASN in a single transaction, so
	// setting them afterwards would leave a window in which the ASN is unidentifiable.
	asnRequest := v4client.ASNRequest{
		Asn:          0, // The ASN value will be assigned by NetBox
		CustomFields: make(map[string]interface{}),
	}

	if asnClaim.Metadata != nil {
		for k, v := range asnClaim.Metadata.Custom {
			asnRequest.CustomFields[k] = v
		}

		desc := TruncateDescription(asnClaim.Metadata.Description)
		asnRequest.Description = &desc

		// fail early if tenant requested in the spec does not exist
		if asnClaim.Metadata.Tenant != "" {
			tenantDetails, err := c.getTenantDetails(asnClaim.Metadata.Tenant)
			if err != nil {
				return nil, err
			}
			tenantId := int32(tenantDetails.Id)
			asnRequest.SetTenant(v4client.Int32AsASNRangeRequestTenant(&tenantId))
		}

		// Fail before anything is created in NetBox if the RIR requested in the spec does not
		// exist. The RIR is deliberately not part of the request: the available-asns endpoint
		// overwrites it with the RIR of the parent ASN Range. An override is applied afterwards
		// by the Asn controller, and would fail there, leaving an orphaned ASN behind if the
		// claim were deleted before the Asn resource was created.
		if asnClaim.Metadata.Rir != "" {
			if _, err := c.getRirDetailsByName(ctx, asnClaim.Metadata.Rir); err != nil {
				return nil, err
			}
		}
	}

	// Look up the ASN Range by name
	asnRangeId, err := c.getAsnRangeIdByName(ctx, asnClaim.ParentAsnRange)
	if err != nil {
		return nil, err
	}

	asns, httpResp, execErr := c.clientV4.IpamAPI.IpamAsnRangesAvailableAsnsCreate(ctx, asnRangeId).
		ASNRequest([]v4client.ASNRequest{asnRequest}).Execute()

	closeFunc, handleErr := handleHTTPResponse(httpResp, execErr, http.StatusCreated, "claim available ASN")
	if closeFunc != nil {
		defer func() { err = errors.Join(err, closeFunc()) }()
	}
	if handleErr != nil {
		// NetBox answers with 409 Conflict when the range has no free ASN left. Every
		// other failure is a genuine error and must not be reported as exhaustion.
		if httpResp != nil && httpResp.StatusCode == http.StatusConflict {
			return nil, fmt.Errorf("%w: %w", ErrAsnRangeExhausted, handleErr)
		}
		return nil, handleErr
	}

	if len(asns) == 0 {
		return nil, ErrAsnRangeExhausted
	}

	return &models.ASN{
		Asn: asns[0].Asn,
		Id:  int64(asns[0].Id),
	}, nil
}

func (c *NetboxCompositeClient) getAsnRangeIdByName(ctx context.Context, name string) (int32, error) {
	result, err := c.listAsnRanges(ctx, []string{name})
	if err != nil {
		return 0, err
	}

	if len(result.Results) == 0 {
		return 0, utils.NetboxNotFoundError("ASN Range")
	}

	return result.Results[0].Id, nil
}

// GetAsnRangeRirName returns the name of the RIR the given ASN Range belongs to.
func (c *NetboxCompositeClient) GetAsnRangeRirName(ctx context.Context, name string) (string, error) {
	result, err := c.listAsnRanges(ctx, []string{name})
	if err != nil {
		return "", err
	}

	if len(result.Results) == 0 {
		return "", utils.NetboxNotFoundError("ASN Range")
	}

	return result.Results[0].Rir.Name, nil
}
