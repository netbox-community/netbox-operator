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

import (
	"fmt"
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

type CustomFieldEntry struct {
	key   string
	value string
}

type QueryFilter struct {
	netBoxFields map[string]string
	customFields []CustomFieldEntry
}

func newQueryFilterOperation(netBoxFields map[string]string, customFields []CustomFieldEntry) func(co *runtime.ClientOperation) {
	return func(co *runtime.ClientOperation) {
		co.Params = &QueryFilter{
			netBoxFields: netBoxFields,
			customFields: customFields,
		}
	}
}

func (o *QueryFilter) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
	// We currently write the request by ANDing all the custom fields

	// The idea is to provide filtering of tenant and site here
	// Doing string filtering on tenant and site doesn't really work though, so we will use tenant_id and site_id instead
	// The query format is like the following: http://localhost:8080/ipam/prefixes/?q=&site_id=2
	for key, value := range o.netBoxFields {
		if err := r.SetQueryParam(url.QueryEscape(key), url.QueryEscape(value)); err != nil {
			return err
		}
	}

	// The custom field query format is like the following: http://localhost:8080/ipam/prefixes/?q=&cf_poolName=Pool+2&cf_environment=Production
	// The GitHub issue related to supporting multiple custom field in a query: https://github.com/netbox-community/netbox/issues/7163
	for _, entry := range o.customFields {
		if err := r.SetQueryParam(fmt.Sprintf("cf_%s", url.QueryEscape(entry.key)), entry.value); err != nil {
			return err
		}
	}

	return nil
}
