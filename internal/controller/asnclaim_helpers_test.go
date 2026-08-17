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
	"crypto/sha1"
	"fmt"
	"testing"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func testAsnClaimHash(t *testing.T, claim *netboxv1.AsnClaim, expectedHash string) {
	generatedHash := generateAsnRestorationHash(claim)

	if generatedHash != expectedHash {
		t.Errorf("hash mismatch: expected %#v, got %#v from %#v", expectedHash, generatedHash, claim)
	}
}

func TestGenerateAsnRestorationHash(t *testing.T) {
	claim := &netboxv1.AsnClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-asn-claim",
		},
		Spec: netboxv1.AsnClaimSpec{
			ParentAsnRange: "private-range",
			Tenant:         "test-tenant",
		},
	}

	// Manually compute expected hash: sha1("defaultmy-asn-claimprivate-rangetest-tenant")
	expectedHash := fmt.Sprintf("%x", sha1.Sum([]byte("defaultmy-asn-claimprivate-rangetest-tenant")))
	testAsnClaimHash(t, claim, expectedHash)
}

func TestGenerateAsnRestorationHash_NoTenant(t *testing.T) {
	claim := &netboxv1.AsnClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "production",
			Name:      "asn-claim-no-tenant",
		},
		Spec: netboxv1.AsnClaimSpec{
			ParentAsnRange: "public-range",
		},
	}

	expectedHash := fmt.Sprintf("%x", sha1.Sum([]byte("productionasn-claim-no-tenantpublic-range")))
	testAsnClaimHash(t, claim, expectedHash)
}

func TestGenerateAsnFromAsnClaim(t *testing.T) {
	claim := &netboxv1.AsnClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-asn-claim",
		},
		Spec: netboxv1.AsnClaimSpec{
			ParentAsnRange:   "private-range",
			Tenant:           "test-tenant",
			Comments:         "test comment",
			Description:      "test description",
			CustomFields:     map[string]string{"field1": "value1"},
			PreserveInNetbox: true,
		},
	}

	asnResource := generateAsnFromAsnClaim(claim, 65001, ctrl.Log)

	if asnResource.Name != "my-asn-claim" {
		t.Errorf("expected name %q, got %q", "my-asn-claim", asnResource.Name)
	}
	if asnResource.Namespace != "default" {
		t.Errorf("expected namespace %q, got %q", "default", asnResource.Namespace)
	}
	if asnResource.Spec.Asn != 65001 {
		t.Errorf("expected ASN %d, got %d", 65001, asnResource.Spec.Asn)
	}
	if asnResource.Spec.Tenant != "test-tenant" {
		t.Errorf("expected tenant %q, got %q", "test-tenant", asnResource.Spec.Tenant)
	}
	if asnResource.Spec.Comments != "test comment" {
		t.Errorf("expected comments %q, got %q", "test comment", asnResource.Spec.Comments)
	}
	if asnResource.Spec.Description != "test description" {
		t.Errorf("expected description %q, got %q", "test description", asnResource.Spec.Description)
	}
	if !asnResource.Spec.PreserveInNetbox {
		t.Error("expected PreserveInNetbox to be true")
	}
	if asnResource.Spec.CustomFields["field1"] != "value1" {
		t.Errorf("expected custom field field1=%q, got %q", "value1", asnResource.Spec.CustomFields["field1"])
	}

	// Check restoration hash is set
	hashFieldName := config.GetOperatorConfig().NetboxRestorationHashFieldName
	if _, ok := asnResource.Spec.CustomFields[hashFieldName]; !ok {
		t.Errorf("expected restoration hash field %q to be set", hashFieldName)
	}
}

func TestConvertAsnRangeToLeaseLockName(t *testing.T) {
	name := convertAsnRangeToLeaseLockName("private-range")

	// Should start with "asnrange-"
	if len(name) < 10 {
		t.Errorf("expected lease lock name to start with 'asnrange-', got %q", name)
	}
	if name[:9] != "asnrange-" {
		t.Errorf("expected lease lock name to start with 'asnrange-', got %q", name)
	}

	// Different ranges should produce different lock names
	name2 := convertAsnRangeToLeaseLockName("public-range")
	if name == name2 {
		t.Errorf("expected different lock names for different ranges, both got %q", name)
	}

	// Same range should produce same lock name (deterministic)
	name3 := convertAsnRangeToLeaseLockName("private-range")
	if name != name3 {
		t.Errorf("expected same lock name for same range, got %q and %q", name, name3)
	}
}
