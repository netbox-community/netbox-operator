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

package api

// NetboxCompositeClient holds both the legacy (v3) and modern (v4) clients,
// presenting a single unified interface to callers (controllers).
type NetboxCompositeClient struct {
	clientV3 *NetboxClientV3
	clientV4 *NetboxClientV4
}

// NewNetboxCompositeClient creates a new composite client wrapping both v3 and v4 clients.
func NewNetboxCompositeClient(clientV3 *NetboxClientV3, clientV4 *NetboxClientV4) *NetboxCompositeClient {
	return &NetboxCompositeClient{
		clientV3: clientV3,
		clientV4: clientV4,
	}
}
