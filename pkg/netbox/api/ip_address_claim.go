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

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

type IPFamily int64

const (
	IPv4Family IPFamily = iota + 4
	_                   // Skip 5
	IPv6Family
)

const (
	ipMaskIPv4 = "/32"
	ipMaskIPv6 = "/128"
)

func (c *NetboxCompositeClient) RestoreExistingIpByHash(hash string) (*models.IPAddress, error) {
	customIpSearch := newQueryFilterOperation(nil, []CustomFieldEntry{
		{
			key:   config.GetOperatorConfig().NetboxRestorationHashFieldName,
			value: hash,
		},
	})
	list, err := c.clientV3.Ipam.IpamIPAddressesList(ipam.NewIpamIPAddressesListParams(), nil, customIpSearch)
	if err != nil {
		return nil, err
	}

	// TODO: find a better way?
	if list.Payload.Count != nil && *list.Payload.Count == 0 {
		return nil, nil
	}

	// We should not have more than 1 result...
	if len(list.Payload.Results) != 1 {
		return nil, fmt.Errorf("incorrect number of restoration results, number of results: %v", len(list.Payload.Results))
	}
	res := list.Payload.Results[0]
	if res.Address == nil {
		return nil, errors.New("ipaddress in netbox is nil")
	}

	return &models.IPAddress{
		IpAddress: *res.Address,
	}, nil
}

// GetAvailableIpAddressByClaim searches an available IpAddress in Netbox matching IpAddressClaim requirements
func (c *NetboxCompositeClient) GetAvailableIpAddressByClaim(ctx context.Context, ipAddressClaim *models.IPAddressClaim) (*models.IPAddress, error) {
	// fail early if tenant requested in the spec does not exists
	_, err := c.getTenantDetails(ipAddressClaim.Metadata.Tenant)
	if err != nil {
		return nil, err
	}

	responseParentPrefix, err := c.getPrefix(
		ctx,
		&models.Prefix{
			Prefix:   ipAddressClaim.ParentPrefix,
			Metadata: ipAddressClaim.Metadata,
		})
	if err != nil {
		return nil, err
	}
	if len(responseParentPrefix.Results) == 0 {
		return nil, utils.NetboxNotFoundError("parent prefix")
	}

	parentPrefixId := responseParentPrefix.Results[0].Id
	responseAvailableIPs, err := c.GetAvailableIpAddressesByParentPrefix(parentPrefixId)
	if err != nil {
		return nil, err
	}

	ipAddress, err := SetIpAddressMask(responseAvailableIPs.Payload[0].Address, responseAvailableIPs.Payload[0].Family)
	if err != nil {
		return nil, err
	}

	return &models.IPAddress{
		IpAddress: ipAddress,
	}, nil
}

func (c *NetboxCompositeClient) GetAvailableIpAddressesByParentPrefix(parentPrefixId int32) (*ipam.IpamPrefixesAvailableIpsListOK, error) {
	requestAvailableIPs := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(int64(parentPrefixId))
	responseAvailableIPs, err := c.clientV3.Ipam.IpamPrefixesAvailableIpsList(requestAvailableIPs, nil)
	if err != nil {
		return nil, err
	}
	if len(responseAvailableIPs.Payload) == 0 {
		return nil, ErrParentPrefixExhausted
	}
	return responseAvailableIPs, nil
}
