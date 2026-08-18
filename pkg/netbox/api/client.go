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

package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strconv"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	v3client "github.com/netbox-community/go-netbox/v3/netbox/client"
	"github.com/netbox-community/netbox-operator/pkg/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	"github.com/netbox-community/netbox-operator/pkg/netbox/interfaces"
	"k8s.io/client-go/tools/metrics"
)

const (
	RequestTimeout = 1200
)

type NetboxClientV3 struct {
	Ipam    interfaces.IpamInterface
	Tenancy interfaces.TenancyInterface
	Extras  interfaces.ExtrasInterface
	Dcim    interfaces.DcimInterface
}

// Checks that the Netbox host is properly configured for the operator to function.
// Currently only checks that the required custom fields for IP address handling have been added.
func (c *NetboxCompositeClient) VerifyNetboxConfiguration() error {
	customFields, err := c.clientV3.Extras.ExtrasCustomFieldsList(extras.NewExtrasCustomFieldsListParams().WithName(&config.GetOperatorConfig().NetboxRestorationHashFieldName), nil)
	if err != nil {
		return err
	}

	if len(customFields.Payload.Results) != 1 {
		return fmt.Errorf("netbox missing custom field '%s' for restoration hash", config.GetOperatorConfig().NetboxRestorationHashFieldName)
	}
	return nil
}

type InstrumentedRoundTripper struct {
	Transport http.RoundTripper
	// Limiter applies a token-bucket rate limit to outbound NetBox requests.
	// When nil, no throttling is performed.
	Limiter *rate.Limiter
}

func (irt *InstrumentedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if irt.Limiter != nil {
		// Block until the bucket has a token (or the request context is cancelled).
		if err := irt.Limiter.Wait(req.Context()); err != nil {
			return nil, err
		}
	}

	resp, err := irt.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	metrics.RequestResult.Increment(context.TODO(), strconv.Itoa(resp.StatusCode), req.Method, req.Host)
	return resp, nil
}

// newNetboxLimiter builds a token-bucket limiter from the operator config,
// or returns nil when throttling is disabled (QPS <= 0).
func newNetboxLimiter(qps float64, burst int) *rate.Limiter {
	if qps <= 0 {
		return nil
	}
	if burst < 1 {
		burst = 1
	}
	return rate.NewLimiter(rate.Limit(qps), burst)
}

func GetNetboxClient() (*NetboxClientV3, error) {

	logger := log.StandardLogger()
	logger.Debug(fmt.Sprintf("Initializing netbox client at host %v", config.GetOperatorConfig().NetboxHost))

	var desiredRuntimeClientSchemes []string
	desiredRuntimeClientSchemes = []string{"http"}
	if config.GetOperatorConfig().HttpsEnable {
		desiredRuntimeClientSchemes = []string{"https"}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}
	if config.GetOperatorConfig().CaCert != "" {
		caRootPool := x509.NewCertPool()
		certData, err := config.GetOperatorConfig().LoadCaCert()
		if err != nil {
			return nil, err
		}
		ok := caRootPool.AppendCertsFromPEM(certData)
		if !ok {
			logger.Fatalf("Unable to parse certificate at path %s with contents: %s", config.GetOperatorConfig().CaCert, certData)
		}
		tlsConfig.RootCAs = caRootPool
	}

	var baseTransport http.RoundTripper = &InstrumentedRoundTripper{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Limiter: newNetboxLimiter(
			config.GetOperatorConfig().NetboxRateLimitQPS,
			config.GetOperatorConfig().NetboxRateLimitBurst,
		),
	}

	httpClient := &http.Client{
		Transport: baseTransport,
		Timeout:   time.Second * time.Duration(RequestTimeout),
	}

	transport := httptransport.NewWithClient(config.GetOperatorConfig().NetboxHost, v3client.DefaultBasePath, desiredRuntimeClientSchemes, httpClient)
	transport.DefaultAuthentication = httptransport.APIKeyAuth("Authorization",
		"header",
		fmt.Sprintf("Token %v", config.GetOperatorConfig().AuthToken))
	transport.SetLogger(log.StandardLogger())

	auxNetboxClient := v3client.New(transport, nil)
	netboxClient := &NetboxClientV3{
		Ipam:    auxNetboxClient.Ipam,
		Tenancy: auxNetboxClient.Tenancy,
		Extras:  auxNetboxClient.Extras,
		Dcim:    auxNetboxClient.Dcim,
	}

	return netboxClient, nil
}
