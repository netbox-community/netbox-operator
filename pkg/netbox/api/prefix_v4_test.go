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

	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TODO: add tests for more coverage of the function, e.g. site and tenant not found, only site found, only tenant found, etc.

func TestWritablePrefixRequestV4_NoMetadata(t *testing.T) {
	compositeClient := &NetboxCompositeClient{}

	result, err := compositeClient.writablePrefixRequestV4(&models.Prefix{
		Prefix: "10.0.0.0/24",
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "10.0.0.0/24", result.Prefix)
}

func TestWritablePrefixRequestV4_MetadataNoTenantNoSite(t *testing.T) {
	compositeClient := &NetboxCompositeClient{}

	comments := "my comment"
	description := "my description"
	customFields := map[string]string{"key1": "val1", "key2": "val2"}

	result, err := compositeClient.writablePrefixRequestV4(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Comments:    comments,
			Description: description,
			Custom:      customFields,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "10.0.0.0/24", result.Prefix)
	assert.Equal(t, comments+warningComment, result.GetComments())
	assert.Equal(t, TruncateDescription(description), result.GetDescription())
	assert.Equal(t, "val1", result.GetCustomFields()["key1"])
	assert.Equal(t, "val2", result.GetCustomFields()["key2"])
}

func TestWritablePrefixRequestV4_WithTenant(t *testing.T) {
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

	result, err := compositeClient.writablePrefixRequestV4(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Tenant: tenantName,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(tenantId), *result.GetTenant().Int32)
}

func TestWritablePrefixRequestV4_WithSite(t *testing.T) {
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

	result, err := compositeClient.writablePrefixRequestV4(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Site: siteName,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "dcim.site", result.GetScopeType())
	assert.Equal(t, int32(siteId), result.GetScopeId())
}

func TestWritablePrefixRequestV4_WithTenantAndSite(t *testing.T) {
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

	result, err := compositeClient.writablePrefixRequestV4(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Tenant: tenantName,
			Site:   siteName,
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(tenantId), *result.GetTenant().Int32)
	assert.Equal(t, "dcim.site", result.GetScopeType())
	assert.Equal(t, int32(siteId), result.GetScopeId())
}

func TestWritablePrefixRequestV4_TenantError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantName := "NonExistentTenant"
	tenantListRequestInput := tenancy.NewTenancyTenantsListParams().WithName(&tenantName)

	mockTenancy := mock_interfaces.NewMockTenancyInterface(ctrl)
	mockTenancy.EXPECT().TenancyTenantsList(tenantListRequestInput, nil).Return(nil, errors.New("tenant not found"))

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Tenancy: mockTenancy},
	}

	result, err := compositeClient.writablePrefixRequestV4(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Tenant: tenantName,
		},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestWritablePrefixRequestV4_SiteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	siteName := "NonExistentSite"
	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&siteName)

	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)
	mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(nil, errors.New("site not found"))

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Dcim: mockDcim},
	}

	result, err := compositeClient.writablePrefixRequestV4(&models.Prefix{
		Prefix: "10.0.0.0/24",
		Metadata: &models.NetboxMetadata{
			Site: siteName,
		},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
}
