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

package controller

import (
	"testing"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
)

func testIpAddressClaimHash(t *testing.T, ipAddressClaim *netboxv1.IpAddressClaim, expectedHash string) {
	generatedHash := generateIpAddressRestorationHash(ipAddressClaim)

	if generatedHash != expectedHash {
		t.Errorf("hash mismatch: expected %#v, got %#v from %#v", expectedHash, generatedHash, ipAddressClaim)
	}
}

func TestBackwardCompatibilityOfGenerateIpAddressRestorationHash(t *testing.T) {
	{
		// pre-VRF hash, computed from the formula before vrf was added
		// concatenated string = "defaultipaddressclaim-sample10.0.0.0/24Dunder-Mifflin, Inc."
		ipAddressClaim := &netboxv1.IpAddressClaim{
			Spec: netboxv1.IpAddressClaimSpec{
				ParentPrefix: "10.0.0.0/24",
				Tenant:       "Dunder-Mifflin, Inc.",
			},
		}
		ipAddressClaim.Namespace = "default"
		ipAddressClaim.Name = "ipaddressclaim-sample"

		testIpAddressClaimHash(t, ipAddressClaim, "58e767e057425e870ab4ed13a5852d159bf36d1e")
	}
	{
		// same claim with spec.vrf set, must produce a different hash
		// concatenated string = "defaultipaddressclaim-sample10.0.0.0/24Dunder-Mifflin, Inc.red"
		ipAddressClaim := &netboxv1.IpAddressClaim{
			Spec: netboxv1.IpAddressClaimSpec{
				ParentPrefix: "10.0.0.0/24",
				Tenant:       "Dunder-Mifflin, Inc.",
				Vrf:          "red",
			},
		}
		ipAddressClaim.Namespace = "default"
		ipAddressClaim.Name = "ipaddressclaim-sample"

		testIpAddressClaimHash(t, ipAddressClaim, "6fee198f39cc575538d67923799e32b32e718639")
	}
}
