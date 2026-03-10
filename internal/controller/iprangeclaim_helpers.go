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
	"crypto/sha1"
	"fmt"
	"net/netip"

	"github.com/go-logr/logr"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateIpRangeFromIpRangeClaim(ctx context.Context, claim *netboxv1.IpRangeClaim, startIp string, endIp string) *netboxv1.IpRange {
	logger := log.FromContext(ctx)
	ipRangeResource := &netboxv1.IpRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.Namespace,
		},
		Spec: generateIpRangeSpec(claim, startIp, endIp, logger),
	}
	return ipRangeResource
}

func generateIpRangeSpec(claim *netboxv1.IpRangeClaim, startIp string, endIp string, logger logr.Logger) netboxv1.IpRangeSpec {
	// log a warning if the netboxOperatorRestorationHash name is a key in the customFields map of the IpRangeClaim
	_, ok := claim.Spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName]
	if ok {
		logger.Info(fmt.Sprintf("Warning: restoration hash is calculated from spec, custom field with key %s will be ignored", config.GetOperatorConfig().NetboxRestorationHashFieldName))
	}

	// Copy customFields from claim and add restoration hash
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}

	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateIpRangeRestorationHash(claim)

	return netboxv1.IpRangeSpec{
		StartAddress:     startIp,
		EndAddress:       endIp,
		Tenant:           claim.Spec.Tenant,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generateIpRangeRestorationHash(claim *netboxv1.IpRangeClaim) string {
	rd := IpRangeClaimRestorationData{
		Namespace:    claim.Namespace,
		Name:         claim.Name,
		ParentPrefix: claim.Spec.ParentPrefix,
		Tenant:       claim.Spec.Tenant,
		Size:         fmt.Sprintf("%d", claim.Spec.Size),
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+rd.Name+rd.ParentPrefix+rd.Tenant+rd.Size)))
}

type IpRangeClaimRestorationData struct {
	// only use immutable fields
	Namespace    string
	Name         string
	ParentPrefix string
	Tenant       string
	Size         string
}

// ipsInRange returns all IP addresses from startAddr to endAddr (inclusive).
// Supports both IPv4 and IPv6 addresses.
func ipsInRange(startAddr, endAddr string) ([]string, error) {
	startIP, err := netip.ParseAddr(startAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start address: %s", startAddr)
	}
	endIP, err := netip.ParseAddr(endAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end address: %s", endAddr)
	}

	if startIP.Is6() != endIP.Is6() {
		return nil, fmt.Errorf("start and end addresses must be of the same IP version")
	}

	if endIP.Less(startIP) {
		return nil, fmt.Errorf("start address %s is greater than end address %s", startAddr, endAddr)
	}

	var ips []string
	for ip := startIP; ; ip = ip.Next() {
		ips = append(ips, ip.String())
		if ip == endIP {
			break
		}
	}
	return ips, nil
}
