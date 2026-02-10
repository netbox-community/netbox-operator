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
	"context"

	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	netboxModels "github.com/netbox-community/go-netbox/v3/netbox/models"
	nclient "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
)

// NetboxCompositeClient holds both the legacy (v3) and modern (v4) clients,
// presenting a single unified interface to callers (controllers).
type NetboxCompositeClient struct {
	Legacy *NetboxClient
	Modern *NetboxClientV4
}

// NewNetboxCompositeClient creates a new composite client wrapping both v3 and v4 clients.
func NewNetboxCompositeClient(legacy *NetboxClient, modern *NetboxClientV4) *NetboxCompositeClient {
	return &NetboxCompositeClient{
		Legacy: legacy,
		Modern: modern,
	}
}

// VerifyNetboxConfiguration delegates to the legacy client.
func (c *NetboxCompositeClient) VerifyNetboxConfiguration() error {
	return c.Legacy.VerifyNetboxConfiguration()
}

// --- Prefix operations ---

func (c *NetboxCompositeClient) ReserveOrUpdatePrefix(ctx context.Context, prefix *models.Prefix) (*nclient.Prefix, error) {
	return c.Modern.ReserveOrUpdatePrefix(ctx, c.Legacy, prefix)
}

func (c *NetboxCompositeClient) GetPrefix(ctx context.Context, prefix *models.Prefix) (*nclient.PaginatedPrefixList, error) {
	return c.Modern.GetPrefix(ctx, prefix)
}

func (c *NetboxCompositeClient) DeletePrefix(ctx context.Context, prefixId int32) error {
	return c.Modern.DeletePrefix(ctx, prefixId)
}

// --- IP Range operations ---

func (c *NetboxCompositeClient) ReserveOrUpdateIpRange(ctx context.Context, ipRange *models.IpRange) (*nclient.IPRange, error) {
	return c.Modern.ReserveOrUpdateIpRange(ctx, c.Legacy, ipRange)
}

func (c *NetboxCompositeClient) DeleteIpRange(ctx context.Context, ipRangeId int64) error {
	return c.Modern.DeleteIpRange(ctx, ipRangeId)
}

// --- Operations still on v3 (not yet migrated) ---

func (c *NetboxCompositeClient) GetAvailableIpAddressByClaim(ctx context.Context, ipAddressClaim *models.IPAddressClaim) (*models.IPAddress, error) {
	return c.Legacy.GetAvailableIpAddressByClaim(ctx, c.Modern, ipAddressClaim)
}

func (c *NetboxCompositeClient) GetAvailableIpRangeByClaim(ctx context.Context, ipRangeClaim *models.IpRangeClaim) (*models.IpRange, error) {
	return c.Legacy.GetAvailableIpRangeByClaim(ctx, c.Modern, ipRangeClaim)
}

func (c *NetboxCompositeClient) GetAvailablePrefixByClaim(ctx context.Context, prefixClaim *models.PrefixClaim) (*models.Prefix, error) {
	return c.Legacy.GetAvailablePrefixByClaim(ctx, c.Modern, prefixClaim)
}

func (c *NetboxCompositeClient) GetAvailablePrefixesByParentPrefixSelector(ctx context.Context, prefixClaimSpec *netboxv1.PrefixClaimSpec) ([]*models.Prefix, error) {
	return c.Legacy.GetAvailablePrefixesByParentPrefixSelector(ctx, c.Modern, prefixClaimSpec)
}

func (c *NetboxCompositeClient) RestoreExistingIpByHash(hash string) (*models.IPAddress, error) {
	return c.Legacy.RestoreExistingIpByHash(hash)
}

func (c *NetboxCompositeClient) RestoreExistingIpRangeByHash(hash string) (*models.IpRange, error) {
	return c.Legacy.RestoreExistingIpRangeByHash(hash)
}

func (c *NetboxCompositeClient) RestoreExistingPrefixByHash(hash string, requestedPrefixLength string) (*models.Prefix, error) {
	return c.Legacy.RestoreExistingPrefixByHash(hash, requestedPrefixLength)
}

// --- IP Address operations (v3 only, not yet migrated) ---

func (c *NetboxCompositeClient) ReserveOrUpdateIpAddress(ipAddress *models.IPAddress) (*netboxModels.IPAddress, error) {
	return c.Legacy.ReserveOrUpdateIpAddress(ipAddress)
}

func (c *NetboxCompositeClient) DeleteIpAddress(ipAddressId int64) error {
	return c.Legacy.DeleteIpAddress(ipAddressId)
}

func (c *NetboxCompositeClient) GetAvailableIpAddressesByIpRange(ipRangeId int64) (*ipam.IpamIPRangesAvailableIpsListOK, error) {
	return c.Legacy.GetAvailableIpAddressesByIpRange(ipRangeId)
}
