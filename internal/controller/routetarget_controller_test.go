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
	"errors"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("RouteTarget Controller", Ordered, func() {

	const timeout = time.Second * 4
	const interval = time.Millisecond * 250

	var unexpectedCallCh chan error

	BeforeEach(func() {
		unexpectedCallCh = make(chan error)
	})

	AfterEach(func() {
		By("Resetting the mock controller")
		resetRouteTargetMockFunctions()
	})

	DescribeTable("Reconciler (route target CR without owner reference)", func(
		cr *netboxv1.RouteTarget, // our CR as typed object
		IpamAPIMocks []func(*mock_interfaces.MockIpamAPI, chan error),
		RouteTargetListRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error),
		RouteTargetCreateRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error),
		RouteTargetUpdateRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error),
		RouteTargetDestroyRequestMocks []func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error),
		restorationHashMismatch bool, // To check for deletion if restoration hash does not match
		expectedConditionReady metav1.Condition, // Expected state of the ConditionReady condition
		expectedCRStatus netboxv1.RouteTargetStatus, // Expected status of the CR
	) {
		By("Setting up mocks")
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

		// Create our CR
		By("Creating RouteTarget CR")
		Eventually(k8sClient.Create(ctx, cr), timeout, interval).Should(Succeed())

		createdCR := &netboxv1.RouteTarget{}

		if restorationHashMismatch {
			// the CR is not owned by a RouteTargetClaim, so there is nothing that would
			// recreate it: the controller has to keep it and report the failure instead
			// of silently discarding user owned state
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)).To(Succeed())
				g.Expect(apismeta.IsStatusConditionFalse(createdCR.Status.Conditions, netboxv1.ConditionRouteTargetReadyFalse.Type)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			Expect(createdCR.Status.RouteTargetId).To(BeZero())

			// the CR keeps failing and requeueing, so it has to be removed before the
			// mocks of the next test case are installed
			Expect(k8sClient.Delete(ctx, createdCR)).Should(Succeed())
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
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)).To(Succeed())
				cond := apismeta.FindStatusCondition(createdCR.Status.Conditions, expectedConditionReady.Type)
				g.Expect(cond).NotTo(BeNil())
				g.Expect(cond.Status).To(Equal(expectedConditionReady.Status))
				g.Expect(cond.Reason).To(Equal(expectedConditionReady.Reason))
			}, timeout, interval).Should(Succeed())

			// Check that the expected route target id is present in the status
			Expect(createdCR.Status.RouteTargetId).To(Equal(expectedCRStatus.RouteTargetId))

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
		Entry("Create RouteTarget CR, reserve new route target in NetBox",
			defaultRouteTargetCR(false),
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
			false, netboxv1.ConditionRouteTargetReadyTrue, ExpectedRouteTargetStatus),
		Entry("Create RouteTarget CR, route target already reserved in NetBox, preserved in netbox",
			defaultRouteTargetCR(true),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIList,
				mockRouteTargetAPIUpdate,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetListRequestByName(routeTargetValue),
				mockRouteTargetListRequestExecuteExistingNoHash,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){
				mockRouteTargetUpdateRequestBuild,
				mockRouteTargetUpdateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){},
			false, netboxv1.ConditionRouteTargetReadyTrue, ExpectedRouteTargetStatus),
		Entry("Create RouteTarget CR, route target already reserved in NetBox",
			defaultRouteTargetCR(false),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIList,
				mockRouteTargetAPIUpdate,
				mockRouteTargetAPIDestroy,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetListRequestByName(routeTargetValue),
				mockRouteTargetListRequestExecuteExistingNoHash,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){
				mockRouteTargetUpdateRequestBuild,
				mockRouteTargetUpdateRequestExecuteSuccess,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){
				mockRouteTargetDestroyRequestExecuteSuccess,
			},
			false, netboxv1.ConditionRouteTargetReadyTrue, ExpectedRouteTargetStatus),
		Entry("Create RouteTarget CR, reserve failure",
			defaultRouteTargetCR(false),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIList,
				mockRouteTargetAPICreate,
				mockRouteTargetAPIDestroyZeroId,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetListRequestByName(routeTargetValue),
				mockRouteTargetListRequestExecuteEmpty,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){
				mockRouteTargetCreateRequestBuild,
				mockRouteTargetCreateRequestExecuteFail,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){
				mockRouteTargetDestroyRequestExecuteNotFound,
			},
			false, netboxv1.ConditionRouteTargetReadyFalse, netboxv1.RouteTargetStatus{}),
		Entry("Create RouteTarget CR, restoration hash mismatch, standalone CR is kept",
			defaultRouteTargetCreatedByClaim(true),
			[]func(*mock_interfaces.MockIpamAPI, chan error){
				mockRouteTargetAPIList,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsListRequest, chan error){
				mockRouteTargetListRequestByName(routeTargetValue),
				mockRouteTargetListRequestExecuteExistingWithMismatchedHash,
			},
			[]func(*mock_interfaces.MockIpamRouteTargetsCreateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsUpdateRequest, chan error){},
			[]func(*mock_interfaces.MockIpamRouteTargetsDestroyRequest, chan error){},
			true, metav1.Condition{}, netboxv1.RouteTargetStatus{}),
	)
})

var _ = Describe("RouteTarget updateStatus", func() {
	newStatusTestObject := func() (*netboxv1.RouteTarget, *netboxv1.RouteTarget) {
		obj := &netboxv1.RouteTarget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "status-test",
				Namespace: "default",
			},
		}
		return obj, obj.DeepCopy()
	}

	newStatusTestReconciler := func(obj *netboxv1.RouteTarget, patchErr error) *RouteTargetReconciler {
		baseClient := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithStatusSubresource(obj.DeepCopy()).
			WithObjects(obj.DeepCopy()).
			Build()

		return &RouteTargetReconciler{
			Client: &statusPatchInterceptClient{
				Client: baseClient,
				statusWriter: &statusPatchInterceptWriter{
					SubResourceWriter: baseClient.Status(),
					patchErr:          patchErr,
				},
			},
			EventStatusRecorder: NewEventStatusRecorder(record.NewFakeRecorder(10)),
		}
	}

	It("requeues without returning the domain error when the status patch succeeds", func() {
		obj, statusBase := newStatusTestObject()
		reconciler := newStatusTestReconciler(obj, nil)

		result, err := reconciler.updateStatus(context.Background(), obj, statusBase, ctrl.Result{}, NewDomainError("reserve failed"))

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{Requeue: true}))

		cond := apismeta.FindStatusCondition(obj.Status.Conditions, netboxv1.ConditionRouteTargetReadyFalse.Type)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Message).To(ContainSubstring("reserve failed"))
	})

	It("ignores a not found status patch error after a domain error", func() {
		obj, statusBase := newStatusTestObject()
		notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: "netbox.dev", Resource: "routetargets"}, obj.Name)
		reconciler := newStatusTestReconciler(obj, notFoundErr)

		result, err := reconciler.updateStatus(context.Background(), obj, statusBase, ctrl.Result{}, NewDomainError("reserve failed"))

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{Requeue: true}))
	})

	It("returns only the later patch error when it happens after a domain error", func() {
		obj, statusBase := newStatusTestObject()
		patchErr := errors.New("status patch failed")
		reconciler := newStatusTestReconciler(obj, patchErr)

		result, err := reconciler.updateStatus(context.Background(), obj, statusBase, ctrl.Result{}, NewDomainError("reserve failed"))

		Expect(result).To(Equal(ctrl.Result{}))
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, patchErr)).To(BeTrue())

		var domainErr *DomainError
		Expect(errors.As(err, &domainErr)).To(BeFalse())
	})

	It("keeps both non-domain errors when reconcile and patch both fail", func() {
		obj, statusBase := newStatusTestObject()
		reconcileErr := errors.New("reconcile failed")
		patchErr := errors.New("status patch failed")
		reconciler := newStatusTestReconciler(obj, patchErr)

		result, err := reconciler.updateStatus(context.Background(), obj, statusBase, ctrl.Result{}, reconcileErr)

		Expect(result).To(Equal(ctrl.Result{}))
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, reconcileErr)).To(BeTrue())
		Expect(errors.Is(err, patchErr)).To(BeTrue())
	})
})
