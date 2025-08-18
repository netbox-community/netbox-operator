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

	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTags_GetTagDetailsByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)

	tagName := "myTag"
	tagSlug := "mytag"
	tagId := int64(1)

	tagListRequestInput := extras.NewExtrasTagsListParams().WithName(&tagName)
	tagListOutput := &extras.ExtrasTagsListOK{
		Payload: &extras.ExtrasTagsListOKBody{
			Results: []*netboxModels.Tag{
				{
					ID:   tagId,
					Name: &tagName,
					Slug: &tagSlug,
				},
			},
		},
	}

	mockExtras.EXPECT().ExtrasTagsList(tagListRequestInput, nil).Return(tagListOutput, nil)
	netboxClient := &NetboxClient{Extras: mockExtras}

	actual, err := netboxClient.GetTagDetails(tagName, "")
	assert.NoError(t, err)
	assert.Equal(t, tagName, actual.Name)
	assert.Equal(t, tagId, actual.Id)
	assert.Equal(t, tagSlug, actual.Slug)
}

func TestTags_GetTagDetailsBySlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)

	tagName := "myTag"
	tagSlug := "mytag"
	tagId := int64(1)

	tagListRequestInput := extras.NewExtrasTagsListParams().WithSlug(&tagSlug)
	tagListOutput := &extras.ExtrasTagsListOK{
		Payload: &extras.ExtrasTagsListOKBody{
			Results: []*netboxModels.Tag{
				{
					ID:   tagId,
					Name: &tagName,
					Slug: &tagSlug,
				},
			},
		},
	}

	mockExtras.EXPECT().ExtrasTagsList(tagListRequestInput, nil).Return(tagListOutput, nil)
	netboxClient := &NetboxClient{Extras: mockExtras}

	actual, err := netboxClient.GetTagDetails("", tagSlug)
	assert.NoError(t, err)
	assert.Equal(t, tagName, actual.Name)
	assert.Equal(t, tagId, actual.Id)
	assert.Equal(t, tagSlug, actual.Slug)
}

func TestTags_GetTagDetailsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)

	tagName := "notfound"
	tagListRequestInput := extras.NewExtrasTagsListParams().WithName(&tagName)
	tagListOutput := &extras.ExtrasTagsListOK{
		Payload: &extras.ExtrasTagsListOKBody{
			Results: []*netboxModels.Tag{},
		},
	}

	mockExtras.EXPECT().ExtrasTagsList(tagListRequestInput, nil).Return(tagListOutput, nil)
	netboxClient := &NetboxClient{Extras: mockExtras}

	actual, err := netboxClient.GetTagDetails(tagName, "")
	assert.Nil(t, actual)
	assert.EqualError(t, err, "failed to fetch tag 'notfound/': not found")
}

func TestTags_GetTagDetailsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)

	tagName := "error"
	tagListRequestInput := extras.NewExtrasTagsListParams().WithName(&tagName)

	mockExtras.EXPECT().ExtrasTagsList(tagListRequestInput, nil).Return(nil, errors.New("some error"))
	netboxClient := &NetboxClient{Extras: mockExtras}

	actual, err := netboxClient.GetTagDetails(tagName, "")
	assert.Nil(t, actual)
	assert.Contains(t, err.Error(), "failed to fetch Tag details")
}

func TestTags_GetTagDetailsNoInput(t *testing.T) {
	netboxClient := &NetboxClient{}
	actual, err := netboxClient.GetTagDetails("", "")
	assert.Nil(t, actual)
	assert.Contains(t, err.Error(), "either name or slug must be provided")
}
