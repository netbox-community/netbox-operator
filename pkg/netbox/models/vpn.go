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

package models

type L2VPN struct {
	Name       string          `json:"name,omitempty"`
	Slug       string          `json:"slug,omitempty"`
	Type       string          `json:"type,omitempty"`
	Identifier int64           `json:"identifier,omitempty"`
	Id         int64           `json:"id,omitempty"`
	Metadata   *NetboxMetadata `json:"metadata,omitempty"`
}

type L2VPNClaim struct {
	Type                 string          `json:"type,omitempty"`
	Identifier           int64           `json:"identifier,omitempty"`
	IdentifierRangeStart int64           `json:"identifierRangeStart,omitempty"`
	IdentifierRangeEnd   int64           `json:"identifierRangeEnd,omitempty"`
	Metadata             *NetboxMetadata `json:"metadata,omitempty"`
}
