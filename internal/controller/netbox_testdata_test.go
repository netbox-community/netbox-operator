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

package controller

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/v3/netbox/client/tenancy"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// -----------------------------
// default Values of CRs
// -----------------------------
var name = "ipaddress-test"
var namespace = "default"

var description = "integration test"

var comments = "integration test comment"

var siteSlug = "mars-ip-claim"

var ipAddress = "1.0.0.1/32"
var ipAddressFamily = int64(4)
var parentPrefix = "1.0.0.0/28"

var siteId = int64(2)
var site = "Mars"

var tenantId = int64(1)
var tenant = "test-tenant"
var tenantSlug = "test-tenant-slug"

var restorationHash = "6f6c67651f0b43b2969ba2ae35c74fc91815513b"

var customFields = map[string]string{"example_field": "example value"}
var customFieldsWithHash = map[string]string{"example_field": "example value", "netboxOperatorRestorationHash": restorationHash}

var netboxLabel = "Status"
var value = "active"

// -----------------------------
// default CRs
// -----------------------------

func defaultIpAddressCR(preserveInNetbox bool) *netboxv1.IpAddress {
	return &netboxv1.IpAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: netboxv1.IpAddressSpec{
			IpAddress:        ipAddress,
			Tenant:           tenant,
			CustomFields:     customFields,
			Comments:         comments,
			Description:      description,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

func defaultIpAddressCreatedByClaim(preserveInNetbox bool) *netboxv1.IpAddress {
	return &netboxv1.IpAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: netboxv1.IpAddressSpec{
			IpAddress:        ipAddress,
			Tenant:           tenant,
			CustomFields:     customFieldsWithHash,
			Comments:         comments,
			Description:      description,
			PreserveInNetbox: preserveInNetbox,
		},
	}
}

func defaultIpAddressClaimCR() *netboxv1.IpAddressClaim {
	return &netboxv1.IpAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: netboxv1.IpAddressClaimSpec{
			ParentPrefix:     parentPrefix,
			Tenant:           tenant,
			CustomFields:     customFields,
			Comments:         comments,
			Description:      description,
			PreserveInNetbox: false,
		},
	}
}

// -----------------------------
// netbox mock responses
// -----------------------------

func mockedResponseNestedSite() *netboxModels.NestedSite {
	return &netboxModels.NestedSite{
		ID:   siteId,
		Name: &site,
		Slug: &siteSlug,
	}
}

func mockedResponseNestedTenant() *netboxModels.NestedTenant {
	return &netboxModels.NestedTenant{
		Name: &tenant,
		ID:   tenantId,
		Slug: &siteSlug,
	}
}

func mockedResponseExpectedAvailableIpAddress() []*netboxModels.AvailableIP {
	return []*netboxModels.AvailableIP{
		{
			Address: ipAddress,
			Family:  ipAddressFamily,
		},
	}
}

func mockedResponseIPAddress() *netboxModels.IPAddress {
	currentTime := strfmt.DateTime(time.Now())
	return &netboxModels.IPAddress{
		ID:          int64(1),
		Address:     &ipAddress,
		Display:     ipAddress,
		Created:     &currentTime,
		LastUpdated: &currentTime,
		Comments:    comments,
		Description: description,
		Tenant:      mockedResponseNestedTenant(),
		Status: &netboxModels.IPAddressStatus{
			Label: &netboxLabel,
			Value: &value,
		}}
}

func mockedResponsePrefixList() *ipam.IpamPrefixesListOKBody {
	return &ipam.IpamPrefixesListOKBody{
		Results: []*netboxModels.Prefix{
			{
				ID:          prefixID,
				Comments:    comments,
				Description: description,
				Display:     parentPrefix,
				Prefix:      &parentPrefix,
				Site:        mockedResponseNestedSite(),
				Tenant:      mockedResponseNestedTenant(),
			},
		},
	}
}

func mockedResponseIPAddressList() *ipam.IpamIPAddressesListOKBody {
	return &ipam.IpamIPAddressesListOKBody{
		Results: []*netboxModels.IPAddress{
			{
				ID:          mockedResponseIPAddress().ID,
				Address:     mockedResponseIPAddress().Address,
				Comments:    mockedResponseIPAddress().Comments,
				Description: mockedResponseIPAddress().Description,
				Display:     mockedResponseIPAddress().Display,
				Tenant:      mockedResponseIPAddress().Tenant,
			},
		},
	}
}

func mockedResponseEmptyIPAddressList() *ipam.IpamIPAddressesListOKBody {
	count := int64(0)
	return &ipam.IpamIPAddressesListOKBody{
		Count:   &count,
		Results: make([]*netboxModels.IPAddress, 0),
	}
}

func mockedResponseTenancyTenantsList() *tenancy.TenancyTenantsListOKBody {
	return &tenancy.TenancyTenantsListOKBody{
		Results: []*netboxModels.Tenant{
			{
				ID:   int64(1),
				Name: &tenant,
				Slug: &tenantSlug,
			},
		},
	}
}

// -----------------------------
// netbox mock expected params
// -----------------------------

// expected inputs for ipam.IpamIPAddressesUpdate()
var nsn = namespace + "/" + name + " // "
var warningComment = " // managed by netbox-operator, please don't edit it in Netbox unless you know what you're doing"
var expectedIpAddressID = int64(1)

var expectedIpToUpdate = &netboxModels.WritableIPAddress{
	Address:  &ipAddress,
	Comments: comments + warningComment,
	CustomFields: map[string]string{
		"example_field": "example value",
	},
	Description: nsn + description + warningComment,
	Status:      "active",
	Tenant:      &tenantId}

var expectedIpToUpdateWithHash = &netboxModels.WritableIPAddress{
	Address:  &ipAddress,
	Comments: comments + warningComment,
	CustomFields: map[string]string{
		"example_field":                 "example value",
		"netboxOperatorRestorationHash": restorationHash,
	},
	Description: nsn + description + warningComment,
	Status:      "active",
	Tenant:      &tenantId}

var ExpectedIpAddressUpdateParams = ipam.NewIpamIPAddressesUpdateParams().WithDefaults().
	WithData(expectedIpToUpdate).WithID(expectedIpAddressID)

var ExpectedIpAddressUpdateWithHashParams = ipam.NewIpamIPAddressesUpdateParams().WithDefaults().
	WithData(expectedIpToUpdateWithHash).WithID(expectedIpAddressID)

var ExpectedTenantsListParams = tenancy.NewTenancyTenantsListParams().WithName(&tenant)

// expected inputs for ipam.IpamPrefixesList method
var ExpectedPrefixListParams = ipam.NewIpamPrefixesListParams().WithPrefix(&parentPrefix)

// expected inputs for ipam.IpamPrefixesAvailableIpsList method
var prefixID = int64(4)

var ExpectedPrefixesAvailableIpsListParams = ipam.NewIpamPrefixesAvailableIpsListParams().WithID(prefixID)

// expected inputs for ipam.IpamIPAddressesList method
var ExpectedIpAddressListParamsWithIpAddressData = ipam.NewIpamIPAddressesListParams().WithAddress(&ipAddress)

var ExpectedIpAddressListParams = ipam.NewIpamIPAddressesListParams()

// expected inputs for ipam.IpamIPAddressesCreate method
var ExpectedIpAddressesCreateParams = ipam.NewIpamIPAddressesCreateParams().WithDefaults().WithData(expectedIpToUpdate)

var ExpectedIpAddressesCreateWithHashParams = ipam.NewIpamIPAddressesCreateParams().WithDefaults().WithData(expectedIpToUpdateWithHash)

// expected inputs for ipam.IpamIPAddressesDelete method
var ExpectedDeleteParams = ipam.NewIpamIPAddressesDeleteParams().WithID(expectedIpAddressID)

var ExpectedIpAddressStatus = netboxv1.IpAddressStatus{IpAddressId: 1}

var ExpectedIpAddressFailedStatus = netboxv1.IpAddressStatus{IpAddressId: 0}

var OperatorNamespace = "default"

var ExpectedIpAddressClaimStatus = netboxv1.IpAddressClaimStatus{
	IpAddress:           ipAddress,
	IpAddressDotDecimal: "1.0.0.1",
	IpAddressName:       name,
}
