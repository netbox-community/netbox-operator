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

var _ = Describe("VLANClaim Controller", func() {
	const (
		VlanClaimName      = "test-vlan-claim"
		VlanClaimNamespace = "default"
		VlanId             = 100
		VlanGroupName      = "test-group"
		SiteName           = "test-site"
		timeout            = time.Second * 10
		duration           = time.Second * 10
		interval           = time.Millisecond * 250
	)

	Context("When creating a VLANClaim", func() {
		It("Should create a Vlan resource and update its status", func() {
			By("Defining a new VLANClaim")
			vlanClaim := &netboxv1.VLANClaim{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "netbox.dev/v1",
					Kind:       "VLANClaim",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      VlanClaimName,
					Namespace: VlanClaimNamespace,
				},
				Spec: netboxv1.VLANClaimSpec{
					VlanId:    VlanId,
					VlanGroup: VlanGroupName,
					Site:      SiteName,
					Name:      "my-vlan",
				},
			}

			// Reset mocks
			resetMockFunctions(ipamMockIpAddress, ipamMockIpAddressClaim, ipamMockVlan, ipamMockVlanClaim, tenancyMock, dcimMock)

			// Mock NetBox interactions for VLANClaim (dynamic allocation / restoration)
			// For this test, let's assume restoration finds nothing and we use the provided VlanId
			ipamMockVlanClaim.EXPECT().IpamVlansList(gomock.Any(), gomock.Any()).Return(mockVlansListResponse(0), nil).AnyTimes()
			ipamMockVlanClaim.EXPECT().IpamVlansList(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockVlansListResponse(0), nil).AnyTimes()

			// Mock VLAN Group details for both reconcilers
			ipamMockVlanClaim.EXPECT().IpamVlanGroupsList(gomock.Any(), gomock.Any()).Return(mockVlanGroupsListResponse(VlanGroupName, 1), nil).AnyTimes()
			ipamMockVlan.EXPECT().IpamVlanGroupsList(gomock.Any(), gomock.Any()).Return(mockVlanGroupsListResponse(VlanGroupName, 1), nil).AnyTimes()

			// Mock Site and Tenant (shared mocks)
			dcimMock.EXPECT().DcimSitesList(gomock.Any(), gomock.Any()).Return(mockSitesListResponse(SiteName), nil).AnyTimes()
			tenancyMock.EXPECT().TenancyTenantsList(gomock.Any(), gomock.Any()).Return(mockTenantsListResponse(), nil).AnyTimes()

			// Mock Vlan creation/update for the Vlan controller
			ipamMockVlan.EXPECT().IpamVlansList(gomock.Any(), gomock.Any()).Return(mockVlansListResponse(0), nil).AnyTimes()
			ipamMockVlan.EXPECT().IpamVlansCreate(gomock.Any(), gomock.Any()).Return(&ipam.IpamVlansCreateCreated{Payload: &netboxModels.VLAN{ID: 1, Name: &vlanClaim.Spec.Name, Vid: &[]int64{int64(VlanId)}[0]}}, nil).AnyTimes()
			ipamMockVlan.EXPECT().IpamVlansUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ipam.IpamVlansUpdateOK{Payload: &netboxModels.VLAN{ID: 1, Name: &vlanClaim.Spec.Name, Vid: &[]int64{int64(VlanId)}[0]}}, nil).AnyTimes()

			By("Creating the VLANClaim in Kubernetes")
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, vlanClaim)).Should(Succeed())

			By("Checking if the Vlan resource was created")
			vlanLookupKey := types.NamespacedName{Name: VlanClaimName, Namespace: VlanClaimNamespace}
			createdVlan := &netboxv1.Vlan{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, vlanLookupKey, createdVlan)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdVlan.Spec.VlanId).Should(Equal(VlanId))
			Expect(createdVlan.Spec.Name).Should(Equal("my-vlan"))
			Expect(createdVlan.Spec.Site).Should(Equal(SiteName))
			Expect(createdVlan.Spec.VlanGroup).Should(Equal(VlanGroupName))

			By("Mocking Vlan controller status update")
			// In a real integration test, the Vlan controller would update the Vlan status.
			// Since we registered both in suite_test, this should happen automatically if we mock correctly.

			By("Checking if the VLANClaim status was updated")
			fetchedClaim := &netboxv1.VLANClaim{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: VlanClaimName, Namespace: VlanClaimNamespace}, fetchedClaim)
				if err != nil {
					return false
				}
				return fetchedClaim.Status.VlanName == VlanClaimName
			}, timeout, interval).Should(BeTrue())
		})
	})

	AfterEach(func() {
		resetMockFunctions(ipamMockIpAddress, ipamMockIpAddressClaim, ipamMockVlan, ipamMockVlanClaim, tenancyMock, dcimMock)

		// Clean up
		vlanClaim := &netboxv1.VLANClaim{}
		err := k8sClient.Get(context.Background(), types.NamespacedName{Name: VlanClaimName, Namespace: VlanClaimNamespace}, vlanClaim)
		if err == nil {
			Expect(k8sClient.Delete(context.Background(), vlanClaim)).Should(Succeed())
		}
	})
})
