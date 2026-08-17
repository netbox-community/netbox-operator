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

var _ = Describe("VlanClaim Controller", Ordered, func() {

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
		resetVlanMockFunctions()
	})

	DescribeTable("Reconciler (vlan claim CR)", func(
		cr *netboxv1.VlanClaim, // our CR as typed object
		expectedVlanSpec netboxv1.VlanSpec, // spec expected on the Vlan CR created by the claim controller
		IpamAPIClaimMocks []func(*mock_interfaces.MockIpamAPI, chan error),
		IpamClaimListRequestMocks []func(*mock_interfaces.MockIpamVlansListRequest, chan error),
		IpamAPIMocks []func(*mock_interfaces.MockIpamAPI, chan error),
		IpamListRequestMocks []func(*mock_interfaces.MockIpamVlansListRequest, chan error),
		IpamCreateRequestMocks []func(*mock_interfaces.MockIpamVlansCreateRequest, chan error),
		IpamUpdateRequestMocks []func(*mock_interfaces.MockIpamVlansUpdateRequest, chan error),
		IpamDestroyRequestMocks []func(*mock_interfaces.MockIpamVlansDestroyRequest, chan error),
		expectedConditionReady bool, // Expected state of the ConditionReady condition
		expectedConditionAssigned bool, // Expected state of the ConditionVlanAssigned condition
		expectedCRStatus netboxv1.VlanClaimStatus, // Expected status of the CR
		rangeLockedByOtherOwner bool, // If the vid range is locked by another owner when the claim CR is created
	) {
		By("Setting up mocks")
		for _, mock := range IpamAPIClaimMocks {
			mock(mockVlanClaimIpamAPI, unexpectedCallCh)
		}
		for _, mock := range IpamClaimListRequestMocks {
			mock(mockVlanClaimListRequest, unexpectedCallCh)
		}
		for _, mock := range IpamAPIMocks {
			mock(mockVlanIpamAPI, unexpectedCallCh)
		}
		for _, mock := range IpamListRequestMocks {
			mock(mockVlanListRequest, unexpectedCallCh)
		}
		for _, mock := range IpamCreateRequestMocks {
			mock(mockVlanCreateRequest, unexpectedCallCh)
		}
		for _, mock := range IpamUpdateRequestMocks {
			mock(mockVlanUpdateRequest, unexpectedCallCh)
		}
		for _, mock := range IpamDestroyRequestMocks {
			mock(mockVlanDestroyRequest, unexpectedCallCh)
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
				Name:      convertVlanRangeToLeaseLockName(cr.Spec.Site, cr.Spec.VidRangeStart, cr.Spec.VidRangeEnd),
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
		By("Creating VlanClaim CR")
		Eventually(k8sClient.Create(ctx, cr), timeout, interval).Should(Succeed())

		createdCR := &netboxv1.VlanClaim{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		// status VlanAssigned should match expectedConditionAssigned once the claim controller has run
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionVlanAssignedTrue.Type) == expectedConditionAssigned
		}, timeout, interval).Should(BeTrue())

		createdVlanCR := &netboxv1.Vlan{}
		if expectedConditionAssigned {
			// check that the vlan CR was created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdVlanCR)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// check that the vlan claim controller created the vlan CR with the correct spec
			Expect(createdVlanCR.Spec).To(Equal(expectedVlanSpec))
		}

		// Now check if conditions are set as expected
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionVlanClaimReadyTrue.Type) == expectedConditionReady
		}, timeout, interval).Should(BeTrue())

		// Check that the expected vid/vlan name are present in the status
		Expect(createdCR.Status.Vid).To(Equal(expectedCRStatus.Vid))
		Expect(createdCR.Status.VlanName).To(Equal(expectedCRStatus.VlanName))

		// Cleanup the netbox resources
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())

		// Wait until the resources are deleted to make sure that it will not interfere with the next test case
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, cr)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		if expectedConditionAssigned {
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdVlanCR)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		}
	},
		Entry("Create VlanClaim CR, assign new vid from range",
			defaultVlanClaimCRWithRange(), expectedVlanSpecFromClaim(defaultVlanClaimCRWithRange(), vlanRangeStart),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockVlanIpamAPIClaimList,
			},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){
				mockVlanClaimListRequestPaging,
				mockVlanClaimListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockVlanIpamAPIList,
				mockVlanIpamAPICreate,
				mockVlanIpamAPIDestroy,
			},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){
				mockVlanListRequestByClaimName,
				mockVlanListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamVlansCreateRequest, chan error){
				mockVlanCreateRequestBuild,
				mockVlanCreateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamVlansUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamVlansDestroyRequest, chan error){
				mockVlanDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.VlanClaimStatus{Vid: vlanRangeStart, VlanName: vlanClaimName},
			false),
		Entry("Create VlanClaim CR, restore existing vid from NetBox",
			defaultVlanClaimCRWithRange(), expectedVlanSpecFromClaim(defaultVlanClaimCRWithRange(), vlanRestoredVid),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockVlanIpamAPIClaimList,
			},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){
				mockVlanClaimListRequestPaging,
				mockVlanClaimListRequestExecuteWithMatchingHash,
			},
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockVlanIpamAPIList,
				mockVlanIpamAPIUpdate,
				mockVlanIpamAPIDestroy,
			},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){
				mockVlanListRequestByClaimName,
				mockVlanListRequestExecuteClaimExistingWithHash(generateVlanRestorationHash(defaultVlanClaimCRWithRange()), vlanRestoredVid),
			},
			[]func(*mock_interfaces.MockIpamVlansCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamVlansUpdateRequest, chan error){
				mockVlanUpdateRequestBuild,
				mockVlanUpdateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamVlansDestroyRequest, chan error){
				mockVlanDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.VlanClaimStatus{Vid: vlanRestoredVid, VlanName: vlanClaimName},
			false),
		Entry("Create VlanClaim CR, explicit vid (no range lock)",
			defaultVlanClaimCRWithVid(), expectedVlanSpecFromClaim(defaultVlanClaimCRWithVid(), vlanVid),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockVlanIpamAPIClaimList,
			},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){
				mockVlanClaimListRequestPaging,
				mockVlanClaimListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockVlanIpamAPIList,
				mockVlanIpamAPICreate,
				mockVlanIpamAPIDestroy,
			},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){
				mockVlanListRequestByClaimName,
				mockVlanListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamVlansCreateRequest, chan error){
				mockVlanCreateRequestBuild,
				mockVlanCreateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamVlansUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamVlansDestroyRequest, chan error){
				mockVlanDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.VlanClaimStatus{Vid: vlanVid, VlanName: vlanClaimName},
			false),
		Entry("Create VlanClaim CR, vid range locked by other resource",
			defaultVlanClaimCRWithRange(), netboxv1.VlanSpec{},
			[]func(*mock_interfaces.MockIpamAPI, chan error){},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){},
			[]func(*mock_interfaces.MockIpamAPI, chan error){},
			[]func(*mock_interfaces.MockIpamVlansListRequest, chan error){},
			[]func(*mock_interfaces.MockIpamVlansCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamVlansUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamVlansDestroyRequest, chan error){},
			false, false,
			netboxv1.VlanClaimStatus{},
			true),
	)
})
