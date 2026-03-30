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
	"errors"
	"testing"

	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
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

	actual, skipsUpdate, err := clientV3.createPrefixV3(prefixToCreate)
	assert.Nil(t, err)
	assert.False(t, skipsUpdate)
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

	actual, skipsUpdate, err := clientV3.updatePrefixV3(prefixId, prefixToUpdate)
	assert.Nil(t, err)
	assert.False(t, skipsUpdate)
	assert.Greater(t, actual.Id, int32(0))
	assert.Equal(t, prefix, actual.Prefix)
	assert.Equal(t, updatedDescription, *actual.Description)
}

func TestBuildWritablePrefixRequestV3_NoTenantNoSite(t *testing.T) {
	compositeClient := &NetboxCompositeClient{}

	prefix := "10.0.0.0/24"
	comments := "my comment"
	description := "my description"
	customFields := map[string]string{"key1": "val1"}

	result, err := compositeClient.buildWritablePrefixRequestV3(&models.Prefix{
		Prefix: prefix,
		Metadata: &models.NetboxMetadata{
			Comments:    comments,
			Description: description,
			Custom:      customFields,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, &prefix, result.Prefix)
	assert.Equal(t, comments+warningComment, result.Comments)
	assert.Equal(t, TruncateDescription(description), result.Description)
	assert.Equal(t, customFields, result.CustomFields)
	assert.Equal(t, "active", result.Status)
	assert.Nil(t, result.Tenant)
	assert.Nil(t, result.Site)
}

func TestBuildWritablePrefixRequestV3_WithTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantName := "Tenant1"
	tenantId := int64(7)
	tenantSlug := "tenant1"

	tenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
	tenantListRequestOutput := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantSlug,
				},
			},
		},
	}

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockTenancy.EXPECT().TenancyTenantsList(tenantListRequestInput, nil).Return(tenantListRequestOutput, nil)

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Tenancy: mockTenancy},
	}

	prefix := "10.0.0.0/24"

	result, err := compositeClient.buildWritablePrefixRequestV3(&models.Prefix{
		Prefix: prefix,
		Metadata: &models.NetboxMetadata{
			Tenant: tenantName,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tenantId, *result.Tenant)
	assert.Nil(t, result.Site)
}

func TestBuildWritablePrefixRequestV3_WithSite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	siteName := "Site1"
	siteId := int64(3)
	siteSlug := "site1"

	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&siteName)
	siteListRequestOutput := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{
				{
					ID:   siteId,
					Name: &siteName,
					Slug: &siteSlug,
				},
			},
		},
	}

	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
	mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(siteListRequestOutput, nil)

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Dcim: mockDcim},
	}

	prefix := "10.0.0.0/24"

	result, err := compositeClient.buildWritablePrefixRequestV3(&models.Prefix{
		Prefix: prefix,
		Metadata: &models.NetboxMetadata{
			Site: siteName,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, siteId, *result.Site)
	assert.Nil(t, result.Tenant)
}

func TestBuildWritablePrefixRequestV3_WithTenantAndSite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantName := "Tenant1"
	tenantId := int64(7)
	tenantSlug := "tenant1"
	siteName := "Site1"
	siteId := int64(3)
	siteSlug := "site1"

	tenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)
	tenantListRequestOutput := &tenancy.TenancyTenantsListOK{
		Payload: &tenancy.TenancyTenantsListOKBody{
			Results: []*netboxModels.Tenant{
				{
					ID:   tenantId,
					Name: &tenantName,
					Slug: &tenantSlug,
				},
			},
		},
	}
	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&siteName)
	siteListRequestOutput := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{
				{
					ID:   siteId,
					Name: &siteName,
					Slug: &siteSlug,
				},
			},
		},
	}

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockTenancy.EXPECT().TenancyTenantsList(tenantListRequestInput, nil).Return(tenantListRequestOutput, nil)
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
	mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(siteListRequestOutput, nil)

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{
			Tenancy: mockTenancy,
			Dcim:    mockDcim,
		},
	}

	prefix := "10.0.0.0/24"

	result, err := compositeClient.buildWritablePrefixRequestV3(&models.Prefix{
		Prefix: prefix,
		Metadata: &models.NetboxMetadata{
			Tenant: tenantName,
			Site:   siteName,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tenantId, *result.Tenant)
	assert.Equal(t, siteId, *result.Site)
}

func TestBuildWritablePrefixRequestV3_TenantError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantName := "NonExistentTenant"
	tenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockTenancy.EXPECT().TenancyTenantsList(tenantListRequestInput, nil).Return(nil, errors.New("tenant not found"))

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Tenancy: mockTenancy},
	}

	result, err := compositeClient.buildWritablePrefixRequestV3(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Tenant: tenantName,
		},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestBuildWritablePrefixRequestV3_SiteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	siteName := "NonExistentSite"
	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&siteName)

	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
	mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(nil, errors.New("site not found"))

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Dcim: mockDcim},
	}

	result, err := compositeClient.buildWritablePrefixRequestV3(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Site: siteName,
		},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
}
