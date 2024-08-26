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
	"time"

	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/swisscom/leaselocker"
	"go.uber.org/mock/gomock"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("IpAddress Controller", func() {

	const timeout = time.Second * 5
	const interval = time.Millisecond * 250

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())

		ipamMock = mock_interfaces.NewMockIpamInterface(mockCtrl)
		tenancyMock = mock_interfaces.NewMockTenancyInterface(mockCtrl)
		extrasMock = mock_interfaces.NewMockExtrasInterface(mockCtrl)

		netboxClient := &api.NetboxClient{
			Ipam:    ipamMock,
			Tenancy: tenancyMock,
			Extras:  extrasMock,
		}

		k8sManager, err := ctrl.NewManager(cfg, k8sManagerOptions)
		Expect(k8sManager.GetConfig()).NotTo(BeNil())
		Expect(err).ToNot(HaveOccurred())

		err = (&IpAddressReconciler{
			Client:            k8sManager.GetClient(),
			Scheme:            k8sManager.GetScheme(),
			Recorder:          k8sManager.GetEventRecorderFor("ip-address-claim-controller"),
			NetboxClient:      netboxClient,
			OperatorNamespace: OperatorNamespace,
			RestConfig:        k8sManager.GetConfig(),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		// Initialize the channel to catch mock calls with unexpected parameters
		unexpectedCallCh = make(chan error)

		go func() {
			defer GinkgoRecover()
			ctx, cancel = context.WithCancel(context.TODO())
			defer cancel()
			err = k8sManager.Start(ctx)
			Expect(err).ToNot(HaveOccurred(), "failed to run manager")
		}()
	})

	AfterEach(func() {
		cancel()
		mockCtrl.Finish()
		// wait so that the k8smanager has time to shut down before next test starts
		time.Sleep(timeout)
	})

	DescribeTable("Reconciler (ip address CR without owner reference)", func(
		cr *netboxv1.IpAddress, // our CR as typed object
		IpamMocks []func(*mock_interfaces.MockIpamInterface, chan error),
		TenancyMocks []func(*mock_interfaces.MockTenancyInterface, chan error),
		expectedConditionReady bool, // Expected state of the ConditionReady condition
		expectedCRStatus netboxv1.IpAddressStatus, // Expected status of the CR
	) {
		By("Setting up mocks")
		for _, mock := range IpamMocks {
			mock(ipamMock, unexpectedCallCh)
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
			case <-time.After(timeout):
				// Test completed without unexpected calls

			case <-catchCtx.Done():
				// Context was cancelled
			}
		}()

		// Create our CR
		By("Creating IpAddress CR")
		Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

		// check that reconsile loop did run a least once by checking that conditions are set
		createdCR := &netboxv1.IpAddress{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return len(createdCR.Status.Conditions) > 0 && err == nil
		}, timeout, interval).Should(BeTrue())

		// Now check if conditions are set as expected
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, createdCR)
			return apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionIpaddressReadyTrue.Type) ==
				expectedConditionReady && err == nil
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

		catchCtxCancel()
	},
		Entry("Create IpAddress CR, reserve new ip address in NetBox, ",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				expectedIpAddressListWithIpAddressFilterEmptyResult,
				expectedIpAddressListWithIpAddressFilter,
				expectedIpamIPAddressesCreate,
				expectedIpamIPAddressesUpdate,
				expectedIpAddressesDelete,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				expectedTenancyTenancyTenantsList,
			},
			true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, reserve new ip address in NetBox, preserved in netbox, ",
			defaultIpAddressCR(true),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				expectedIpAddressListWithIpAddressFilter,
				expectedIpamIPAddressesUpdate,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				expectedTenancyTenancyTenantsList,
			},
			true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, ip address already reserved in NetBox",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				expectedIpAddressListWithIpAddressFilter,
				expectedIpamIPAddressesUpdate,
				expectedIpAddressesDelete,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				expectedTenancyTenancyTenantsList,
			},
			true, ExpectedIpAddressStatus),
		Entry("Create IpAddress CR, reserve or update failure",
			defaultIpAddressCR(false),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				expectedIpAddressListWithIpAddressFilter,
				expectedIpamIPAddressesUpdateFail,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				expectedTenancyTenancyTenantsList,
			},
			false, ExpectedIpAddressFailedStatus),
	)

	DescribeTable("Reconciler (ip address CR with owner reference)", func(
		ipcr *netboxv1.IpAddress, // our CR as typed object
		ipccr *netboxv1.IpAddressClaim,
		IpamMocks []func(*mock_interfaces.MockIpamInterface, chan error),
		TenancyMocks []func(*mock_interfaces.MockTenancyInterface, chan error),
		expectedConditionReady bool, // Expected state of the ConditionReady condition
		expectedCRStatus netboxv1.IpAddressStatus, // Expected status of the CR
	) {
		By("Setting up mocks")
		for _, mock := range IpamMocks {
			mock(ipamMock, unexpectedCallCh)
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
			case <-time.After(timeout):
				// Test completed without unexpected calls

			case <-catchCtx.Done():
				// Context was cancelled
			}
		}()

		// Lock Parent Prefix

		parentPrefixName := strings.Replace(ipccr.Spec.ParentPrefix, "/", "-", -1)

		leaseLockerNSN := types.NamespacedName{Name: parentPrefixName, Namespace: "default"}
		ll, err := leaselocker.NewLeaseLocker(cfg, leaseLockerNSN, "default/ipaddress-test")
		Expect(err).To(BeNil())

		// if the reconciliation of a new ipaddress CR with an owner reference starts
		// ipaddressclaim controller should create a leaselock on the prefix where the
		// ipaddress controller will reserve the ip address
		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		Expect(ll.TryLock(lockCtx)).To(BeTrue())

		// Create our CRs
		By("Creating IpAddressClaim CR")
		Expect(k8sClient.Create(ctx, ipccr)).Should(Succeed())

		// Add owner reference to ip address CR and crete the resource
		Expect(controllerutil.SetOwnerReference(ipccr, ipcr, scheme.Scheme)).Should(Succeed())
		By("Creating IpAddress CR")
		Expect(k8sClient.Create(ctx, ipcr)).Should(Succeed())

		// Check that reconsile loop did run a least once by checking that conditions are set
		createdCR := &netboxv1.IpAddress{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ipcr.GetName(), Namespace: ipcr.GetNamespace()}, createdCR)
			return len(createdCR.Status.Conditions) > 0 && err == nil
		}, timeout, interval).Should(BeTrue())

		// Now check if conditions are set as expected
		createdCR = &netboxv1.IpAddress{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ipcr.GetName(), Namespace: ipcr.GetNamespace()}, createdCR)
			return apismeta.IsStatusConditionTrue(createdCR.Status.Conditions, netboxv1.ConditionIpaddressReadyTrue.Type) ==
				expectedConditionReady && err == nil
		}, timeout, interval).Should(BeTrue())

		// Check that the expected ip address is present in the status
		Expect(createdCR.Status.IpAddressId).To(Equal(expectedCRStatus.IpAddressId))

		// Cleanup the netbox resources
		Expect(k8sClient.Delete(ctx, createdCR)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, ipccr)).Should(Succeed())

		// Wait until the resource is deleted to make sure that it will not interfere with the next test case
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ipcr.GetName(), Namespace: ipcr.GetNamespace()}, createdCR)
			return err != client.IgnoreNotFound(err)
		}, timeout, interval).Should(BeTrue())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ipccr.GetName(), Namespace: ipccr.GetNamespace()}, ipccr)
			return err != client.IgnoreNotFound(err)
		}, timeout, interval).Should(BeTrue())
	},
		Entry("Create IpAddress CR with owner reference, reserve new ip address in NetBox, ",
			defaultIpAddressCR(false), defaultIpAddressClaimCR(),
			[]func(*mock_interfaces.MockIpamInterface, chan error){
				expectedIpAddressListWithIpAddressFilterEmptyResult,
				expectedIpAddressListWithIpAddressFilter,
				expectedIpamIPAddressesCreate,
				expectedIpamIPAddressesUpdate,
				expectedIpAddressesDelete,
			},
			[]func(*mock_interfaces.MockTenancyInterface, chan error){
				expectedTenancyTenancyTenantsList,
			},
			true, ExpectedIpAddressStatus),
	)
})
