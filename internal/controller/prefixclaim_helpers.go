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
	"sort"

	"github.com/go-logr/logr"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generatePrefixFromPrefixClaim(claim *netboxv1.PrefixClaim, prefix string, logger logr.Logger) *netboxv1.Prefix {
	return &netboxv1.Prefix{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.ObjectMeta.Namespace,
		},
		Spec: generatePrefixSpec(claim, prefix, logger),
	}
}

func generatePrefixSpec(claim *netboxv1.PrefixClaim, prefix string, logger logr.Logger) netboxv1.PrefixSpec {
	// log a warning if the netboxOperatorRestorationHash name is a key in the customFields map of the IpAddressClaim
	_, ok := claim.Spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName]
	if ok {
		logger.Info(fmt.Sprintf("Warning: restoration hash is calculated from spec, custom field with key %s will be ignored", config.GetOperatorConfig().NetboxRestorationHashFieldName))
	}

	// Copy customFields from claim and add restoration hash
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}

	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generatePrefixRestorationHash(claim)

	return netboxv1.PrefixSpec{
		Prefix:           prefix,
		Tenant:           claim.Spec.Tenant,
		Site:             claim.Spec.Site,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generatePrefixRestorationHash(claim *netboxv1.PrefixClaim) string {
	parentPrefixSelectorStr := ""
	if len(claim.Spec.ParentPrefixSelector) > 0 {
		// we generate the string by
		// a) sort all keys in non-decreasing order (to avoid reordering the field in the CR causing a different hash to be generated)
		// b) concat all the keys and values in the sequence of key1_value1_..._keyN_valueN

		keyList := make([]string, 0, len(claim.Spec.ParentPrefixSelector))
		for key := range claim.Spec.ParentPrefixSelector {
			keyList = append(keyList, key)
		}
		sort.Strings(keyList)

		for _, key := range keyList {
			if len(parentPrefixSelectorStr) > 0 {
				parentPrefixSelectorStr += "_"
			}
			parentPrefixSelectorStr += key + "_" + claim.Spec.ParentPrefixSelector[key]
		}
	}

	rd := PrefixClaimRestorationData{
		Namespace:            claim.Namespace,
		Name:                 claim.Name,
		ParentPrefix:         claim.Spec.ParentPrefix,
		PrefixLength:         claim.Spec.PrefixLength,
		Tenant:               claim.Spec.Tenant,
		ParentPrefixSelector: parentPrefixSelectorStr,
	}

	return rd.ComputeHash()
}

type PrefixClaimRestorationData struct {
	// only use immutable fields
	Namespace            string
	Name                 string
	ParentPrefix         string
	PrefixLength         string
	Tenant               string
	ParentPrefixSelector string
}

func (rd *PrefixClaimRestorationData) ComputeHash() string {
	if rd == nil {
		return ""
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+rd.Name+rd.ParentPrefix+rd.PrefixLength+rd.Tenant+rd.ParentPrefixSelector)))
}
