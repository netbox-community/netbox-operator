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

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"os"

	"github.com/netbox-community/netbox-operator/pkg/netbox/api"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(netboxv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development:     false,
		StacktraceLevel: zapcore.PanicLevel,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})
	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		// TODO(user): TLSOpts is used to allow configuring the TLS config used for the server. If certificates are
		// not provided, self-signed certificates will be generated by default. This option is not recommended for
		// production environments as self-signed certificates do not offer the same level of trust and security
		// as certificates issued by a trusted Certificate Authority (CA). The primary risk is potentially allowing
		// unauthorized access to sensitive metrics data. Consider replacing with CertDir, CertName, and KeyName
		// to provide certificates, ensuring the server communicates using trusted and secure certificates.
		TLSOpts: tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9a39d07a.netbox.dev",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	netboxClient, err := api.GetNetboxClient()
	if err != nil {
		setupLog.Error(err, "failed to initialize netbox client")
		os.Exit(1)
	}
	err = netboxClient.VerifyNetboxConfiguration()
	if err != nil {
		setupLog.Error(err, "verification of netbox configuration failed")
		os.Exit(1)
	}

	// check existence of ENV POD_NAMESPACE
	operatorNamespace, envVarExists := os.LookupEnv("POD_NAMESPACE")
	if !envVarExists {
		setupLog.Error(errors.New("environment variable 'POD_NAMESPACE' must be defined"), "missing required ENV")
		os.Exit(1)
	}

	if err = (&controller.IpAddressReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		EventStatusRecorder: controller.NewEventStatusRecorder(mgr.GetClient(), mgr.GetEventRecorderFor("ip-address-controller")),
		NetboxClient:        netboxClient,
		OperatorNamespace:   operatorNamespace,
		RestConfig:          mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IpAddress")
		os.Exit(1)
	}
	if err = (&controller.IpAddressClaimReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		EventStatusRecorder: controller.NewEventStatusRecorder(mgr.GetClient(), mgr.GetEventRecorderFor("ip-address-claim-controller")),
		NetboxClient:        netboxClient,
		OperatorNamespace:   operatorNamespace,
		RestConfig:          mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IpAddressClaim")
		os.Exit(1)
	}
	if err = (&controller.PrefixReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		EventStatusRecorder: controller.NewEventStatusRecorder(mgr.GetClient(), mgr.GetEventRecorderFor("prefix-controller")),
		NetboxClient:        netboxClient,
		OperatorNamespace:   operatorNamespace,
		RestConfig:          mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Prefix")
		os.Exit(1)
	}
	if err = (&controller.PrefixClaimReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		EventStatusRecorder: controller.NewEventStatusRecorder(mgr.GetClient(), mgr.GetEventRecorderFor("prefix-claim-controller")),
		NetboxClient:        netboxClient,
		OperatorNamespace:   operatorNamespace,
		RestConfig:          mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PrefixClaim")
		os.Exit(1)
	}
	if err = (&controller.IpRangeClaimReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		EventStatusRecorder: controller.NewEventStatusRecorder(mgr.GetClient(), mgr.GetEventRecorderFor("ip-range-claim-controller")),
		NetboxClient:        netboxClient,
		OperatorNamespace:   operatorNamespace,
		RestConfig:          mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IpRangeClaim")
		os.Exit(1)
	}
	if err = (&controller.IpRangeReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		EventStatusRecorder: controller.NewEventStatusRecorder(mgr.GetClient(), mgr.GetEventRecorderFor("ip-range-controller")),
		NetboxClient:        netboxClient,
		OperatorNamespace:   operatorNamespace,
		RestConfig:          mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IpRange")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
