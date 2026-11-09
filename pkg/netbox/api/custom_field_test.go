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

	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/config"
)

// mockCustomFieldTypes expects one lookup per custom field, an empty type means the custom field doesn't exist
func mockCustomFieldTypes(ctrl *gomock.Controller, fieldTypes map[string]string) *mock_interfaces.MockExtrasInterface {
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)
	for name, fieldType := range fieldTypes {
		results := []*netboxModels.CustomField{}
		if fieldType != "" {
			results = append(results, &netboxModels.CustomField{
				Name: &name,
				Type: &netboxModels.CustomFieldType{Value: &fieldType},
			})
		}
		mockExtras.EXPECT().
			ExtrasCustomFieldsList(extras.NewExtrasCustomFieldsListParams().WithName(&name), nil).
			Return(&extras.ExtrasCustomFieldsListOK{Payload: &extras.ExtrasCustomFieldsListOKBody{Results: results}}, nil).
			Times(1)
	}
	return mockExtras
}

func TestCustomField_ConvertCustomFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// the restoration hash is not looked up
	mockExtras := mockCustomFieldTypes(ctrl, map[string]string{
		"text":         "text",
		"integer":      "integer",
		"boolean":      "boolean",
		"decimal":      "decimal",
		"json":         "json",
		"jsonString":   "json",
		"multiobject":  "multiobject",
		"emptyInteger": "integer",
		"unknown":      "",
	})
	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Extras: mockExtras},
	}

	actual, err := compositeClient.convertCustomFields(map[string]string{
		"text":         "42",
		"integer":      "42",
		"boolean":      "true",
		"decimal":      "1.50",
		"json":         `{"capi_cluster_name": "mgmt", "replicas": 3}`,
		"jsonString":   "not json",
		"multiobject":  "[1, 2]",
		"emptyInteger": "",
		"unknown":      "value",
		config.GetOperatorConfig().NetboxRestorationHashFieldName: "hash",
	})

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"text":        "42",
		"integer":     float64(42),
		"boolean":     true,
		"json":        map[string]interface{}{"capi_cluster_name": "mgmt", "replicas": float64(3)},
		"multiobject": []interface{}{float64(1), float64(2)},
		// passed as is, as before
		"decimal":      "1.50",
		"jsonString":   "not json",
		"emptyInteger": "",
		"unknown":      "value",
		config.GetOperatorConfig().NetboxRestorationHashFieldName: "hash",
	}, actual)
}

func TestCustomField_ConvertCustomFieldsLookupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := "json"
	mockExtras := mock_interfaces.NewMockExtrasInterface(ctrl)
	mockExtras.EXPECT().
		ExtrasCustomFieldsList(extras.NewExtrasCustomFieldsListParams().WithName(&name), nil).
		Return(nil, errors.New("connection refused"))
	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Extras: mockExtras},
	}

	actual, err := compositeClient.convertCustomFields(map[string]string{name: "{}"})

	assert.Nil(t, actual)
	assert.ErrorContains(t, err, "failed to fetch CustomField details")
}

func TestCustomField_TypeIsCached(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	compositeClient := &NetboxCompositeClient{
		clientV3: &NetboxClientV3{Extras: mockCustomFieldTypes(ctrl, map[string]string{"boolean": "boolean"})},
	}

	for range 2 {
		actual, err := compositeClient.convertCustomFields(map[string]string{"boolean": "false"})
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{"boolean": false}, actual)
	}
}
