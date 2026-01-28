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
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

func convertAPITagsToModelTags(tags []netboxv1.Tag) []models.Tag {
	if len(tags) == 0 {
		return nil
	}

	converted := make([]models.Tag, len(tags))
	for i, tag := range tags {
		converted[i] = models.Tag{
			Name: tag.Name,
			Slug: tag.Slug,
		}
	}

	return converted
}

func cloneAPITags(tags []netboxv1.Tag) []netboxv1.Tag {
	if len(tags) == 0 {
		return nil
	}

	cloned := make([]netboxv1.Tag, len(tags))
	copy(cloned, tags)

	return cloned
}
