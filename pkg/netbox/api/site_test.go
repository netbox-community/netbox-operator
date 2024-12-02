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

	"github.com/netbox-community/go-netbox/v3/netbox/client/dcim"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSite_GetSiteDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)

	site := "mySite"

	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&site)

	siteOutputId := int64(1)
	siteOutputSlug := "mysite"
	siteListOutput := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{
				{
					ID:   siteOutputId,
					Name: &site,
					Slug: &siteOutputSlug,
				},
			},
		},
	}

	mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(siteListOutput, nil)
	netboxClient := &NetboxClient{Dcim: mockDcim}

	actual, err := netboxClient.GetSiteDetails(site)
	assert.NoError(t, err)
	assert.Equal(t, site, actual.Name)
	assert.Equal(t, siteOutputId, actual.Id)
	assert.Equal(t, siteOutputSlug, actual.Slug)
}

func TestSite_GetEmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)

	site := "mySite"

	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&site)

	emptyListSiteOutput := &dcim.DcimSitesListOK{
		Payload: &dcim.DcimSitesListOKBody{
			Results: []*netboxModels.Site{},
		},
	}

	mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(emptyListSiteOutput, nil)
	netboxClient := &NetboxClient{Dcim: mockDcim}

	actual, err := netboxClient.GetSiteDetails(site)
	assert.Nil(t, actual)
	assert.EqualError(t, err, "failed to fetch site 'mySite': not found")
}

func TestSite_GetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDcim := mock_interfaces.NewMockDcimInterface(ctrl)

	site := "mySite"

	siteListRequestInput := dcim.NewDcimSitesListParams().WithName(&site)

	expectedErr := "error getting sites list"

	mockDcim.EXPECT().DcimSitesList(siteListRequestInput, nil).Return(nil, errors.New(expectedErr))
	netboxClient := &NetboxClient{Dcim: mockDcim}

	actual, err := netboxClient.GetSiteDetails(site)
	assert.Nil(t, actual)
	assert.EqualError(t, err, "failed to fetch Site details: "+expectedErr)
}
