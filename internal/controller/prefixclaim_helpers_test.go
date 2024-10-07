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

package controller

import (
	"testing"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
)

func testPrefixClaimHash(t *testing.T, prefixClaim *netboxv1.PrefixClaim, expectedHash string) {
	generatedHash := generatePrefixRestorationHash(prefixClaim)

	if generatedHash != expectedHash {
		t.Errorf("hash mistatch: expected %#v, got %#v from %#v", expectedHash, generatedHash, prefixClaim)
	}
}

func TestBackwardCompatibilityOfGeneratePrefixRestorationHash(t *testing.T) {
	{
		// output observed when applied config/samples/netbox_v1_prefixclaim.yaml on commit 064e6b
		// concatenated string = "defaultprefixclaim-sample2.0.0.0/16/28Dunder-Mifflin, Inc."
		// sha1 = "a0601ac7e6d196a82c0e61f9be17313113c3043f"
		prefixClaim := &netboxv1.PrefixClaim{
			Spec: netboxv1.PrefixClaimSpec{
				ParentPrefix:         "2.0.0.0/16", // not used, as we read from the ParentPrefix in Status
				PrefixLength:         "/28",
				Tenant:               "Dunder-Mifflin, Inc.",
				ParentPrefixSelector: nil, // TODO(henrybear327): check the default value of this
			},
			Status: netboxv1.PrefixClaimStatus{
				SelectedParentPrefix: "2.0.0.0/16",
			},
		}
		prefixClaim.Namespace = "default"
		prefixClaim.Name = "prefixclaim-sample"

		testPrefixClaimHash(t, prefixClaim, "a0601ac7e6d196a82c0e61f9be17313113c3043f")
	}
}
