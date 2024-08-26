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

type CustomFieldStringFilter struct {
	CustomFieldName  string
	CustomFieldValue string
}

func newCustomFieldStringFilterOperation(name string, value string) func(co *runtime.ClientOperation) {
	return func(co *runtime.ClientOperation) {
		co.Params = &CustomFieldStringFilter{
			CustomFieldName:  name,
			CustomFieldValue: value,
		}
	}
}

func (o *CustomFieldStringFilter) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
	err := r.SetQueryParam(fmt.Sprintf("cf_%s", url.QueryEscape(o.CustomFieldName)), o.CustomFieldValue)
	if err != nil {
		return err
	}
	return nil
}
