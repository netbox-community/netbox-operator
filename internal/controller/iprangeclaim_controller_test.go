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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("IpRangeClaim Controller", func() {
	Context("When generating the ip range spec", func() {
		It("should create the correct spec", func() {
			ctx := context.TODO()

			claim := &netboxv1.IpRangeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: netboxv1.IpRangeClaimSpec{
					ParentPrefix: "1.0.0.1/28",
					Comments:     "test",
				}}

			claim.Name = "test-claim"

			ipRange := generateIpRangeFromIpRangeClaim(ctx, claim, "1.0.0.1/32", "1.0.0.3/32")
			Expect(ipRange).To(Equal(&netboxv1.IpRange{
				ObjectMeta: metav1.ObjectMeta{
					Name:      claim.Name,
					Namespace: claim.ObjectMeta.Namespace,
				},
				Spec: netboxv1.IpRangeSpec{
					Comments:     "test",
					StartAddress: "1.0.0.1/32",
					EndAddress:   "1.0.0.3/32",
					CustomFields: map[string]string{config.GetOperatorConfig().NetboxRestorationHashFieldName: "331f244f24c08ea3fc6fb7f16cbef20ef2bf02de"},
				},
			}))
		})
	})
})
