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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ownedBy := func(o client.Object, refs ...metav1.OwnerReference) client.Object {
		o.SetOwnerReferences(refs)
		return o
	}

	controllerRef := func(apiVersion, kind string) metav1.OwnerReference {
		controller := true
		return metav1.OwnerReference{APIVersion: apiVersion, Kind: kind, Name: "owner", Controller: &controller}
	}

	It("returns false without any owner reference", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}))).To(BeFalse())
	})

	It("returns false for a non-controlling claim owner", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, metav1.OwnerReference{
			APIVersion: "netbox.dev/v1", Kind: "AsnClaim", Name: "owner",
		}))).To(BeFalse())
	})

	It("returns false for a controller of another api group", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("example.com/v1", "AsnClaim")))).To(BeFalse())
	})

	It("returns false for a netbox.dev controller that is not a claim", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("netbox.dev/v1", "Asn")))).To(BeFalse())
	})

	It("returns false for a claim controller of another kind", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("netbox.dev/v1", "PrefixClaim")))).To(BeFalse())
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Prefix{}, controllerRef("netbox.dev/v1", "AsnClaim")))).To(BeFalse())
	})

	It("returns true for the matching netbox.dev claim controller", func() {
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Asn{}, controllerRef("netbox.dev/v1", "AsnClaim")))).To(BeTrue())
		Expect(isOwnedByClaim(ownedBy(&netboxv1.Prefix{}, controllerRef("netbox.dev/v1", "PrefixClaim")))).To(BeTrue())
		Expect(isOwnedByClaim(ownedBy(&netboxv1.L2VPN{}, controllerRef("netbox.dev/v1", "L2VPNClaim")))).To(BeTrue())
	})
})

var _ = Describe("kindOf", func() {
	DescribeTable("returns the kind of the managed types",
		func(o client.Object, expected string) {
			Expect(kindOf(o)).To(Equal(expected))
		},
		Entry("Asn", &netboxv1.Asn{}, "Asn"),
		Entry("AsnClaim", &netboxv1.AsnClaim{}, "AsnClaim"),
		Entry("IpAddress", &netboxv1.IpAddress{}, "IpAddress"),
		Entry("IpAddressClaim", &netboxv1.IpAddressClaim{}, "IpAddressClaim"),
		Entry("IpRange", &netboxv1.IpRange{}, "IpRange"),
		Entry("IpRangeClaim", &netboxv1.IpRangeClaim{}, "IpRangeClaim"),
		Entry("L2VPN", &netboxv1.L2VPN{}, "L2VPN"),
		Entry("L2VPNClaim", &netboxv1.L2VPNClaim{}, "L2VPNClaim"),
		Entry("Prefix", &netboxv1.Prefix{}, "Prefix"),
		Entry("PrefixClaim", &netboxv1.PrefixClaim{}, "PrefixClaim"),
	)

	It("returns the kind for a typed nil pointer", func() {
		Expect(kindOf((*netboxv1.Prefix)(nil))).To(Equal("Prefix"))
	})

	It("returns an empty string for a nil object", func() {
		Expect(kindOf(nil)).To(BeEmpty())
	})

	It("ignores TypeMeta, which the typed client clears", func() {
		o := &netboxv1.Prefix{TypeMeta: metav1.TypeMeta{Kind: "Asn", APIVersion: "netbox.dev/v1"}}
		Expect(kindOf(o)).To(Equal("Prefix"))
	})
})
