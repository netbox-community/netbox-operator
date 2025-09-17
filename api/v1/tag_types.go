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

package v1

type Tag struct {
	// +kubebuilder:validation:XValidation:rule="(!has(self.name) && has(self.slug)) || (has(self.name) && !has(self.slug))",message="exactly one of name or slug must be specified"
	// +optional
	// Name of the tag
	Name string `json:"name,omitempty"`

	// +optional
	// Slug of the tag
	Slug string `json:"slug,omitempty"`
}
