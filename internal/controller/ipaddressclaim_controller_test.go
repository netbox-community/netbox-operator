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
	"strings"
	"sync"
	"time"

	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/swisscom/leaselocker"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
)

var _ = Describe("IpAddressClaim Controller", Ordered, func() {

	const timeout = time.Second * 4
	const interval = time.Millisecond * 250

	var unexpectedCallCh chan error

	BeforeEach(func() {
		// Initialize the channel to catch mock calls with unexpected parameters
		unexpectedCallCh = make(chan error)
		managerWG := sync.WaitGroup{}
		managerWG.Add(1)
	})

	AfterEach(func() {
		By("Resetting the mock controller")
		resetMockFunctions(ipamMockIpAddress, ipamMockIpAddressClaim, tenancyMock)
	})

	DescribeTable("Reconciler (ip address claim CR)", func(
		cr *netboxv1.IpAddressClaim, // our CR as typed object
		ipcr *netboxv1.IpAddress, // ip address CR expected to be created by ip address claim controller
		ipcrMockStatus netboxv1.IpAddressStatus, // the that will be added to mock the ip address controller
		IpamMocksIpAddressClaim []func(*mock_interfaces.MockIpamInterface, chan error),
		IpamMocksIpAddress []func(*mock_interfaces.MockIpamInterface, chan error),
		TenancyMocks []func(*mock_interfaces.MockTenancyInterface, chan error),
		expectedConditionReady bool, // Expected state of the ConditionReady condition
		expectedConditionIpAssigned bool, // Expected state of the ConditionReady condition
		expectedCRStatus netboxv1.IpAddressClaimStatus, // Expected status of the CR
		prefixLockedByOtherOwner bool, // If prefix is locked by other owner when ipaddress claim CR is created
	) {
		By("Setting up mocks")
		for _, mock := range IpamMocksIpAddressClaim {
			mock(ipamMockIpAddressClaim, unexpectedCallCh)
		}
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

		if prefixLockedByOtherOwner {
			parentPrefixName := strings.ReplaceAll(cr.Spec.ParentPrefix, "/", "-")

			leaseLockerNSN := types.NamespacedName{Name: parentPrefixName, Namespace: OperatorNamespace}
			ll, err := leaselocker.NewLeaseLocker(cfg, leaseLockerNSN, "default/some-other-owner")
			Expect(err).To(BeNil())

			lockCtx, lockCancel := context.WithCancel(ctx)
			defer lockCancel()

			locked := ll.TryLock(lockCtx)
			Expect(locked).To(BeTrue())
		}

		// Create our CR
		By("Creating IpAddressClaim CR")
		Eventually(k8sClient.Create(ctx, cr), timeout, interval).Should(Succeed())

		// check that ip address claim CR was created
		createdCR := &netboxv1.IpAddressClaim{}
		Eventually(func() bool {
			// the created ip address CR has the same namespacedname as the ip address claim CR
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		// status ip assigned should be true if ip address CR was created
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionIpAssignedFalse.Type) == expectedConditionIpAssigned
		}, timeout, interval).Should(BeTrue())

		createdIpCR := &netboxv1.IpAddress{}
		if expectedConditionIpAssigned {
			// check that ip address CR was created
			Eventually(func() bool {
				// the created ip address CR has the same namespaced name as the ip address claim CR
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdIpCR)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// check that the ip address claim controller created the ip address CR with correct spec
			Expect(createdIpCR.Spec).To(Equal(ipcr.Spec))
		}

		// Now check if conditions are set as expected
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionIpClaimReadyTrue.Type) == expectedConditionReady
		}, timeout, interval).Should(BeTrue())

		// Check that the expected ip address is present in the status
		Expect(createdCR.Status.IpAddress).To(Equal(expectedCRStatus.IpAddress))
		Expect(createdCR.Status.IpAddressDotDecimal).To(Equal(expectedCRStatus.IpAddressDotDecimal))
		Expect(createdCR.Status.IpAddressName).To(Equal(expectedCRStatus.IpAddressName))

		// Cleanup the netbox resources
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())

		// Wait until the resources are deleted to make sure that it will not interfere with the next test case
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, cr)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		if expectedConditionIpAssigned {
			Expect(k8sClient.Delete(ctx, createdIpCR)).To(Succeed())
		}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdIpCR)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	},
		Entry("Create IpAddressClaim CR, reserve new ip address in NetBox",
			defaultIpAddressClaimCR(), defaultIpAddressCreatedByClaim(false), ExpectedIpAddressStatus,
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithHashFilterEmptyResult,
				mockPrefixesListWithPrefixFilter,
				mockPrefixesAvailableIpsList,
			},
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilterEmptyResult,
				mockIpamIPAddressesCreateWithHash,
				mockIpAddressesDelete,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				mockTenancyTenancyTenantsList,
			},
			true, true, ExpectedIpAddressClaimStatus, false),
		Entry("Create IpAddressClaim CR, reassign ip from NetBox",
			defaultIpAddressClaimCR(), defaultIpAddressCreatedByClaim(false), ExpectedIpAddressStatus,
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithHashFilter,
			},
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilter,
				mockIpamIPAddressesUpdateWithHash,
				mockIpAddressesDelete,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				mockTenancyTenancyTenantsList,
			},
			true, true, ExpectedIpAddressClaimStatus, false),
		Entry("Create IpAddressClaim CR, prefix locked by other resource",
			defaultIpAddressClaimCR(), defaultIpAddressCreatedByClaim(false), nil,
			nil,
			nil,
			nil,
			false, false, netboxv1.IpAddressClaimStatus{}, true),
	)
})
