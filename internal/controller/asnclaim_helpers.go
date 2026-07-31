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
	"strconv"

	"github.com/go-logr/logr"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateAsnFromAsnClaim(claim *netboxv1.AsnClaim, asn int64, logger logr.Logger) *netboxv1.Asn {
	asnResource := &netboxv1.Asn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.Namespace,
		},
		Spec: generateAsnSpec(claim, asn, logger),
	}
	return asnResource
}

func generateAsnSpec(claim *netboxv1.AsnClaim, asn int64, logger logr.Logger) netboxv1.AsnSpec {
	// log a warning if the netboxOperatorRestorationHash name is a key in the customFields map of the AsnClaim
	_, ok := claim.Spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName]
	if ok {
		logger.Info(fmt.Sprintf("Warning: restoration hash is calculated from spec, custom field with key %s will be ignored", config.GetOperatorConfig().NetboxRestorationHashFieldName))
	}

	// Copy customFields from claim and add restoration hash
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}

	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateAsnRestorationHash(claim)

	return netboxv1.AsnSpec{
		Asn:              asn,
		Tenant:           claim.Spec.Tenant,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generateAsnRestorationHash(claim *netboxv1.AsnClaim) string {
	rd := AsnClaimRestorationData{
		Namespace:      claim.Namespace,
		Name:           claim.Name,
		ParentAsnRange: claim.Spec.ParentAsnRange,
		Tenant:         claim.Spec.Tenant,
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+rd.Name+rd.ParentAsnRange+rd.Tenant)))
}

type AsnClaimRestorationData struct {
	// only use immutable fields
	Namespace      string
	Name           string
	ParentAsnRange string
	Tenant         string
}

func convertAsnRangeToLeaseLockName(asnRange string) string {
	return "asnrange-" + strconv.FormatInt(int64(sha1Hash(asnRange)), 16)
}

func sha1Hash(s string) uint32 {
	h := sha1.Sum([]byte(s))
	return uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
}
