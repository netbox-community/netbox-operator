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
	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"

	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

func (r *NetboxClient) GetTagDetails(name string, slug string) (*models.Tag, error) {
	var request *extras.ExtrasTagsListParams
	if name != "" {
		request = extras.NewExtrasTagsListParams().WithName(&name)
	}
	if slug != "" {
		request = extras.NewExtrasTagsListParams().WithSlug(&slug)
	}

	if name == "" && slug == "" {
		return nil, utils.NetboxError("either name or slug must be provided to fetch Tag details", nil)
	}
	// response, err := r.Tags.ExtrasTagsList(request, nil)
	response, err := r.Extras.ExtrasTagsList(request, nil)
	if err != nil {
		return nil, utils.NetboxError("failed to fetch Tag details", err)
	}

	if len(response.Payload.Results) == 0 {
		return nil, utils.NetboxNotFoundError("tag '" + name + "/" + slug + "'")
	}

	tag := response.Payload.Results[0]
	return &models.Tag{
		Id:   tag.ID,
		Name: *tag.Name,
		Slug: *tag.Slug,
	}, nil

}

func (r *NetboxClient) buildWritableTags(tags []models.Tag) ([]*netboxModels.NestedTag, error) {
	nestedTags := make([]*netboxModels.NestedTag, 0, len(tags))
	for _, tag := range tags {
		tagDetails, err := r.GetTagDetails(tag.Name, tag.Slug)
		if err != nil {
			return nil, err
		}
		nestedTags = append(nestedTags, &netboxModels.NestedTag{ID: tagDetails.Id, Name: &tagDetails.Name, Slug: &tagDetails.Slug})
	}
	return nestedTags, nil
}
