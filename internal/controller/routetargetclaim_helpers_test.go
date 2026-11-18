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

	"github.com/go-logr/logr"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newRouteTargetClaimForHash(namespace string, name string, asn int64, start int64, end int64) *netboxv1.RouteTargetClaim {
	return &netboxv1.RouteTargetClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: netboxv1.RouteTargetClaimSpec{
			Asn:                      asn,
			AssignedNumberRangeStart: start,
			AssignedNumberRangeEnd:   end,
			Tenant:                   "MY_TENANT",
		},
	}
}

func TestGenerateRouteTargetRestorationHash(t *testing.T) {
	// pinned value, also used by tests/e2e/routetarget/routetargetclaim-update-ownerreference:
	// sha1("e2e/routetargetclaim-update-ownerreference-1/64607/MY_TENANT/1/1")
	claim := newRouteTargetClaimForHash("e2e", "routetargetclaim-update-ownerreference-1", 64607, 1, 1)
	expectedHash := "aef759f829ac98c78680f97233ac6e6948f9da25"

	if h := generateRouteTargetRestorationHash(claim); h != expectedHash {
		t.Errorf("hash mismatch: expected %s, got %s", expectedHash, h)
	}
}

func TestGenerateRouteTargetRestorationHash_FieldsAreSeparated(t *testing.T) {
	// without separators ASN 6500 + range start 11 and ASN 65001 + range start 1
	// would concatenate to the same input
	a := newRouteTargetClaimForHash("default", "claim", 6500, 11, 20)
	b := newRouteTargetClaimForHash("default", "claim", 65001, 1, 20)

	if generateRouteTargetRestorationHash(a) == generateRouteTargetRestorationHash(b) {
		t.Errorf("expected different hashes for %v and %v", a.Spec, b.Spec)
	}
}

func TestGenerateRouteTargetRestorationHash_IgnoresMutableFields(t *testing.T) {
	a := newRouteTargetClaimForHash("default", "claim", 65000, 1, 10)
	b := a.DeepCopy()
	b.Spec.Description = "changed"
	b.Spec.Comments = "changed"
	b.Spec.PreserveInNetbox = true
	b.Spec.CustomFields = map[string]string{"example_field": "changed"}

	if generateRouteTargetRestorationHash(a) != generateRouteTargetRestorationHash(b) {
		t.Error("restoration hash must only depend on immutable fields")
	}
}

func TestGenerateRouteTargetSpec(t *testing.T) {
	claim := newRouteTargetClaimForHash("default", "claim", 65000, 1, 10)
	claim.Spec.CustomFields = map[string]string{
		"example_field": "value",
		config.GetOperatorConfig().NetboxRestorationHashFieldName: "user-provided",
	}

	spec := generateRouteTargetSpec(claim, "65000:1", logr.Discard())

	if spec.RouteTarget != "65000:1" {
		t.Errorf("expected route target 65000:1, got %s", spec.RouteTarget)
	}
	if spec.CustomFields["example_field"] != "value" {
		t.Errorf("expected custom fields to be copied from claim, got %v", spec.CustomFields)
	}
	if spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] != generateRouteTargetRestorationHash(claim) {
		t.Errorf("expected user-provided restoration hash to be overridden, got %v", spec.CustomFields)
	}
	if _, ok := claim.Spec.CustomFields["example_field"]; !ok || claim.Spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] != "user-provided" {
		t.Error("generateRouteTargetSpec must not mutate the claim's custom fields")
	}
}

func TestRouteTargetLockName(t *testing.T) {
	if n := routeTargetLockName(65000); n != "route-target-asn-65000" {
		t.Errorf("unexpected lock name %s", n)
	}
}
