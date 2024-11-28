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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
)

var ipRangeRecondiler *IpRangeReconciler

var _ = Describe("IpRange Controller", func() {
	Context("When generating NetBox IpRange  Model form IpRangeSpec", func() {
		// dummy reconciler
		ipRangeRecondiler = &IpRangeReconciler{
			NetboxClient: &api.NetboxClient{
				Ipam:    ipamMockIpAddress,
				Tenancy: tenancyMock,
				Dcim:    dcimMock,
			},
		}

		// default IpRange
		ipRange := &netboxv1.IpRange{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
			},
			Spec: netboxv1.IpRangeSpec{
				StartAddress: "1.0.0.1/32",
				EndAddress:   "1.0.0.5/32",
				Comments:     "a comment",
				Description:  "a description",
				Tenant:       "a tenant",
				CustomFields: map[string]string{"custom_field_2": "valueToBeSet"},
			}}
		ipRange.Name = "test-claim"

		// default managedCustomFieldsAnnotation
		managedCustomFieldsAnnotation := "{\"custom_field_1\":\"valueToBeRemoved\"}"

		// default request
		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      "test-claim",
				Namespace: "default",
			},
		}

		It("should create the correct ip range model", func() {
			ipRangeModel, err := ipRangeRecondiler.generateNetboxIpRangeModelFromIpRangeSpec(ipRange, req, managedCustomFieldsAnnotation)

			Expect(ipRangeModel).To(Equal(&models.IpRange{
				Metadata: &models.NetboxMetadata{
					Comments:    "a comment",
					Description: "default/test-claim // a description // managed by netbox-operator, please don't edit it in Netbox unless you know what you're doing",
					Custom:      map[string]string{"custom_field_2": "valueToBeSet", "custom_field_1": ""},
					Tenant:      "a tenant",
				},
				StartAddress: "1.0.0.1/32",
				EndAddress:   "1.0.0.5/32",
			}))

			Expect(err).To(BeNil())
		})

		It("should return error if parsing of annotation fails", func() {
			invalidManagedCustomFieldsAnnotation := "{:\"valueToBeRemoved\"}"
			ipRangeModel, err := ipRangeRecondiler.generateNetboxIpRangeModelFromIpRangeSpec(ipRange, req, invalidManagedCustomFieldsAnnotation)

			Expect(ipRangeModel).To(BeNil())

			Expect(err).To(HaveOccurred())
		})
	})
})
