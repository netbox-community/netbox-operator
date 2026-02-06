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
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

// to ensure compatibility with older NetBox versions the CreatePrefix and UpdatePrefix
// functions for the v3 client are still required

func (r *NetboxClient) CreatePrefixV3(prefix *netboxModels.WritablePrefix) (*netboxModels.Prefix, error) {
	requestCreatePrefix := ipam.
		NewIpamPrefixesCreateParams().
		WithDefaults().
		WithData(prefix)
	responseCreatePrefix, err := r.Ipam.
		IpamPrefixesCreate(requestCreatePrefix, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to create Prefix", err)
	}
	return responseCreatePrefix.Payload, nil
}

func (r *NetboxClient) UpdatePrefixV3(prefixId int64, prefix *netboxModels.WritablePrefix) (*netboxModels.Prefix, error) {
	requestUpdatePrefix := ipam.NewIpamPrefixesUpdateParams().
		WithDefaults().
		WithData(prefix).
		WithID(prefixId)
	responseUpdatePrefix, err := r.Ipam.IpamPrefixesUpdate(requestUpdatePrefix, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to update Prefix", err)
	}
	return responseUpdatePrefix.Payload, nil
}
