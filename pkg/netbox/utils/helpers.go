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

package utils

func NormalizeCustomFields(customFields interface{}) map[string]interface{} {
	switch fields := customFields.(type) {
	case nil:
		return nil
	case map[string]interface{}:
		return fields
	case map[string]string:
		normalized := make(map[string]interface{}, len(fields))
		for key, value := range fields {
			normalized[key] = value
		}
		return normalized
	default:
		return nil
	}
}

func NeedsUpdate[C any, D any](current *C, desired *D, checks ...func(*C, *D) bool) bool {
	if current == nil && desired != nil {
		return true
	}
	if current != nil && desired == nil {
		return true
	}
	for _, check := range checks {
		if check(current, desired) {
			return true
		}
	}
	return false
}

func CompareCustomFields(
	current map[string]interface{},
	desired map[string]interface{},
) bool {

	for k, v := range desired {
		if current[k] != v {
			return true
		}
	}
	return false
}
