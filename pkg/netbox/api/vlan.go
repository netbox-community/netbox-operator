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
	"net/http"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (r *NetboxClient) GetVlan(vlan *models.Vlan) (*ipam.IpamVlansListOK, error) {
	vidStr := fmt.Sprintf("%d", vlan.VlanId)
	requestVlan := ipam.NewIpamVlansListParams().
		WithVid(&vidStr)

	if vlan.Metadata != nil {
		if vlan.Metadata.Site != "" {
			siteDetails, err := r.GetSiteDetails(vlan.Metadata.Site)
			if err != nil {
				return nil, err
			}
			siteIdStr := fmt.Sprintf("%d", siteDetails.Id)
			requestVlan = requestVlan.WithSiteID(&siteIdStr)
		}
	}

	responseVlan, err := r.Ipam.IpamVlansList(requestVlan, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to fetch Vlan details", err)
	}

	return responseVlan, nil
}

func (r *NetboxClient) CreateVlan(vlan *netboxModels.WritableVLAN) (*netboxModels.VLAN, error) {
	requestCreateVlan := ipam.NewIpamVlansCreateParams().
		WithDefaults().
		WithData(vlan)
	responseCreateVlan, err := r.Ipam.IpamVlansCreate(requestCreateVlan, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to create Vlan", err)
	}
	return responseCreateVlan.Payload, nil
}

func (r *NetboxClient) UpdateVlan(vlanId int64, vlan *netboxModels.WritableVLAN) (*netboxModels.VLAN, error) {
	requestUpdateVlan := ipam.NewIpamVlansUpdateParams().
		WithDefaults().
		WithData(vlan).
		WithID(vlanId)
	responseUpdateVlan, err := r.Ipam.IpamVlansUpdate(requestUpdateVlan, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to update Vlan", err)
	}
	return responseUpdateVlan.Payload, nil
}

func (r *NetboxClient) DeleteVlan(vlanId int64) error {
	requestDeleteVlan := ipam.NewIpamVlansDeleteParams().WithID(vlanId)
	_, err := r.Ipam.IpamVlansDelete(requestDeleteVlan, nil)
	if err != nil {
		switch typedErr := err.(type) {
		case *ipam.IpamVlansDeleteDefault:
			if typedErr.IsCode(http.StatusNotFound) {
				return nil
			}
			return utils.NetboxError("Failed to delete vlan from Netbox", err)
		default:
			return utils.NetboxError("Failed to delete vlan from Netbox", err)
		}
	}
	return nil
}

func (r *NetboxClient) GetVlanGroupDetails(vlanGroupName string) (*models.VlanGroup, error) {
	requestVlanGroup := ipam.NewIpamVlanGroupsListParams().WithName(&vlanGroupName)
	responseVlanGroup, err := r.Ipam.IpamVlanGroupsList(requestVlanGroup, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to fetch VlanGroup details", err)
	}

	if len(responseVlanGroup.Payload.Results) == 0 {
		return nil, fmt.Errorf("vlangroup %s not found in Netbox", vlanGroupName)
	}

	return &models.VlanGroup{
		Id:   responseVlanGroup.Payload.Results[0].ID,
		Name: *responseVlanGroup.Payload.Results[0].Name,
		Slug: *responseVlanGroup.Payload.Results[0].Slug,
	}, nil
}
