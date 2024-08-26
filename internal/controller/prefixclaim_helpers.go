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

func generatePrefixFromPrefixClaim(claim *netboxv1.PrefixClaim, prefix string) *netboxv1.Prefix {
	return &netboxv1.Prefix{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.ObjectMeta.Namespace,
		},
		Spec: generatePrefixSpec(claim, prefix),
	}
}

func generatePrefixSpec(claim *netboxv1.PrefixClaim, prefix string) netboxv1.PrefixSpec {
	return netboxv1.PrefixSpec{
		Prefix:           prefix,
		Tenant:           claim.Spec.Tenant,
		CustomFields:     claim.Spec.CustomFields,
		RestorationHash:  generatePrefixRestorationHash(claim),
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generatePrefixRestorationHash(claim *netboxv1.PrefixClaim) string {
	rd := PrefixClaimRestorationData{
		Namespace:    claim.Namespace,
		Name:         claim.Name,
		ParentPrefix: claim.Spec.ParentPrefix,
		PrefixLength: claim.Spec.PrefixLength,
		Tenant:       claim.Spec.Tenant,
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+rd.Name+rd.ParentPrefix+rd.PrefixLength+rd.Tenant)))
}

type PrefixClaimRestorationData struct {
	// only use immutable fields
	Namespace    string
	Name         string
	ParentPrefix string
	PrefixLength string
	Tenant       string
}
