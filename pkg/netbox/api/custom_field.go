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
	"encoding/json"

	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"

	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/utils"
)

// values of these custom field types are parsed as JSON
var jsonCustomFieldTypes = map[string]bool{
	"integer":     true,
	"boolean":     true,
	"json":        true,
	"multiselect": true,
	"object":      true,
	"multiobject": true,
}

func (c *NetboxCompositeClient) convertCustomFields(customFields map[string]string) (map[string]interface{}, error) {
	converted := make(map[string]interface{}, len(customFields))
	for key, value := range customFields {
		// the restoration hash is always a text field
		if key == config.GetOperatorConfig().NetboxRestorationHashFieldName {
			converted[key] = value
			continue
		}

		fieldType, err := c.getCustomFieldType(key)
		if err != nil {
			return nil, err
		}
		if !jsonCustomFieldTypes[fieldType] {
			converted[key] = value
			continue
		}

		// invalid JSON (e.g. an empty value) is passed as is and validated by NetBox
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err != nil {
			converted[key] = value
			continue
		}
		converted[key] = parsed
	}

	return converted, nil
}

// NetBox doesn't allow to change the type of a custom field, so it is cached
func (c *NetboxCompositeClient) getCustomFieldType(name string) (string, error) {
	if fieldType, ok := c.customFieldTypes.Load(name); ok {
		return fieldType.(string), nil
	}

	request := extras.NewExtrasCustomFieldsListParams().WithName(&name)
	response, err := c.clientV3.Extras.ExtrasCustomFieldsList(request, nil)
	if err != nil {
		return "", utils.NetboxError("failed to fetch CustomField details", err)
	}
	// unknown custom fields are passed as is and rejected by NetBox
	if len(response.Payload.Results) != 1 || response.Payload.Results[0].Type == nil {
		return "", nil
	}

	fieldType := *response.Payload.Results[0].Type.Value
	c.customFieldTypes.Store(name, fieldType)
	return fieldType, nil
}
