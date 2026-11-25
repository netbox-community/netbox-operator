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
	"strings"

	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"

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
		converted[key] = value

		// invalid JSON (e.g. an empty value) is passed as is and validated by NetBox
		if !json.Valid([]byte(value)) {
			continue
		}

		fieldType, err := c.getCustomFieldType(key)
		if err != nil {
			return nil, err
		}
		if !jsonCustomFieldTypes[fieldType] {
			continue
		}

		// UseNumber keeps large integers exact
		decoder := json.NewDecoder(strings.NewReader(value))
		decoder.UseNumber()
		var parsed interface{}
		if err := decoder.Decode(&parsed); err != nil {
			return nil, err
		}
		converted[key] = parsed
	}

	return converted, nil
}

// the type of a custom field can't be changed in NetBox, so it is cached
// if a custom field is deleted and recreated with another type, the operator has to be restarted
func (c *NetboxCompositeClient) getCustomFieldType(name string) (string, error) {
	if fieldType, ok := c.customFieldTypes.Load(name); ok {
		return fieldType.(string), nil
	}

	request := extras.NewExtrasCustomFieldsListParams().WithName(&name)
	response, err := c.clientV3.Extras.ExtrasCustomFieldsList(request, nil)
	if err != nil {
		return "", utils.NetboxError("failed to fetch custom field '"+name+"' details", err)
	}
	// unknown custom fields are passed as is and rejected by NetBox
	// they aren't cached, so a custom field created later is picked up without a restart
	results := response.Payload.Results
	if len(results) != 1 || results[0].Type == nil || results[0].Type.Value == nil {
		return "", nil
	}

	fieldType := *results[0].Type.Value
	c.customFieldTypes.Store(name, fieldType)
	return fieldType, nil
}
