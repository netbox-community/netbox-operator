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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Asn Controller", func() {
	Context("When generating NetBox ASN Model from AsnSpec", func() {
		// default Asn
		asn := &netboxv1.Asn{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
			},
			Spec: netboxv1.AsnSpec{
				Asn:          65001,
				Comments:     "a comment",
				Description:  "a description",
				Tenant:       "a tenant",
				CustomFields: map[string]string{"custom_field_2": "valueToBeSet"},
			},
		}
		asn.Name = "test-asn"

		// default managedCustomFieldsAnnotation
		managedCustomFieldsAnnotation := "{\"custom_field_1\":\"valueToBeRemoved\"}"

		// default request
		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      "test-asn",
				Namespace: "default",
			},
		}

		It("should create the correct ASN model", func() {
			asnModel, err := generateNetboxAsnModelFromAsnSpec(&asn.Spec, req, managedCustomFieldsAnnotation)

			Expect(asnModel).To(Equal(&models.ASN{
				Asn: 65001,
				Metadata: &models.NetboxMetadata{
					Comments:    "a comment",
					Description: "default/test-asn // a description",
					Custom:      map[string]string{"custom_field_2": "valueToBeSet", "custom_field_1": ""},
					Tenant:      "a tenant",
				},
			}))

			Expect(err).To(BeNil())
		})

		It("should return error if parsing of annotation fails", func() {
			invalidManagedCustomFieldsAnnotation := "{:\"valueToBeRemoved\"}"
			asnModel, err := generateNetboxAsnModelFromAsnSpec(&asn.Spec, req, invalidManagedCustomFieldsAnnotation)

			Expect(asnModel).To(BeNil())

			Expect(err).To(HaveOccurred())
		})

		It("should handle empty managed custom fields annotation", func() {
			asnModel, err := generateNetboxAsnModelFromAsnSpec(&asn.Spec, req, "")

			Expect(asnModel).To(Equal(&models.ASN{
				Asn: 65001,
				Metadata: &models.NetboxMetadata{
					Comments:    "a comment",
					Description: "default/test-asn // a description",
					Custom:      map[string]string{"custom_field_2": "valueToBeSet"},
					Tenant:      "a tenant",
				},
			}))

			Expect(err).To(BeNil())
		})

		It("should handle nil custom fields in spec", func() {
			asnNoCustomFields := &netboxv1.Asn{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-asn",
				},
				Spec: netboxv1.AsnSpec{
					Asn:         65001,
					Comments:    "a comment",
					Description: "a description",
					Tenant:      "a tenant",
				},
			}

			asnModel, err := generateNetboxAsnModelFromAsnSpec(&asnNoCustomFields.Spec, req, managedCustomFieldsAnnotation)

			Expect(asnModel).To(Equal(&models.ASN{
				Asn: 65001,
				Metadata: &models.NetboxMetadata{
					Comments:    "a comment",
					Description: "default/test-asn // a description",
					Custom:      map[string]string{"custom_field_1": ""},
					Tenant:      "a tenant",
				},
			}))

			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("Asn Controller reconciling against NetBox", Ordered, func() {

	const timeout = time.Second * 10
	const interval = time.Millisecond * 250

	hashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName

	It("should delete the Asn CR when the restoration hash does not match NetBox", func() {
		store := newAsnStore(65000, 65010)
		installAsnMocks(store)
		// The ASN exists in NetBox but was never allocated by this operator, so it does
		// not carry a restoration hash and must not be adopted.
		store.seed(65001, map[string]interface{}{})

		asn := &netboxv1.Asn{
			ObjectMeta: metav1.ObjectMeta{Name: "asn-hash-mismatch", Namespace: "default"},
			Spec: netboxv1.AsnSpec{
				Asn:          65001,
				Description:  "a description",
				CustomFields: map[string]string{hashKey: "0000000000000000000000000000000000000000"},
			},
		}
		Expect(k8sClient.Create(ctx, asn)).To(Succeed())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: asn.Name, Namespace: asn.Namespace}, &netboxv1.Asn{})
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		// The ASN in NetBox must be left untouched.
		Expect(store.get(65001)).NotTo(BeNil())
	})

	It("should keep the RIR of an existing ASN when updating it", func() {
		store := newAsnStore(65000, 65010)
		installAsnMocks(store)
		hash := "1111111111111111111111111111111111111111"
		store.seed(65002, map[string]interface{}{hashKey: hash})

		asn := &netboxv1.Asn{
			ObjectMeta: metav1.ObjectMeta{Name: "asn-keep-rir", Namespace: "default"},
			Spec: netboxv1.AsnSpec{
				Asn:          65002,
				Description:  "a new description",
				CustomFields: map[string]string{hashKey: hash},
			},
		}
		Expect(k8sClient.Create(ctx, asn)).To(Succeed())

		Eventually(func() bool {
			updated := &netboxv1.Asn{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: asn.Name, Namespace: asn.Namespace}, updated); err != nil {
				return false
			}
			return apismeta.IsStatusConditionTrue(updated.Status.Conditions, netboxv1.ConditionAsnReadyTrue.Type)
		}, timeout, interval).Should(BeTrue())

		stored := store.get(65002)
		Expect(stored).NotTo(BeNil())
		Expect(stored.Description).NotTo(BeNil())
		Expect(*stored.Description).To(ContainSubstring("a new description"))
		Expect(stored.Rir.Get()).NotTo(BeNil())
		Expect(stored.Rir.Get().Id).To(Equal(asnTestRirId))

		Expect(k8sClient.Delete(ctx, asn)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: asn.Name, Namespace: asn.Namespace}, &netboxv1.Asn{})
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	It("should reject ASN values outside of the 32 bit ASN range", func() {
		for _, value := range []int64{0, 4294967296} {
			asn := &netboxv1.Asn{
				ObjectMeta: metav1.ObjectMeta{Name: "asn-invalid", Namespace: "default"},
				Spec:       netboxv1.AsnSpec{Asn: value, Description: "a description"},
			}
			Expect(k8sClient.Create(ctx, asn)).To(MatchError(apierrors.IsInvalid, "invalid"), "ASN %d should be rejected", value)
		}
	})
})
