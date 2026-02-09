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
	"time"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Vlan Controller", func() {
	const (
		VlanName      = "static-vlan"
		VlanNamespace = "default"
		VlanId        = 200
		VlanGroupName = "test-group"
		SiteName      = "test-site"
		timeout       = time.Second * 10
		interval      = time.Millisecond * 250
	)

	Context("When creating a Vlan resource", func() {
		It("Should sync it to NetBox", func() {
			By("Defining a new Vlan")
			vlan := &netboxv1.Vlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      VlanName,
					Namespace: VlanNamespace,
				},
				Spec: netboxv1.VlanSpec{
					VlanId:    VlanId,
					VlanGroup: VlanGroupName,
					Site:      SiteName,
					Name:      "my-static-vlan",
				},
			}

			// Reset mocks
			resetAllMockFunctions(ipamMockIpAddress, ipamMockIpAddressClaim, ipamMockVlan, ipamMockVlanClaim, tenancyMock, dcimMock)

			// Mock NetBox interactions
			ipamMockVlan.EXPECT().IpamVlansList(gomock.Any(), gomock.Any()).Return(mockVlansListResponse(), nil).AnyTimes()
			ipamMockVlan.EXPECT().IpamVlansCreate(gomock.Any(), gomock.Any()).Return(&ipam.IpamVlansCreateCreated{Payload: &netboxModels.VLAN{ID: 2, Name: &vlan.Spec.Name, Vid: &[]int64{int64(VlanId)}[0]}}, nil).AnyTimes()

			// Mock VLAN Group details
			ipamMockVlan.EXPECT().IpamVlanGroupsList(gomock.Any(), gomock.Any()).Return(mockVlanGroupsListResponse(VlanGroupName, 1), nil).AnyTimes()

			// Mock Site and Tenant
			dcimMock.EXPECT().DcimSitesList(gomock.Any(), gomock.Any()).Return(mockSitesListResponse(SiteName), nil).AnyTimes()
			tenancyMock.EXPECT().TenancyTenantsList(gomock.Any(), gomock.Any()).Return(mockTenantsListResponse(), nil).AnyTimes()

			By("Creating the Vlan in Kubernetes")
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, vlan)).Should(Succeed())

			By("Checking if the Vlan status was updated")
			fetchedVlan := &netboxv1.Vlan{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: VlanName, Namespace: VlanNamespace}, fetchedVlan)
				if err != nil {
					return false
				}
				return fetchedVlan.Status.VlanId != 0
			}, timeout, interval).Should(BeTrue())

			Expect(fetchedVlan.Status.VlanId).Should(Equal(int64(2)))
		})
	})
})
