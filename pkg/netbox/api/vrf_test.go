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
	"testing"

	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestVrf_GetVrfDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	vrf := "myVrf"

	vrfListRequestInput := ipam.NewIpamVrfsListParams().WithName(&vrf)

	vrfOutputId := int64(1)
	vrfListOutput := &ipam.IpamVrfsListOK{
		Payload: &ipam.IpamVrfsListOKBody{
			Results: []*netboxModels.VRF{
				{
					ID:   vrfOutputId,
					Name: &vrf,
				},
			},
		},
	}

	mockIpam.EXPECT().IpamVrfsList(vrfListRequestInput, nil).Return(vrfListOutput, nil)
	netboxClient := &NetboxClient{Ipam: mockIpam}

	actual, err := netboxClient.GetVrfDetails(vrf)
	assert.NoError(t, err)
	assert.Equal(t, vrf, actual.Name)
	assert.Equal(t, vrfOutputId, actual.Id)
}

func TestVrf_GetEmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	vrf := "myVrf"

	vrfListRequestInput := ipam.NewIpamVrfsListParams().WithName(&vrf)

	emptyListVrfOutput := &ipam.IpamVrfsListOK{
		Payload: &ipam.IpamVrfsListOKBody{
			Results: []*netboxModels.VRF{},
		},
	}

	mockIpam.EXPECT().IpamVrfsList(vrfListRequestInput, nil).Return(emptyListVrfOutput, nil)
	netboxClient := &NetboxClient{Ipam: mockIpam}

	actual, err := netboxClient.GetVrfDetails(vrf)
	assert.Nil(t, actual)
	assert.EqualError(t, err, "failed to fetch vrf 'myVrf': not found")
}

func TestVrf_GetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	vrf := "myVrf"

	vrfListRequestInput := ipam.NewIpamVrfsListParams().WithName(&vrf)

	expectedErr := "error getting vrfs list"

	mockIpam.EXPECT().IpamVrfsList(vrfListRequestInput, nil).Return(nil, errors.New(expectedErr))
	netboxClient := &NetboxClient{Ipam: mockIpam}

	actual, err := netboxClient.GetVrfDetails(vrf)
	assert.Nil(t, actual)
	assert.EqualError(t, err, "failed to fetch VRF details: "+expectedErr)
}
