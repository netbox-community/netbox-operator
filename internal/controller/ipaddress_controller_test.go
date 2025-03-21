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

	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("IpAddress Controller", Ordered, func() {

	const timeout = time.Second * 4
	const interval = time.Millisecond * 250

	var unexpectedCallCh chan error

	BeforeEach(func() {
		// Initialize the channel to catch mock calls with unexpected parameters
		unexpectedCallCh = make(chan error)
	})

	AfterEach(func() {
		By("Resetting the mock controller")
		resetMockFunctions(ipamMockIpAddress, ipamMockIpAddressClaim, tenancyMock)
	})

	DescribeTable("Reconciler (ip address CR without owner reference)", func(
		cr *netboxv1.IpAddress, // our CR as typed object
		IpamMocksIpAddress []func(*mock_interfaces.MockIpamInterface, chan error),
		TenancyMocks []func(*mock_interfaces.MockTenancyInterface, chan error),
		restorationHashMismatch bool, // To check for deletion if restoration hash does not match
		expectedConditionReady bool, // Expected state of the ConditionReady condition
		expectedCRStatus netboxv1.IpAddressStatus, // Expected status of the CR
	) {
		By("Setting up mocks")
		for _, mock := range IpamMocksIpAddress {
			mock(ipamMockIpAddress, unexpectedCallCh)
		}
		for _, mock := range TenancyMocks {
			mock(tenancyMock, unexpectedCallCh)
		}

		catchCtx, catchCtxCancel := context.WithCancel(context.Background())
		defer catchCtxCancel()

		// Goroutine to monitor mock calls with unexpected parameters
		go func() {
			defer GinkgoRecover()
			select {
			case errMsg := <-unexpectedCallCh:
				Fail(errMsg.Error())

			case <-catchCtx.Done():
				// Context was cancelled
			}
		}()

		// Create our CR
		By("Creating IpAddress CR")
		Eventually(k8sClient.Create(ctx, cr), timeout, interval).Should(Succeed())

		createdCR := &netboxv1.IpAddress{}

		if restorationHashMismatch {
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		} else {

			// check that reconcile loop did run a least once by checking that conditions are set
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
				return err == nil && len(createdCR.Status.Conditions) > 0
			}, timeout, interval).Should(BeTrue())

			// Now check if conditions are set as expected
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
				return err == nil &&
					apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionIpaddressReadyTrue.Type) == expectedConditionReady
			}, timeout, interval).Should(BeTrue())

			// Check that the expected ip address is present in the status
			Expect(createdCR.Status.IpAddressId).To(Equal(expectedCRStatus.IpAddressId))

			// Cleanup the netbox resources
			Expect(k8sClient.Delete(ctx, createdCR)).Should(Succeed())

			// Wait until the resource is deleted to make sure that it will not interfere with the next test case
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
				return err != client.IgnoreNotFound(err)
			}, timeout, interval).Should(BeTrue())
		}

		catchCtxCancel()
	},
		Entry("Create IpAddress CR, reserve new ip address in NetBox, ",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilterEmptyResult,
				mockIpamIPAddressesCreate,
				mockIpAddressesDelete,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				mockTenancyTenancyTenantsList,
			},
			false, true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, ip address already reserved in NetBox, preserved in netbox, ",
			defaultIpAddressCR(true),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilter,
				mockIpamIPAddressesUpdate,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				mockTenancyTenancyTenantsList,
			},
			false, true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, ip address already reserved in NetBox",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilter,
				mockIpamIPAddressesUpdate,
				mockIpAddressesDelete,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				mockTenancyTenancyTenantsList,
			},
			false, true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, reserve or update failure",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilter,
				mockIpamIPAddressesUpdateFail,
				mockIpAddressesDeleteFail,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				mockTenancyTenancyTenantsList,
			},
			false, false, ExpectedIpAddressFailedStatus),
		Entry("Create IpAddress CR, restoration hash mismatch",
			defaultIpAddressCreatedByClaim(true),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithHashFilterMismatch,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				mockTenancyTenancyTenantsList,
			},
			true, false, nil),
	)
})
