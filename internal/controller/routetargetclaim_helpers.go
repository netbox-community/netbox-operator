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

func generateRouteTargetFromRouteTargetClaim(ctx context.Context, claim *netboxv1.RouteTargetClaim, routeTarget string) *netboxv1.RouteTarget {
	logger := log.FromContext(ctx)
	routeTargetResource := &netboxv1.RouteTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.Namespace,
		},
		Spec: generateRouteTargetSpec(claim, routeTarget, logger),
	}
	return routeTargetResource
}

func generateRouteTargetSpec(claim *netboxv1.RouteTargetClaim, routeTarget string, logger logr.Logger) netboxv1.RouteTargetSpec {
	// log a warning if the netboxOperatorRestorationHash name is a key in the customFields map of the RouteTargetClaim
	_, ok := claim.Spec.CustomFields[config.GetOperatorConfig().NetboxRestorationHashFieldName]
	if ok {
		logger.Info(fmt.Sprintf("Warning: restoration hash is calculated from spec, custom field with key %s will be ignored", config.GetOperatorConfig().NetboxRestorationHashFieldName))
	}

	// Copy customFields from claim and add restoration hash
	customFields := make(map[string]string, len(claim.Spec.CustomFields)+1)
	for k, v := range claim.Spec.CustomFields {
		customFields[k] = v
	}

	customFields[config.GetOperatorConfig().NetboxRestorationHashFieldName] = generateRouteTargetRestorationHash(claim)

	return netboxv1.RouteTargetSpec{
		RouteTarget:      routeTarget,
		Tenant:           claim.Spec.Tenant,
		CustomFields:     customFields,
		Description:      claim.Spec.Description,
		Comments:         claim.Spec.Comments,
		PreserveInNetbox: claim.Spec.PreserveInNetbox,
	}
}

func generateRouteTargetRestorationHash(claim *netboxv1.RouteTargetClaim) string {
	rd := RouteTargetClaimRestorationData{
		Namespace:                claim.Namespace,
		Name:                     claim.Name,
		Asn:                      fmt.Sprintf("%d", claim.Spec.Asn),
		Tenant:                   claim.Spec.Tenant,
		AssignedNumberRangeStart: fmt.Sprintf("%d", claim.Spec.AssignedNumberRangeStart),
		AssignedNumberRangeEnd:   fmt.Sprintf("%d", claim.Spec.AssignedNumberRangeEnd),
	}
	return rd.ComputeHash()
}

type RouteTargetClaimRestorationData struct {
	// only use immutable fields
	Namespace                string
	Name                     string
	Asn                      string
	Tenant                   string
	AssignedNumberRangeStart string
	AssignedNumberRangeEnd   string
}

func (rd *RouteTargetClaimRestorationData) ComputeHash() string {
	if rd == nil {
		return ""
	}
	// fields are separated so that e.g. ASN 6500 with range start 11 can't
	// collide with ASN 65001 with range start 1
	return fmt.Sprintf("%x", sha1.Sum([]byte(rd.Namespace+"/"+rd.Name+"/"+rd.Asn+"/"+rd.Tenant+"/"+rd.AssignedNumberRangeStart+"/"+rd.AssignedNumberRangeEnd)))
}

// routeTargetLockName returns the lease lock name serializing route target
// allocation across all RouteTargetClaims of the same ASN. Route targets of
// different ASNs can never collide, but ranges from different claims of the
// same ASN may overlap, so the lock is keyed by ASN rather than by range.
func routeTargetLockName(asn int64) string {
	return fmt.Sprintf("route-target-asn-%d", asn)
}
