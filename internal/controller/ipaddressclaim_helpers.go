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
	"crypto/sha1"
	"fmt"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateIpAddressFromIpAddressClaim(claim *netboxv1.IpAddressClaim, ip string) *netboxv1.IpAddress {
	ipAddressResource := &netboxv1.IpAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.ObjectMeta.Namespace,
		},
		Spec: generateIpAddressSpec(claim, ip),
	}
	return ipAddressResource
}

func generateIpAddressSpec(claim *netboxv1.IpAddressClaim, ip string) netboxv1.IpAddressSpec {
	return netboxv1.IpAddressSpec{
		IpAddress:        ip,
		Tenant:           claim.Spec.Tenant,
		CustomFields:     claim.Spec.CustomFields,
		RestorationHash:  generateIpAddressRestorationHash(claim),
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generateIpAddressRestorationHash(claim *netboxv1.IpAddressClaim) string {
	rd := IpAddressClaimRestorationData{
		Namespace:    claim.Namespace,
		Name:         claim.Name,
		ParentPrefix: claim.Spec.ParentPrefix,
		Tenant:       claim.Spec.Tenant,
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+rd.Name+rd.ParentPrefix+rd.Tenant)))
}

type IpAddressClaimRestorationData struct {
	// only use immutable fields
	Namespace    string
	Name         string
	ParentPrefix string
	Tenant       string
}
