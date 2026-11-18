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

var _ = Describe("RouteTargetClaim Controller", Ordered, func() {

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
		resetRouteTargetMockFunctions()
	})

	DescribeTable("Reconciler (route target claim CR)", func(
		cr *netboxv1.RouteTargetClaim, // our CR as typed object
		expectedRouteTargetSpec netboxv1.RouteTargetSpec, // spec expected on the RouteTarget CR created by the claim controller
		IpamAPIClaimMocks []func(*mock_interfaces.MockIpamAPI, chan error),
		RouteTargetClaimListRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error),
		IpamAPIMocks []func(*mock_interfaces.MockIpamAPI, chan error),
		RouteTargetListRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error),
		RouteTargetCreateRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error),
		RouteTargetUpdateRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error),
		RouteTargetDestroyRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error),
		expectedConditionReady bool, // Expected state of the ConditionReady condition
		expectedConditionAssigned bool, // Expected state of the ConditionRouteTargetAssigned condition
		expectedCRStatus netboxv1.RouteTargetClaimStatus, // Expected status of the CR
		asnLockedByOtherOwner bool, // If the route target ASN lock is held by another owner when the claim CR is created
	) {
		By("Setting up mocks")
		for _, mock := range IpamAPIClaimMocks {
			mock(mockIpamAPIRouteTargetClaim, unexpectedCallCh)
		}
		for _, mock := range RouteTargetClaimListRequestMocks {
			mock(mockRouteTargetClaimListRequest, unexpectedCallCh)
		}
		for _, mock := range IpamAPIMocks {
			mock(mockIpamAPIRouteTarget, unexpectedCallCh)
		}
		for _, mock := range RouteTargetListRequestMocks {
			mock(mockRouteTargetListRequest, unexpectedCallCh)
		}
		for _, mock := range RouteTargetCreateRequestMocks {
			mock(mockRouteTargetCreateRequest, unexpectedCallCh)
		}
		for _, mock := range RouteTargetUpdateRequestMocks {
			mock(mockRouteTargetUpdateRequest, unexpectedCallCh)
		}
		for _, mock := range RouteTargetDestroyRequestMocks {
			mock(mockRouteTargetDestroyRequest, unexpectedCallCh)
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

		if asnLockedByOtherOwner {
			leaseLockerNSN := types.NamespacedName{
				Name:      routeTargetLockName(routeTargetAsn),
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
		By("Creating RouteTargetClaim CR")
		Eventually(k8sClient.Create(ctx, cr), timeout, interval).Should(Succeed())

		createdCR := &netboxv1.RouteTargetClaim{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		// status RouteTargetAssigned should match expectedConditionAssigned once the claim controller has run
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionRouteTargetAssignedTrue.Type) == expectedConditionAssigned
		}, timeout, interval).Should(BeTrue())

		createdRouteTargetCR := &netboxv1.RouteTarget{}
		if expectedConditionAssigned {
			// check that the route target CR was created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdRouteTargetCR)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// check that the route target claim controller created the route target CR with the correct spec
			Expect(createdRouteTargetCR.Spec).To(Equal(expectedRouteTargetSpec))
		}

		// Now check if conditions are set as expected
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return err == nil &&
				apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionRouteTargetClaimReadyTrue.Type) == expectedConditionReady
		}, timeout, interval).Should(BeTrue())

		// Check that the expected route target/route target name are present in the status
		Expect(createdCR.Status.RouteTarget).To(Equal(expectedCRStatus.RouteTarget))
		Expect(createdCR.Status.RouteTargetName).To(Equal(expectedCRStatus.RouteTargetName))

		// Cleanup the netbox resources
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())

		// Wait until the resources are deleted to make sure that it will not interfere with the next test case
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, cr)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		if expectedConditionAssigned {
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdRouteTargetCR)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		}
	},
		Entry("Create RouteTargetClaim CR, assign new route target from range",
			defaultRouteTargetClaimCRWithRange(), expectedRouteTargetSpecFromClaim(defaultRouteTargetClaimCRWithRange(), "65000:200"),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIClaimList,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetClaimListRequestPaging,
				mockRouteTargetClaimListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIList,
				mockRouteTargetAPICreate,
				mockRouteTargetAPIDestroy,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetListRequestByName("65000:200"),
				mockRouteTargetListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){
				mockRouteTargetCreateRequestBuild,
				mockRouteTargetCreateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){
				mockRouteTargetDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.RouteTargetClaimStatus{RouteTarget: "65000:200", RouteTargetName: routeTargetClaimName},
			false),
		Entry("Create RouteTargetClaim CR, restore existing route target from NetBox",
			defaultRouteTargetClaimCRWithRange(), expectedRouteTargetSpecFromClaim(defaultRouteTargetClaimCRWithRange(), routeTargetRestoredValue),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIClaimList,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetClaimListRequestPaging,
				mockRouteTargetClaimListRequestExecuteWithMatchingHash,
			},
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIList,
				mockRouteTargetAPIUpdate,
				mockRouteTargetAPIDestroy,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetListRequestByName(routeTargetRestoredValue),
				mockRouteTargetListRequestExecuteClaimExistingWithHash(generateRouteTargetRestorationHash(defaultRouteTargetClaimCRWithRange()), routeTargetRestoredValue),
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){
				mockRouteTargetUpdateRequestBuild,
				mockRouteTargetUpdateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){
				mockRouteTargetDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.RouteTargetClaimStatus{RouteTarget: routeTargetRestoredValue, RouteTargetName: routeTargetClaimName},
			false),
		Entry("Create RouteTargetClaim CR, exact route target",
			defaultRouteTargetClaimCRExact(), expectedRouteTargetSpecFromClaim(defaultRouteTargetClaimCRExact(), routeTargetValue),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIClaimList,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetClaimListRequestPaging,
				mockRouteTargetClaimListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIList,
				mockRouteTargetAPICreate,
				mockRouteTargetAPIDestroy,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetListRequestByName(routeTargetValue),
				mockRouteTargetListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){
				mockRouteTargetCreateRequestBuild,
				mockRouteTargetCreateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){
				mockRouteTargetDestroyRequestExecuteSuccess,
			},
			true, true,
			netboxv1.RouteTargetClaimStatus{RouteTarget: routeTargetValue, RouteTargetName: routeTargetClaimName},
			false),
		Entry("Create RouteTargetClaim CR, range-based claim blocked by ASN lock held by other resource",
			defaultRouteTargetClaimCRWithRange(), netboxv1.RouteTargetSpec{},
			[]func(*mock_interfaces.MockIpamAPI, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){},
			[]func(*mock_interfaces.MockIpamAPI, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){},
			false, false,
			netboxv1.RouteTargetClaimStatus{},
			true),
		Entry("Create RouteTargetClaim CR, exact claim blocked by ASN lock held by other resource",
			defaultRouteTargetClaimCRExact(), netboxv1.RouteTargetSpec{},
			[]func(*mock_interfaces.MockIpamAPI, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){},
			[]func(*mock_interfaces.MockIpamAPI, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){},
			false, false,
			netboxv1.RouteTargetClaimStatus{},
			true),
	)
})
