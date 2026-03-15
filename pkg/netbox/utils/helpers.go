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

func CheckCustomFields[C any, D any](
	getCurrent func(*C) map[string]interface{},
	getDesired func(*D) map[string]interface{},
) func(*C, *D) bool {
	return func(c *C, d *D) bool {
		current := getCurrent(c)
		for k, v := range getDesired(d) {
			if current[k] != v {
				return true
			}
		}
		return false
	}
}
