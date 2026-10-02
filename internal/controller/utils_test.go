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
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
)

var _ = Describe("excludeDomainErrors", func() {
	It("returns nil for a nil error", func() {
		Expect(excludeDomainErrors(nil)).To(BeNil())
	})

	It("keeps a plain non-domain error", func() {
		plainErr := errors.New("plain error")

		remaining := excludeDomainErrors(plainErr)

		Expect(remaining).To(HaveOccurred())
		Expect(errors.Is(remaining, plainErr)).To(BeTrue())
	})

	It("removes a domain error", func() {
		remaining := excludeDomainErrors(NewDomainError("domain failure"))

		Expect(remaining).To(BeNil())
	})

	It("removes wrapped domain errors", func() {
		wrappedDomainErr := fmt.Errorf("outer wrapper: %w", NewDomainError("domain failure"))

		remaining := excludeDomainErrors(wrappedDomainErr)

		Expect(remaining).To(BeNil())
	})

	It("keeps only the non-domain error from a joined error", func() {
		plainErr := errors.New("plain error")
		joinedErr := errors.Join(NewDomainError("domain failure"), plainErr)

		remaining := excludeDomainErrors(joinedErr)

		Expect(remaining).To(HaveOccurred())
		Expect(errors.Is(remaining, plainErr)).To(BeTrue())

		var domainErr *DomainError
		Expect(errors.As(remaining, &domainErr)).To(BeFalse())
	})

	It("returns nil when every joined leaf is a domain error", func() {
		joinedErr := errors.Join(
			NewDomainError("first domain failure"),
			fmt.Errorf("wrapped: %w", NewDomainError("second domain failure")),
		)

		Expect(excludeDomainErrors(joinedErr)).To(BeNil())
	})

	It("preserves all non-domain errors in nested joined errors", func() {
		firstPlainErr := errors.New("first plain error")
		secondPlainErr := errors.New("second plain error")
		nestedJoinedErr := errors.Join(
			errors.Join(NewDomainError("domain failure"), firstPlainErr),
			fmt.Errorf("wrapped: %w", secondPlainErr),
		)

		remaining := excludeDomainErrors(nestedJoinedErr)

		Expect(remaining).To(HaveOccurred())
		Expect(errors.Is(remaining, firstPlainErr)).To(BeTrue())
		Expect(errors.Is(remaining, secondPlainErr)).To(BeTrue())

		var domainErr *DomainError
		Expect(errors.As(remaining, &domainErr)).To(BeFalse())
	})
})

var _ = Describe("isOwnedByClaim", func() {
	var testScheme *runtime.Scheme

	BeforeEach(func() {
		testScheme = runtime.NewScheme()
		Expect(netboxv1.AddToScheme(testScheme)).To(Succeed())
	})

	ownedBy := func(o client.Object, refs ...metav1.OwnerReference) client.Object {
		o.SetOwnerReferences(refs)
		return o
	}

	controllerRef := func(apiVersion, kind string) metav1.OwnerReference {
		controller := true
		return metav1.OwnerReference{APIVersion: apiVersion, Kind: kind, Name: "owner", Controller: &controller}
	}

	It("returns false without any owner reference", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}), testScheme)).To(BeFalse())
	})

	It("returns false for a non-controlling claim owner", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, metav1.OwnerReference{
			APIVersion: "netbox.dev/v1", Kind: "AsnClaim", Name: "owner",
		}), testScheme)).To(BeFalse())
	})

	It("returns false for a controller of another api group", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("example.com/v1", "AsnClaim")), testScheme)).To(BeFalse())
	})

	It("returns false for a netbox.dev controller that is not a claim", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("netbox.dev/v1", "Asn")), testScheme)).To(BeFalse())
	})

	It("returns false for a claim controller of another kind", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("netbox.dev/v1", "PrefixClaim")), testScheme)).To(BeFalse())
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Prefix{}, controllerRef("netbox.dev/v1", "AsnClaim")), testScheme)).To(BeFalse())
	})

	It("returns false for an object that is not registered in the scheme", func() {
		Expect(isOwnedByClaim(ownedBy(&corev1.ConfigMap{}, controllerRef("netbox.dev/v1", "ConfigMapClaim")), testScheme)).To(BeFalse())
	})

	It("resolves the kind from the scheme, not from TypeMeta", func() {
		o := &netboxv1.Prefix{TypeMeta: metav1.TypeMeta{Kind: "Asn", APIVersion: "netbox.dev/v1"}}
		Expect(isOwnedByClaim(ownedBy(o, controllerRef("netbox.dev/v1", "AsnClaim")), testScheme)).To(BeFalse())
		Expect(isOwnedByClaim(ownedBy(o, controllerRef("netbox.dev/v1", "PrefixClaim")), testScheme)).To(BeTrue())
	})

	It("ignores the owner api version, matching on group only", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("netbox.dev/v2", "AsnClaim")), testScheme)).To(BeTrue())
	})

	DescribeTable("pairs every managed object kind with its own claim",
		func(o client.Object, claimKind string) {
			Expect(isOwnedByClaim(ownedBy(o, controllerRef("netbox.dev/v1", claimKind)), testScheme)).To(BeTrue())
		},
		Entry("Asn", &netboxv1.Asn{}, "AsnClaim"),
		Entry("IpAddress", &netboxv1.IpAddress{}, "IpAddressClaim"),
		Entry("IpRange", &netboxv1.IpRange{}, "IpRangeClaim"),
		Entry("L2VPN", &netboxv1.L2VPN{}, "L2VPNClaim"),
		Entry("Prefix", &netboxv1.Prefix{}, "PrefixClaim"),
	)
})
