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
	"testing"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestConvertAPITagsToModelTags(t *testing.T) {
	apiTags := []netboxv1.Tag{{Name: "first"}, {Slug: "second"}}
	modelTags := convertAPITagsToModelTags(apiTags)

	assert.Len(t, modelTags, 2)
	assert.Equal(t, "first", modelTags[0].Name)
	assert.Equal(t, "second", modelTags[1].Slug)
}

func TestCloneAPITags(t *testing.T) {
	original := []netboxv1.Tag{{Name: "original"}}
	cloned := cloneAPITags(original)

	assert.Len(t, cloned, 1)
	cloned[0].Name = "updated"
	assert.Equal(t, "original", original[0].Name)

	assert.Nil(t, cloneAPITags(nil))
}
