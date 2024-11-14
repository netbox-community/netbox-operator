/*
Copyright 2024 The Kubernetes authors.

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

// Modified by Swisscom (Schweiz) AG.
// Copyright 2024 Swisscom (Schweiz) AG

package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	"go.uber.org/mock/gomock"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	netboxdevv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var k8sManagerOptions ctrl.Options
var testEnv *envtest.Environment
var mockCtrl *gomock.Controller
var ipamMockIpAddress *mock_interfaces.MockIpamInterface
var ipamMockIpAddressClaim *mock_interfaces.MockIpamInterface
var tenancyMock *mock_interfaces.MockTenancyInterface
var dcimMock *mock_interfaces.MockDcimInterface
var ctx context.Context
var cancel context.CancelFunc

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.29.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = netboxv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("defining k8sManager option to disable metrics server")
	k8sManagerOptions = ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: server.Options{
			BindAddress: "0", // Disable the metrics server
		},
	}

	err = netboxdevv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mockCtrl = gomock.NewController(GinkgoT(), gomock.WithOverridableExpectations())

	ipamMockIpAddress = mock_interfaces.NewMockIpamInterface(mockCtrl)
	ipamMockIpAddressClaim = mock_interfaces.NewMockIpamInterface(mockCtrl)
	tenancyMock = mock_interfaces.NewMockTenancyInterface(mockCtrl)
	dcimMock = mock_interfaces.NewMockDcimInterface(mockCtrl)

	k8sManager, err := ctrl.NewManager(cfg, k8sManagerOptions)
	Expect(k8sManager.GetConfig()).NotTo(BeNil())
	Expect(err).ToNot(HaveOccurred())

	err = (&IpAddressReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("ip-address-claim-controller"),
		NetboxClient: &api.NetboxClient{
			Ipam:    ipamMockIpAddress,
			Tenancy: tenancyMock,
			Dcim:    dcimMock,
		},
		OperatorNamespace: OperatorNamespace,
		RestConfig:        k8sManager.GetConfig(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&IpAddressClaimReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("ip-address-claim-controller"),
		NetboxClient: &api.NetboxClient{
			Ipam:    ipamMockIpAddressClaim,
			Tenancy: tenancyMock,
			Dcim:    dcimMock,
		},
		OperatorNamespace: OperatorNamespace,
		RestConfig:        k8sManager.GetConfig(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		ctx, cancel = context.WithCancel(context.TODO())
		defer func() { cancel() }()

		Expect(k8sManager.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	mockCtrl.Finish()

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
