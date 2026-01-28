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
	"time"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
)

var _ = Describe("Service LoadBalancer Controller", Ordered, func() {
	const timeout = time.Second * 4
	const interval = time.Millisecond * 250

	BeforeEach(func() {
		allowIpAddressNetboxCalls(ipamMockIpAddress)
	})

	AfterEach(func() {
		resetMockFunctions(ipamMockIpAddress, ipamMockIpAddressClaim, tenancyMock)
	})

	It("creates and prunes IpAddress CRs for LoadBalancer services", func() {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lb-service",
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{
					{Port: 80},
				},
			},
		}

		By("Creating a load balancer service")
		Eventually(k8sClient.Create(ctx, service), timeout, interval).Should(Succeed())

		By("Adding ingress IPs to service status")
		Eventually(func() error {
			latest := &corev1.Service{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, latest); err != nil {
				return err
			}
			latest.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
				{IP: "1.0.0.1"},
				{IP: "1.0.0.2"},
			}
			return k8sClient.Status().Update(ctx, latest)
		}, timeout, interval).Should(Succeed())

		By("Refreshing service metadata for label matching")
		Eventually(func() error {
			latest := &corev1.Service{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, latest); err != nil {
				return err
			}
			service = latest
			return nil
		}, timeout, interval).Should(Succeed())

		By("Ensuring ipaddress CRs are created for both IPs")
		Eventually(func() ([]string, error) {
			list := &netboxv1.IpAddressList{}
			if err := k8sClient.List(ctx, list, client.InNamespace(service.Namespace), client.MatchingLabels(serviceLoadBalancerLabels(service))); err != nil {
				return nil, err
			}
			ipAddresses := make([]string, 0, len(list.Items))
			for _, item := range list.Items {
				ipAddresses = append(ipAddresses, item.Spec.IpAddress)
			}
			return ipAddresses, nil
		}, timeout, interval).Should(ConsistOf("1.0.0.1/32", "1.0.0.2/32"))

		By("Removing one ingress IP and pruning the stale ipaddress")
		Eventually(func() error {
			latest := &corev1.Service{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, latest); err != nil {
				return err
			}
			latest.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
				{IP: "1.0.0.1"},
			}
			return k8sClient.Status().Update(ctx, latest)
		}, timeout, interval).Should(Succeed())

		Eventually(func() ([]string, error) {
			list := &netboxv1.IpAddressList{}
			if err := k8sClient.List(ctx, list, client.InNamespace(service.Namespace), client.MatchingLabels(serviceLoadBalancerLabels(service))); err != nil {
				return nil, err
			}
			ipAddresses := make([]string, 0, len(list.Items))
			for _, item := range list.Items {
				ipAddresses = append(ipAddresses, item.Spec.IpAddress)
			}
			return ipAddresses, nil
		}, timeout, interval).Should(ConsistOf("1.0.0.1/32"))

		By("Cleaning up the service")
		Expect(k8sClient.Delete(ctx, service)).To(Succeed())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, &corev1.Service{})
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})

func allowIpAddressNetboxCalls(ipamMock *mock_interfaces.MockIpamInterface) {
	ipamMock.EXPECT().IpamIPAddressesList(gomock.Any(), gomock.Any()).
		Return(&ipam.IpamIPAddressesListOK{Payload: mockedResponseEmptyIPAddressList()}, nil).
		AnyTimes()
	ipamMock.EXPECT().IpamIPAddressesCreate(gomock.Any(), nil).
		Return(&ipam.IpamIPAddressesCreateCreated{Payload: mockedResponseIPAddress()}, nil).
		AnyTimes()
	ipamMock.EXPECT().IpamIPAddressesDelete(gomock.Any(), nil).
		Return(&ipam.IpamIPAddressesDeleteNoContent{}, nil).
		AnyTimes()
	ipamMock.EXPECT().IpamIPAddressesUpdate(gomock.Any(), nil).
		Return(&ipam.IpamIPAddressesUpdateOK{Payload: &netboxModels.IPAddress{ID: 1}}, nil).
		AnyTimes()
}
