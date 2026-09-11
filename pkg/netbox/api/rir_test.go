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
	"context"
	"net/http"
	"testing"

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rirName := "RFC 6996"

	t.Run("get RIR details by name", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		expectRirLookup(ctrl, mockIpamAPI, rirName,
			[]v4client.RIR{{Id: 9, Name: rirName, Slug: "rfc-6996"}})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getRirDetailsByName(context.TODO(), rirName)

		assert.NoError(t, err)
		assert.Equal(t, int64(9), actual.Id)
		assert.Equal(t, rirName, actual.Name)
		assert.Equal(t, "rfc-6996", actual.Slug)
	})

	t.Run("unknown RIR name", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		expectRirLookup(ctrl, mockIpamAPI, "Nonexistent", []v4client.RIR{})

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getRirDetailsByName(context.TODO(), "Nonexistent")

		assert.Error(t, err)
		assert.Nil(t, actual)
	})

	t.Run("server error", func(t *testing.T) {
		mockIpamAPI := mock_interfaces.NewMockIpamAPI(ctrl)
		req := mock_interfaces.NewMockIpamRirsListRequest(ctrl)
		mockIpamAPI.EXPECT().IpamRirsList(gomock.Any()).Return(req)
		req.EXPECT().Name([]string{rirName}).Return(req)
		req.EXPECT().Limit(int32(listPageSize)).Return(req)
		req.EXPECT().Execute().
			Return(nil, &http.Response{StatusCode: 500, Body: http.NoBody}, nil)

		compositeClient := &NetboxCompositeClient{clientV4: &NetboxClientV4{IpamAPI: mockIpamAPI}}

		actual, err := compositeClient.getRirDetailsByName(context.TODO(), rirName)

		assert.Error(t, err)
		assert.Nil(t, actual)
	})
}
