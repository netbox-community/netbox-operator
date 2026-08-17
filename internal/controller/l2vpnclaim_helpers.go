/*
Copyright 2026 Swisscom (Schweiz) AG.

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

	"github.com/go-logr/logr"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateL2VPNFromL2VPNClaim(ctx context.Context, claim *netboxv1.L2VPNClaim, identifier int64) *netboxv1.L2VPN {
	logger := log.FromContext(ctx)
	l2vpnResource := &netboxv1.L2VPN{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.Namespace,
		},
		Spec: generateL2VPNSpec(claim, identifier, logger),
	}
	return l2vpnResource
}

func generateL2VPNSpec(claim *netboxv1.L2VPNClaim, identifier int64, logger logr.Logger) netboxv1.L2VPNSpec {
	// log a warning if the netboxOperatorRestorationHash name is a key in the customFields map of the L2VPNClaim
	_, ok := claim.Spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName]
	if ok {
		logger.Info(fmt.Sprintf("Warning: restoration hash is calculated from spec, custom field with key %s will be ignored", config.GetOperatorConfig().NetboxRestorationHashFieldName))
	}

	// Copy customFields from claim and add restoration hash
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}

	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateL2VPNRestorationHash(claim)

	return netboxv1.L2VPNSpec{
		Name:             claim.Name,
		Type:             claim.Spec.Type,
		Identifier:       identifier,
		Tenant:           claim.Spec.Tenant,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generateL2VPNRestorationHash(claim *netboxv1.L2VPNClaim) string {
	rd := L2VPNClaimRestorationData{
		Namespace:            claim.Namespace,
		Name:                 claim.Name,
		Type:                 claim.Spec.Type,
		Tenant:               claim.Spec.Tenant,
		Identifier:           fmt.Sprintf("%d", claim.Spec.Identifier),
		IdentifierRangeStart: fmt.Sprintf("%d", claim.Spec.IdentifierRangeStart),
		IdentifierRangeEnd:   fmt.Sprintf("%d", claim.Spec.IdentifierRangeEnd),
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+rd.Name+rd.Type+rd.Tenant+rd.Identifier+rd.IdentifierRangeStart+rd.IdentifierRangeEnd)))
}

type L2VPNClaimRestorationData struct {
	// only use immutable fields
	Namespace            string
	Name                 string
	Type                 string
	Tenant               string
	Identifier           string
	IdentifierRangeStart string
	IdentifierRangeEnd   string
}

// convertL2VPNRangeToLeaseLockName builds a lease lock name identifying the
// shared identifier range a range-based L2VPNClaim draws from, so that
// concurrent claims against the same range serialize their allocation.
func convertL2VPNRangeToLeaseLockName(type_ string, start int64, end int64) string {
	return fmt.Sprintf("l2vpn-%s-%d-%d", type_, start, end)
}
