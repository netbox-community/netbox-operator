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
