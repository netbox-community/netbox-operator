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
	"fmt"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

func (r *NetboxClient) RestoreExistingVlanByHash(hash string) (*models.Vlan, error) {
	customVlanSearch := newQueryFilterOperation(nil, []CustomFieldEntry{
		{
			key:   config.GetOperatorConfig().NetboxRestorationHashFieldName,
			value: hash,
		},
	})
	list, err := r.Ipam.IpamVlansList(ipam.NewIpamVlansListParams(), nil, customVlanSearch)
	if err != nil {
		return nil, err
	}

	var filteredResults []*netboxModels.VLAN
	for _, v := range list.Payload.Results {
		if v != nil && v.CustomFields != nil {
			cfMap, ok := v.CustomFields.(map[string]interface{})
			if !ok {
				continue
			}
			if val, ok := cfMap[config.GetOperatorConfig().NetboxRestorationHashFieldName]; ok {
				if val == hash {
					filteredResults = append(filteredResults, v)
				}
			}
		}
	}

	if len(filteredResults) == 0 {
		return nil, nil
	}

	if len(filteredResults) > 1 {
		return nil, fmt.Errorf("incorrect number of restoration results for VLAN after filtering, number of results: %v", len(filteredResults))
	}

	res := filteredResults[0]
	vid := 0
	if res.Vid != nil {
		vid = int(*res.Vid)
	}
	return &models.Vlan{
		VlanId: vid,
		Name:   *res.Name,
	}, nil
}

// GetAvailableVlanByClaim searches for an available VID in NetBox matching VLANClaim requirements
func (r *NetboxClient) GetAvailableVlanByClaim(vlanClaim *models.VLANClaim) (*models.Vlan, error) {
	vlanGroup, err := r.GetVlanGroupDetails(vlanClaim.VlanGroup)
	if err != nil {
		return nil, err
	}

	vlanGroupIdStr := fmt.Sprintf("%d", vlanGroup.Id)
	params := ipam.NewIpamVlansListParams().WithGroupID(&vlanGroupIdStr)

	// We might need to handle pagination if there are many VLANs in a group
	// For now, let's assume standard limit is enough or we loop
	existingVlans, err := r.Ipam.IpamVlansList(params, nil)
	if err != nil {
		return nil, err
	}

	usedVids := make(map[int]bool)
	for _, v := range existingVlans.Payload.Results {
		if v.Vid != nil {
			usedVids[int(*v.Vid)] = true
		}
	}

	// Find first available VID between 1 and 4094
	// TODO: Respect VlanGroup min/max if we can fetch them
	allocatedVid := -1
	for i := 1; i <= 4094; i++ {
		if !usedVids[i] {
			allocatedVid = i
			break
		}
	}

	if allocatedVid == -1 {
		return nil, fmt.Errorf("no available VIDs found in VlanGroup %s", vlanClaim.VlanGroup)
	}

	return &models.Vlan{
		VlanId: allocatedVid,
	}, nil
}
