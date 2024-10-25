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
	"testing"

	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"

	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTenancy_GetTenantDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPrefix := mock_interfaces.NewMockTenancyInterface(ctrl)

	tenant := "myTenant"

	tenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&tenant)

	tenantOutputId := int64(1)
	tenantOutputSlug := "mytenant"
	tenantListOutput := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantOutputId,
					Name: &tenant,
					Slug: &tenantOutputSlug,
				},
			},
		},
	}

	mockPrefix.EXPECT().TenancyTenantsList(tenantListRequestInput, nil).Return(tenantListOutput, nil)
	netboxClient := &NetboxClient{Tenancy: mockPrefix}

	actual, err := netboxClient.GetTenantDetails(tenant)
	assert.NoError(t, err)
	assert.Equal(t, tenant, actual.Name)
	assert.Equal(t, tenantOutputId, actual.Id)
	assert.Equal(t, tenantOutputSlug, actual.Slug)
}

func TestTenancy_GetWrongTenantDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPrefix := mock_interfaces.NewMockTenancyInterface(ctrl)

	wrongTenant := "wrongTenant"

	wrongTenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&wrongTenant)

	emptyListTenantOutput := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{},
		},
	}

	mockPrefix.EXPECT().TenancyTenantsList(wrongTenantListRequestInput, nil).Return(emptyListTenantOutput, nil)
	netboxClient := &NetboxClient{Tenancy: mockPrefix}

	actual, err := netboxClient.GetTenantDetails(wrongTenant)
	assert.Nil(t, actual)
	assert.EqualError(t, err, "failed to fetch tenant 'wrongTenant': not found")
}
