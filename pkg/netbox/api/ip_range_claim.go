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
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (r *NetboxClient) RestoreExistingIpRangeByHash(hash string) (*models.IpRange, error) {
	customIpRangeSearch := newQueryFilterOperation(nil, []CustomFieldEntry{
		{
			key:   config.GetOperatorConfig().NetboxRestorationHashFieldName,
			value: hash,
		},
	})
	list, err := r.Ipam.IpamIPRangesList(ipam.NewIpamIPRangesListParams(), nil, customIpRangeSearch)
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
	if res.StartAddress == nil || res.EndAddress == nil {
		return nil, errors.New("invalid IP range")
	}

	return &models.IpRange{
		StartAddress: *res.StartAddress,
		EndAddress:   *res.EndAddress,
		Id:           res.ID,
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
	responseAvailableIPs, err := r.GetAvailableIpAddressesByParentPrefix(parentPrefixId)
	if err != nil {
		return nil, err
	}

	startAddress, endAddress, err := searchAvailableIpRange(responseAvailableIPs, ipRangeClaim.Size)
	if err != nil {
		return nil, err
	}

	startAddress, err = r.SetIpAddressMask(startAddress, responseAvailableIPs.Payload[0].Family)
	if err != nil {
		return nil, err
	}

	endAddress, err = r.SetIpAddressMask(endAddress, responseAvailableIPs.Payload[0].Family)
	if err != nil {
		return nil, err
	}

	return &models.IpRange{
		StartAddress: startAddress,
		EndAddress:   endAddress,
	}, nil
}

// GetAvailableIpsByIpRange returns all available Ips in Netbox matching IpRangeClaim requirements
func (r *NetboxClient) GetAvailableIpAddressesByIpRange(ipRangeId int64) (*ipam.IpamIPRangesAvailableIpsListOK, error) {
	requestAvailableIPs := ipam.NewIpamIPRangesAvailableIpsListParams().WithID(ipRangeId)
	responseAvailableIPs, err := r.Ipam.IpamIPRangesAvailableIpsList(requestAvailableIPs, nil)
	if err != nil {
		return nil, err
	}
	return responseAvailableIPs, nil
}

func searchAvailableIpRange(availableIps *ipam.IpamPrefixesAvailableIpsListOK, requiredSize int) (string, string, error) {
	// this function receives a list of available IPs it chan have IPv4 or IPv6 IPs
	// it will search for the first available range of IPs with the required size
	// it will return the start and end IP of the range
	var startAddress, endAddress string
	consecutiveCount := 0

	for i := 0; i < len(availableIps.Payload); i++ {
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
		if i == 0 || areConsecutiveIPs(previousIP, currentIp) {
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

// Check if two IP addresses (IPv4 or IPv6) are consecutive
func areConsecutiveIPs(ip1, ip2 net.IP) bool {
	if ip1.To4() != nil && ip2.To4() != nil {
		return areConsecutiveIPv4(ip1, ip2)
	}
	if ip1.To16() != nil && ip2.To16() != nil {
		return areConsecutiveIPv6(ip1, ip2)
	}
	return false
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

func ipToUint128(ip net.IP) (uint64, uint64) {
	ip = ip.To16()
	return binary.BigEndian.Uint64(ip[:8]), binary.BigEndian.Uint64(ip[8:])
}

func areConsecutiveIPv4(ip1, ip2 net.IP) bool {
	return ipToUint32(ip2)-ipToUint32(ip1) == 1
}

func areConsecutiveIPv6(ip1, ip2 net.IP) bool {
	ip1High, ip1Low := ipToUint128(ip1)
	ip2High, ip2Low := ipToUint128(ip2)
	if ip1High == ip2High {
		return ip2Low-ip1Low == 1
	}
	return ip2High-ip1High == 1 && ip2Low == 0 && ip1Low == ^uint64(0)
}
