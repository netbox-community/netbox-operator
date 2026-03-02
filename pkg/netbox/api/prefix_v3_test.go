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
	"testing"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPrefix_CreatePrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	//prefix mock input
	prefix := "10.112.140.0/24"
	siteId := int64(1)
	tenantId := int64(1)
	comment := "a comment"
	description := "very useful prefix"
	prefixToCreate := &netboxModels.WritablePrefix{
		Comments:    comment,
		Description: description,
		Prefix:      &prefix,
		Site:        &siteId,
		Tenant:      &tenantId,
	}
	createPrefixInput := ipam.
		NewIpamPrefixesCreateParams().
		WithDefaults().
		WithData(prefixToCreate)

	//prefix mock output
	createPrefixOutput := &ipam.IpamPrefixesCreateCreated{
		Payload: &netboxModels.Prefix{
			ID:          int64(1),
			Comments:    comment,
			Description: description,
			Display:     prefix,
			Prefix:      &prefix,
			Site: &netboxModels.NestedSite{
				ID: siteId,
			},
			Tenant: &netboxModels.NestedTenant{
				ID: tenantId,
			},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesCreate(createPrefixInput, nil).Return(createPrefixOutput, nil)

	clientV3 := &NetboxClientV3{
		Ipam: mockPrefixIpam,
	}

	actual, err := clientV3.createPrefixV3(prefixToCreate)
	assert.Nil(t, err)
	assert.Greater(t, actual.Id, int32(0))
	assert.Equal(t, prefix, actual.Prefix)
	assert.Equal(t, description, *actual.Description)
	assert.Equal(t, prefix, actual.Prefix)
}

func TestPrefix_UpdatePrefix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPrefixIpam := mock_interfaces.NewMockIpamInterface(ctrl)

	//prefix mock input
	prefixId := int64(1)
	prefix := "10.112.140.0/24"
	siteId := int64(1)
	tenantId := int64(1)
	comment := "a comment"
	updatedDescription := "updated"
	prefixToUpdate := &netboxModels.WritablePrefix{
		Comments:    comment,
		Description: updatedDescription,
		Prefix:      &prefix,
		Site:        &siteId,
		Tenant:      &tenantId,
	}
	updatePrefixInput := ipam.
		NewIpamPrefixesUpdateParams().
		WithDefaults().
		WithData(prefixToUpdate).
		WithID(prefixId)

	//prefix mock output
	updatePrefixOutput := &ipam.IpamPrefixesUpdateOK{
		Payload: &netboxModels.Prefix{
			ID:          int64(1),
			Comments:    comment,
			Description: updatedDescription,
			Display:     prefix,
			Prefix:      &prefix,
			Site: &netboxModels.NestedSite{
				ID: siteId,
			},
			Tenant: &netboxModels.NestedTenant{
				ID: tenantId,
			},
		},
	}

	mockPrefixIpam.EXPECT().IpamPrefixesUpdate(updatePrefixInput, nil).Return(updatePrefixOutput, nil)

	clientV3 := &NetboxClientV3{
		Ipam: mockPrefixIpam,
	}

	actual, err := clientV3.updatePrefixV3(prefixId, prefixToUpdate)
	assert.Nil(t, err)
	assert.Greater(t, actual.Id, int32(0))
	assert.Equal(t, prefix, actual.Prefix)
	assert.Equal(t, updatedDescription, *actual.Description)
}
