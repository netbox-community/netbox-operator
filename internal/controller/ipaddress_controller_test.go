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
	"errors"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type statusPatchInterceptClient struct {
	client.Client
	statusWriter client.SubResourceWriter
}

func (c *statusPatchInterceptClient) Status() client.SubResourceWriter {
	return c.statusWriter
}

type statusPatchInterceptWriter struct {
	client.SubResourceWriter
	patchErr error
}

func (w *statusPatchInterceptWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return w.SubResourceWriter.Create(ctx, obj, subResource, opts...)
}

func (w *statusPatchInterceptWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

func (w *statusPatchInterceptWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if w.patchErr != nil {
		return w.patchErr
	}
	return w.SubResourceWriter.Patch(ctx, obj, patch, opts...)
}

func (w *statusPatchInterceptWriter) Apply(ctx context.Context, obj k8sruntime.ApplyConfiguration, opts ...client.SubResourceApplyOption) error {
	return w.SubResourceWriter.Apply(ctx, obj, opts...)
}

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

	tenancyMocks := []func(*mock_interfaces.MockTenancyInterface, chan error){
		mockTenancyTenancyTenantsList,
	}

	ipAddressListThenUpdate := func(updateMock func(*mock_interfaces.MockIpamInterface, chan error)) []func(*mock_interfaces.MockIpamInterface, chan error) {
		return []func(*mock_interfaces.MockIpamInterface, chan error){
			mockIpAddressListWithIpAddressFilter,
			updateMock,
		}
	}

	ipAddressListThenUpdateTwice := func() []func(*mock_interfaces.MockIpamInterface, chan error) {
		return []func(*mock_interfaces.MockIpamInterface, chan error){
			mockIpAddressListWithIpAddressFilter,
			mockIpamIPAddressesUpdateOnce,
			mockIpamIPAddressesUpdateOnce,
		}
	}

	DescribeTable("Reconciler (ip address CR without owner reference)", func(
		cr *netboxv1.IpAddress, // our CR as typed object
		IpamMocksIpAddress []func(*mock_interfaces.MockIpamInterface, chan error),
		TenancyMocks []func(*mock_interfaces.MockTenancyInterface, chan error),
		restorationHashMismatch bool, // To check for deletion if restoration hash does not match
		expectedConditionReady metav1.Condition, // Expected state of the ConditionReady condition
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
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)).To(Succeed())
				cond := apismeta.FindStatusCondition(createdCR.Status.Conditions, expectedConditionReady.Type)
				g.Expect(cond).NotTo(BeNil())
				g.Expect(cond.Status).To(Equal(expectedConditionReady.Status))
				g.Expect(cond.Reason).To(Equal(expectedConditionReady.Reason))
			}, timeout, interval).Should(Succeed())

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
		Entry("Create IpAddress CR, reserve new ip address in NetBox",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilterEmptyResult,
				mockIpamIPAddressesCreate,
				mockIpAddressesDelete,
			},
			tenancyMocks,
			false, true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, ip address already reserved in NetBox, preserved in netbox",
			defaultIpAddressCR(true),
			ipAddressListThenUpdate(mockIpamIPAddressesUpdateOnce),
			tenancyMocks,
			false, true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, ip address already reserved in NetBox",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithIpAddressFilter,
				mockIpamIPAddressesUpdateOnce,
				mockIpAddressesDelete,
			},
			tenancyMocks,
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
			false, netboxv1.ConditionIpaddressReadyFalse, ExpectedIpAddressFailedStatus),
		Entry("Create IpAddress CR, restoration hash mismatch",
			defaultIpAddressCreatedByClaim(true),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				mockIpAddressListWithHashFilterMismatch,
			},
			tenancyMocks,
			true, false, nil),
		Entry("Create IpAddress CR, skip update when already up to date in NetBox (after convergence)",
			defaultIpAddressCR(true),
			ipAddressListThenUpdateTwice(),
			tenancyMocks,
			false, true, ExpectedIpAddressStatus),
	)
})

var _ = Describe("IpAddress updateStatus", func() {
	newStatusTestObject := func() (*netboxv1.IpAddress, *netboxv1.IpAddress) {
		obj := &netboxv1.IpAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "status-test",
				Namespace: "default",
			},
		}
		return obj, obj.DeepCopy()
	}

	newStatusTestReconciler := func(obj *netboxv1.IpAddress, patchErr error) *IpAddressReconciler {
		baseClient := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithStatusSubresource(obj.DeepCopy()).
			WithObjects(obj.DeepCopy()).
			Build()

		return &IpAddressReconciler{
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

		cond := apismeta.FindStatusCondition(obj.Status.Conditions, netboxv1.ConditionIpaddressReadyFalse.Type)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Message).To(ContainSubstring("reserve failed"))
	})

	It("ignores a not found status patch error after a domain error", func() {
		obj, statusBase := newStatusTestObject()
		notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: "netbox.dev", Resource: "ipaddresses"}, obj.Name)
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
