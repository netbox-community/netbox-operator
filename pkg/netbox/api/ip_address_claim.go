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
	"errors"
	"fmt"
	"net"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

const (
	ipMask = "/32"
)

func (r *NetboxClient) RestoreExistingIpByHash(customFieldName string, hash string) (*models.IPAddress, error) {
	customIpSearch := newCustomFieldStringFilterOperation(customFieldName, hash)
	list, err := r.Ipam.IpamIPAddressesList(ipam.NewIpamIPAddressesListParams(), nil, customIpSearch)
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
func (r *NetboxClient) GetAvailableIpAddressByClaim(ipAddressClaim *models.IPAddressClaim) (*models.IPAddress, error) {

	responseParentPrefix, err := r.GetPrefix(&models.Prefix{
		Prefix:   ipAddressClaim.ParentPrefix,
		Metadata: ipAddressClaim.Metadata,
	})
	if err != nil {
		return nil, err
	}
	if len(responseParentPrefix.Payload.Results) == 0 {
		return nil, errors.New("parent prefix not found")
	}

	parentPrefixId := responseParentPrefix.Payload.Results[0].ID
	responseAvailableIPs, err := r.GetAvailableIpAddressesByParentPrefix(parentPrefixId)
	if err != nil {
		return nil, err
	}

	ipAddress, err := r.SetIpAddressMask(responseAvailableIPs.Payload[0].Address, ipMask)
	if err != nil {
		return nil, err
	}

	return &models.IPAddress{
		IpAddress: ipAddress,
	}, nil
}

func (r *NetboxClient) GetAvailableIpAddressesByParentPrefix(parentPrefixId int64) (*ipam.IpamPrefixesAvailableIpsListOK, error) {
	requestAvailableIPs := ipam.NewIpamPrefixesAvailableIpsListParams().WithID(parentPrefixId)
	responseAvailableIPs, err := r.Ipam.IpamPrefixesAvailableIpsList(requestAvailableIPs, nil)
	if err != nil {
		return nil, err
	}
	if len(responseAvailableIPs.Payload) == 0 {
		return nil, errors.New("parent prefix exhausted")
	}
	return responseAvailableIPs, nil
}

func (r *NetboxClient) SetIpAddressMask(ip string, ipMask string) (string, error) {
	ipAddress, _, err := net.ParseCIDR(ip)
	if err != nil {
		return "", err
	}
	ipAddressWithNewMask := ipAddress.String() + ipMask
	return ipAddressWithNewMask, nil
}
