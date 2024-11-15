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
	"errors"
	"fmt"
	"net"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (r *NetboxClient) RestoreExistingIpRangeByHash(customFieldName string, hash string) (*models.IpRange, error) {
	customIpSearch := newCustomFieldStringFilterOperation(customFieldName, hash)
	list, err := r.Ipam.IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, customIpSearch)
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
	if res.StartAddress == nil {
		return nil, errors.New("iprange in netbox is nil")
	}

	return &models.IpRange{
		StartAddress: *res.StartAddress,
		EndAddress:   *res.EndAddress,
	}, nil
}

// GetAvailableIpRangeByClaim searches an available IpRange in Netbox matching IpRangeClaim requirements
func (r *NetboxClient) GetAvailableIpRangeByClaim(ipRangeClaim *models.IpRangeClaim) (*models.IpRange, error) {
	_, err := r.GetTenantDetails(ipRangeClaim.Metadata.Tenant)
	if err != nil {
		return nil, err
	}

	responseParentPrefix, err := r.GetPrefix(&models.Prefix{
		Prefix:   ipRangeClaim.ParentPrefix,
		Metadata: ipRangeClaim.Metadata,
	})
	if err != nil {
		return nil, err
	}
	if len(responseParentPrefix.Payload.Results) == 0 {
		return nil, utils.NetboxNotFoundError("parent prefix")
	}

	parentPrefixId := responseParentPrefix.Payload.Results[0].ID
	responseAvailableIPs, err := r.GetAvailableIpRangesByParentPrefix(parentPrefixId)
	if err != nil {
		return nil, err
	}

	startAddress, endAddress, err := searchAvailableIpRange(responseAvailableIPs, ipRangeClaim.Size)
	if err != nil {
		return nil, err
	}

	var ipMask string
	if responseAvailableIPs.Payload[0].Family == int64(IPv4Family) {
		ipMask = ipMaskIPv4
	} else if responseAvailableIPs.Payload[0].Family == int64(IPv6Family) {
		ipMask = ipMaskIPv6
	} else {
		return nil, errors.New("available ip has unknown IP family")
	}

	startAddress, err = r.SetIpRangeMask(startAddress, ipMask)
	if err != nil {
		return nil, err
	}

	endAddress, err = r.SetIpRangeMask(endAddress, ipMask)
	if err != nil {
		return nil, err
	}

	return &models.IpRange{
		StartAddress: startAddress,
		EndAddress:   endAddress,
	}, nil
}

func searchAvailableIpRange(availableIps *ipam.IpamPrefixesAvailableIpsListOK, requiredSize int) (string, string, error) {
	// this function receives a list of available IPs it chan have IPv4 or IPv6 IPs
	// it will search for the first available range of IPs with the required size
	// it will return the start and end IP of the range
	var startAddress, endAddress string
	consecutiveCount := 0

	for i := 1; i < len(availableIps.Payload); i++ {
		currentIp, _, err := net.ParseCIDR(availableIps.Payload[i].Address)
		if err != nil {
			return "", "", err
		}
		var previousIP net.IP
		if i > 0 {
			previousIP, _, err = net.ParseCIDR(availableIps.Payload[i-1].Address)
			if err != nil {
				return "", "", err
			}
		}
		if i == 0 || bytes.Compare(currentIp, previousIP) == 1 {
			consecutiveCount++
			if consecutiveCount == requiredSize {
				startAddress = availableIps.Payload[i-requiredSize+1].Address
				endAddress = availableIps.Payload[i].Address
				break
			}
		} else {
			consecutiveCount = 1
		}
	}

	if consecutiveCount < requiredSize {
		return "", "", errors.New("not enough consecutive IPs available")
	}

	return startAddress, endAddress, nil
}

func (r *NetboxClient) GetAvailableIpRangesByParentPrefix(parentPrefixId int64) (*ipam.IpamPrefixesAvailableIpsListOK, error) {
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

func (r *NetboxClient) SetIpRangeMask(ip string, ipMask string) (string, error) {
	ipRange, _, err := net.ParseCIDR(ip)
	if err != nil {
		return "", err
	}
	ipRangeWithNewMask := ipRange.String() + ipMask
	return ipRangeWithNewMask, nil
}
