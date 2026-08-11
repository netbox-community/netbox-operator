/*
Copyright 2026 Swisscom (Schweiz) AG.

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

var _ = Describe("L2VPNClaim Controller", Ordered, func() {

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
		resetVpnMockFunctions()
	})

	DescribeTable("Reconciler (l2vpn claim CR)", func(
		cr *netboxv1.L2VPNClaim, // our CR as typed object
		expectedL2VPNSpec netboxv1.L2VPNSpec, // spec expected on the L2VPN CR created by the claim controller
		VpnAPIClaimMocks []func(*mock_interfaces.MockVpnAPI, chan error),
		VpnClaimListRequestMocks []func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error),
		VpnAPIMocks []func(*mock_interfaces.MockVpnAPI, chan error),
		VpnListRequestMocks []func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error),
		VpnCreateRequestMocks []func(*mock_interfaces.MockVpnL2vpnsCreateRequest, chan error),
		VpnUpdateRequestMocks []func(*mock_interfaces.MockVpnL2vpnsUpdateRequest, chan error),
		VpnDestroyRequestMocks []func(*mock_interfaces.MockVpnL2vpnsDestroyRequest, chan error),
		expectedConditionReady bool, // Expected state of the ConditionReady condition
		expectedConditionAssigned bool, // Expected state of the ConditionL2VPNAssigned condition
		expectedCRStatus netboxv1.L2VPNClaimStatus, // Expected status of the CR
		rangeLockedByOtherOwner bool, // If the identifier range is locked by another owner when the claim CR is created
	) {
		By("Setting up mocks")
		for _, mock := range VpnAPIClaimMocks {
			mock(mockVpnAPIClaim, unexpectedCallCh)
		}
		for _, mock := range VpnClaimListRequestMocks {
			mock(mockVpnClaimListRequest, unexpectedCallCh)
		}
		for _, mock := range VpnAPIMocks {
			mock(mockVpnAPI, unexpectedCallCh)
		}
		for _, mock := range VpnListRequestMocks {
			mock(mockVpnListRequest, unexpectedCallCh)
		}
		for _, mock := range VpnCreateRequestMocks {
			mock(mockVpnCreateRequest, unexpectedCallCh)
		}
		for _, mock := range VpnUpdateRequestMocks {
			mock(mockVpnUpdateRequest, unexpectedCallCh)
		}
		for _, mock := range VpnDestroyRequestMocks {
			mock(mockVpnDestroyRequest, unexpectedCallCh)
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

		if rangeLockedByOtherOwner {
			leaseLockerNSN := types.NamespacedName{
				Name:      convertL2VPNRangeToLeaseLockName(cr.Spec.Type, cr.Spec.IdentifierRangeStart, cr.Spec.IdentifierRangeEnd),
				Namespace: OperatorNamespace,
			}
			ll, err := leaselocker.NewLeaseLocker(cfg, leaseLockerNSN, "default/some-other-owner")
			Expect(err).To(BeNil())

			lockCtx, lockCancel := context.WithCancel(ctx)
			defer lockCancel()

			locked := ll.TryLock(lockCtx)
			Expect(locked).To(BeTrue())
		}

		// Create our CR
		By("Creating L2VPNClaim CR")
		Eventually(k8sClient.Create(ctx, cr), timeout, interval).Should(Succeed())

		createdCR := &netboxv1.L2VPNClaim{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		// status L2VPNAssigned should match expectedConditionAssigned once the claim controller has run
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionL2VPNAssignedTrue.Type) == expectedConditionAssigned
		}, timeout, interval).Should(BeTrue())

		createdL2VPNCR := &netboxv1.L2VPN{}
		if expectedConditionAssigned {
			// check that the l2vpn CR was created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdL2VPNCR)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// check that the l2vpn claim controller created the l2vpn CR with the correct spec
			Expect(createdL2VPNCR.Spec).To(Equal(expectedL2VPNSpec))
		}

		// Now check if conditions are set as expected
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionL2VPNClaimReadyTrue.Type) == expectedConditionReady
		}, timeout, interval).Should(BeTrue())

		// Check that the expected identifier/l2vpn name are present in the status
		Expect(createdCR.Status.Identifier).To(Equal(expectedCRStatus.Identifier))
		Expect(createdCR.Status.L2VPNName).To(Equal(expectedCRStatus.L2VPNName))

		// Cleanup the netbox resources
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())

		// Wait until the resources are deleted to make sure that it will not interfere with the next test case
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, cr)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		if expectedConditionAssigned {
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdL2VPNCR)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		}
	},
		Entry("Create L2VPNClaim CR, assign new identifier from range",
			defaultL2VPNClaimCRWithRange(), expectedL2VPNSpecFromClaim(defaultL2VPNClaimCRWithRange(), l2vpnRangeStart),
			[]func(*mock_interfaces.MockVpnAPI, chan error){
				mockVpnAPIClaimList,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){
				mockVpnClaimListRequestPaging,
				mockVpnClaimListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockVpnAPI, chan error){
				mockVpnAPIList,
				mockVpnAPICreate,
				mockVpnAPIDestroy,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){
				mockVpnListRequestByClaimName,
				mockVpnListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsCreateRequest, chan error){
				mockVpnCreateRequestBuild,
				mockVpnCreateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsDestroyRequest, chan error){
				mockVpnDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.L2VPNClaimStatus{Identifier: l2vpnRangeStart, L2VPNName: l2vpnClaimName},
			false),
		Entry("Create L2VPNClaim CR, restore existing identifier from NetBox",
			defaultL2VPNClaimCRWithRange(), expectedL2VPNSpecFromClaim(defaultL2VPNClaimCRWithRange(), l2vpnRestoredIdentifier),
			[]func(*mock_interfaces.MockVpnAPI, chan error){
				mockVpnAPIClaimList,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){
				mockVpnClaimListRequestPaging,
				mockVpnClaimListRequestExecuteWithMatchingHash,
			},
			[]func(*mock_interfaces.MockVpnAPI, chan error){
				mockVpnAPIList,
				mockVpnAPIUpdate,
				mockVpnAPIDestroy,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){
				mockVpnListRequestByClaimName,
				mockVpnListRequestExecuteClaimExistingWithHash(generateL2VPNRestorationHash(defaultL2VPNClaimCRWithRange()), l2vpnRestoredIdentifier),
			},
			[]func(*mock_interfaces.MockVpnL2vpnsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsUpdateRequest, chan error){
				mockVpnUpdateRequestBuild,
				mockVpnUpdateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsDestroyRequest, chan error){
				mockVpnDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.L2VPNClaimStatus{Identifier: l2vpnRestoredIdentifier, L2VPNName: l2vpnClaimName},
			false),
		Entry("Create L2VPNClaim CR, explicit identifier (no range lock)",
			defaultL2VPNClaimCRWithIdentifier(), expectedL2VPNSpecFromClaim(defaultL2VPNClaimCRWithIdentifier(), l2vpnIdentifier),
			[]func(*mock_interfaces.MockVpnAPI, chan error){
				mockVpnAPIClaimList,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){
				mockVpnClaimListRequestPaging,
				mockVpnClaimListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockVpnAPI, chan error){
				mockVpnAPIList,
				mockVpnAPICreate,
				mockVpnAPIDestroy,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){
				mockVpnListRequestByClaimName,
				mockVpnListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsCreateRequest, chan error){
				mockVpnCreateRequestBuild,
				mockVpnCreateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockVpnL2vpnsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsDestroyRequest, chan error){
				mockVpnDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.L2VPNClaimStatus{Identifier: l2vpnIdentifier, L2VPNName: l2vpnClaimName},
			false),
		Entry("Create L2VPNClaim CR, identifier range locked by other resource",
			defaultL2VPNClaimCRWithRange(), netboxv1.L2VPNSpec{},
			[]func(*mock_interfaces.MockVpnAPI, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){},
			[]func(*mock_interfaces.MockVpnAPI, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsListRequest, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockVpnL2vpnsDestroyRequest, chan error){},
			false, false,
			netboxv1.L2VPNClaimStatus{},
			true),
	)
})
