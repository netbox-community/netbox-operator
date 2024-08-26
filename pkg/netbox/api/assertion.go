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
	"testing"

	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/stretchr/testify/assert"
)

func AssertNil(t *testing.T, err error) {

	t.Helper()
	assert.Nil(t, err)
}

func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	assert.EqualError(t, err, msg)
}

func AssertIpAddress(t *testing.T, given *netboxModels.WritableIPAddress, actual *netboxModels.IPAddress) {

	t.Helper()

	assert.Greater(t, actual.ID, int64(0))
	assert.Equal(t, given.Address, actual.Address)
	assert.Equal(t, given.Comments, actual.Comments)
	assert.Equal(t, given.Description, actual.Description)
	assert.Equal(t, int64(2), actual.Tenant.ID)
}
