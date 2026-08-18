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
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateVlanFromVlanClaim(ctx context.Context, claim *netboxv1.VlanClaim, vid int32) *netboxv1.Vlan {
	logger := log.FromContext(ctx)
	vlanResource := &netboxv1.Vlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.Namespace,
		},
		Spec: generateVlanSpec(claim, vid, logger),
	}
	return vlanResource
}

func generateVlanSpec(claim *netboxv1.VlanClaim, vid int32, logger logr.Logger) netboxv1.VlanSpec {
	// log a warning if the netboxOperatorRestorationHash name is a key in the customFields map of the VlanClaim
	_, ok := claim.Spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName]
	if ok {
		logger.Info(fmt.Sprintf("Warning: restoration hash is calculated from spec, custom field with key %s will be ignored", config.GetOperatorConfig().NetboxRestorationHashFieldName))
	}

	// Copy customFields from claim and add restoration hash
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}

	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateVlanRestorationHash(claim)

	return netboxv1.VlanSpec{
		Name:             claim.Name,
		Vid:              vid,
		Site:             claim.Spec.Site,
		Tenant:           claim.Spec.Tenant,
		Status:           claim.Spec.Status,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generateVlanRestorationHash(claim *netboxv1.VlanClaim) string {
	rd := VlanClaimRestorationData{
		Namespace:     claim.Namespace,
		Name:          claim.Name,
		Site:          claim.Spec.Site,
		Tenant:        claim.Spec.Tenant,
		Vid:           fmt.Sprintf("%d", claim.Spec.Vid),
		VidRangeStart: fmt.Sprintf("%d", claim.Spec.VidRangeStart),
		VidRangeEnd:   fmt.Sprintf("%d", claim.Spec.VidRangeEnd),
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+rd.Name+rd.Site+rd.Tenant+rd.Vid+rd.VidRangeStart+rd.VidRangeEnd)))
}

type VlanClaimRestorationData struct {
	// only use immutable fields
	Namespace     string
	Name          string
	Site          string
	Tenant        string
	Vid           string
	VidRangeStart string
	VidRangeEnd   string
}

var leaseLockNameInvalidCharsRegex = regexp.MustCompile(`[^a-z0-9-]+`)

// convertVlanRangeToLeaseLockName builds a lease lock name identifying the
// shared VID range a range-based VlanClaim draws from, so that concurrent
// claims against the same site+range serialize their allocation. Unlike
// L2VPN's "type" (a controlled enum), a NetBox Site name is free text, so it
// is sanitized here to satisfy the Lease resource's DNS-1123 name
// requirements.
func convertVlanRangeToLeaseLockName(site string, start int32, end int32) string {
	sanitizedSite := leaseLockNameInvalidCharsRegex.ReplaceAllString(strings.ToLower(strings.TrimSpace(site)), "-")
	sanitizedSite = strings.Trim(sanitizedSite, "-")
	if sanitizedSite == "" {
		sanitizedSite = "global"
	}
	return fmt.Sprintf("vlan-%s-%d-%d", sanitizedSite, start, end)
}
