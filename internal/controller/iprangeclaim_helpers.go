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
	"net"
	"strings"

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
	startIP := net.ParseIP(startAddr)
	if startIP == nil {
		return nil, fmt.Errorf("failed to parse start address: %s", startAddr)
	}
	endIP := net.ParseIP(endAddr)
	if endIP == nil {
		return nil, fmt.Errorf("failed to parse end address: %s", endAddr)
	}

	// Determine IP version from the original string notation rather than
	// relying on To4(), which would treat IPv4-mapped IPv6 addresses
	// (e.g. "::ffff:192.168.1.1") as IPv4.
	startIsIPv6 := strings.Contains(startAddr, ":")
	endIsIPv6 := strings.Contains(endAddr, ":")

	if startIsIPv6 != endIsIPv6 {
		return nil, fmt.Errorf("start and end addresses must be of the same IP version")
	}

	if !startIsIPv6 {
		startIP = startIP.To4()
		endIP = endIP.To4()
		if startIP == nil || endIP == nil {
			return nil, fmt.Errorf("failed to normalize IPv4 addresses")
		}
	} else {
		startIP = startIP.To16()
		endIP = endIP.To16()
	}

	if ipGreaterThan(startIP, endIP) {
		return nil, fmt.Errorf("start address %s is greater than end address %s", startAddr, endAddr)
	}

	var ips []string
	for ip := copyIP(startIP); !ipGreaterThan(ip, endIP); incrementIP(ip) {
		ips = append(ips, ip.String())
	}
	return ips, nil
}

func copyIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func ipGreaterThan(a, b net.IP) bool {
	for i := range a {
		if a[i] < b[i] {
			return false
		}
		if a[i] > b[i] {
			return true
		}
	}
	return false
}
