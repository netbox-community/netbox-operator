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

package models

type Tenant struct {
	Id   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type Site struct {
	Id   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type NetboxMetadata struct {
	Comments    string            `json:"comments,omitempty"`
	Custom      map[string]string `json:"customFields,omitempty"`
	Description string            `json:"description,omitempty"`
	Region      string            `json:"region,omitempty"`
	Site        string            `json:"site,omitempty"`
	Tenant      string            `json:"tenant,omitempty"`
}

type IPAddress struct {
	IpAddress string          `json:"ipAddress,omitempty"`
	Metadata  *NetboxMetadata `json:"metadata,omitempty"`
}

type IPAddressClaim struct {
	ParentPrefix string          `json:"parentPrefix,omitempty"`
	Metadata     *NetboxMetadata `json:"metadata,omitempty"`
}

type Prefix struct {
	Prefix   string          `json:"prefix,omitempty"`
	Metadata *NetboxMetadata `json:"metadata,omitempty"`
}

type PrefixClaim struct {
	ParentPrefix string          `json:"parentPrefix,omitempty"`
	PrefixLength string          `json:"prefixLength,omitempty"`
	Metadata     *NetboxMetadata `json:"metadata,omitempty"`
}

type IpRange struct {
	StartAddress string          `json:"startAddress,omitempty"`
	EndAddress   string          `json:"endAddress,omitempty"`
	Metadata     *NetboxMetadata `json:"metadata,omitempty"`
}

type IpRangeClaim struct {
	ParentPrefix string          `json:"prefix,omitempty"`
	Size         int             `json:"size,omitempty"`
	Metadata     *NetboxMetadata `json:"metadata,omitempty"`
}
