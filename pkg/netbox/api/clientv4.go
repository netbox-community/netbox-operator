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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"time"

	v4client "github.com/netbox-community/go-netbox/v4"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/interfaces"
	log "github.com/sirupsen/logrus"
)

type NetboxClientV4 struct {
	client    *v4client.APIClient
	IpamAPI   interfaces.IpamAPI
	StatusAPI interfaces.StatusAPI
}

func GetNetboxClientV4() (*NetboxClientV4, error) {
	logger := log.StandardLogger()
	logger.Debug(fmt.Sprintf("Initializing netbox client v4 at host %v", config.GetOperatorConfig().NetboxHost))

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

	httpClient := &http.Client{
		Transport: &InstrumentedRoundTripper{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		Timeout: time.Second * time.Duration(RequestTimeout),
	}

	desiredRuntimeClientScheme := "http"
	if config.GetOperatorConfig().HttpsEnable {
		desiredRuntimeClientScheme = "https"
	}

	cfg := v4client.NewConfiguration()
	cfg.Scheme = desiredRuntimeClientScheme
	cfg.Host = config.GetOperatorConfig().NetboxHost
	cfg.DefaultHeader["Authorization"] = fmt.Sprintf("Token %v", config.GetOperatorConfig().AuthToken)
	cfg.HTTPClient = httpClient
	client := v4client.NewAPIClient(cfg)

	return &NetboxClientV4{
		client:    client,
		IpamAPI:   &ipamV4APIAdapter{api: client.IpamAPI},
		StatusAPI: &statusV4APIAdapter{api: client.StatusAPI},
	}, nil
}

func handleHTTPResponse(httpResp *http.Response, err error, expectedStatus int, action string) (closeFunc func() error, retErr error) {
	if httpResp != nil && httpResp.Body != nil {
		closeFunc = httpResp.Body.Close
	}

	if httpResp == nil {
		return closeFunc, fmt.Errorf("failed to %s: %w", action, err)
	}

	if httpResp.StatusCode != expectedStatus {
		var body []byte
		if httpResp.Body != nil {
			body, _ = io.ReadAll(httpResp.Body)
		}
		return closeFunc, fmt.Errorf("failed to %s: status %d, body: %s", action, httpResp.StatusCode, string(body))
	}

	if err != nil {
		return closeFunc, fmt.Errorf("failed to %s: %w", action, err)
	}

	return closeFunc, nil
}
