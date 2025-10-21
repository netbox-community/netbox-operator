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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/netbox-community/go-netbox/v3/netbox/client/extras"
	"github.com/netbox-community/go-netbox/v3/netbox/client/ipam"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
)

var (
	// TODO(henrybear327): centralize errors
	ErrNoPrefixMatchsSizeCriteria = errors.New("no available prefix matches size criteria")
)

func (r *NetboxClient) RestoreExistingPrefixByHash(hash string, requestedPrefixLength string) (*models.Prefix, error) {
	customPrefixSearch := newQueryFilterOperation(nil, []CustomFieldEntry{
		{
			key:   config.GetOperatorConfig().NetboxRestorationHashFieldName,
			value: hash,
		},
	})
	list, err := r.Ipam.IpamPrefixesList(ipam.NewIpamPrefixesListParams(), nil, customPrefixSearch)
	if err != nil {
		return nil, err
	}

	// TODO: find a better way?
	if list.Payload.Count != nil && *list.Payload.Count == 0 {
		return nil, nil
	}

	// Filter for exact prefix length
	prefixesWithExactPrefixLength := make([]*models.Prefix, 0)
	for _, prefix := range list.Payload.Results {
		if strings.Contains(*prefix.Prefix, requestedPrefixLength) {
			prefixesWithExactPrefixLength = append(prefixesWithExactPrefixLength, &models.Prefix{
				Prefix: *prefix.Prefix,
			})
		}
	}

	// We should not have more than 1 result...
	if len(prefixesWithExactPrefixLength) > 1 {
		return nil, fmt.Errorf("too many restoration results found in NetBox for hash %s and prefix length %s, number of results: %v", hash, requestedPrefixLength, len(prefixesWithExactPrefixLength))
	} else if len(prefixesWithExactPrefixLength) == 0 {
		return nil, fmt.Errorf("no prefix found in NetBox with restoration hash %s and prefix length %s", hash, requestedPrefixLength)
	}
	res := prefixesWithExactPrefixLength[0]
	if res.Prefix == "" {
		return nil, errors.New("prefix in netbox is empty")
	}

	return &models.Prefix{
		Prefix: res.Prefix,
	}, nil
}

func validatePrefixLengthOrError(prefixClaim *models.PrefixClaim, prefixFamily int64) error {
	parentPrefixSplit := strings.Split(prefixClaim.ParentPrefix, "/")
	if len(parentPrefixSplit) != 2 {
		return errors.New("invalid parent prefix format")
	}

	parentPrefixLength, err := strconv.Atoi(parentPrefixSplit[1])
	if err != nil {
		return err
	}

	requestedPrefixLength, err := strconv.Atoi(strings.TrimPrefix(prefixClaim.PrefixLength, "/"))
	if err != nil {
		return err
	}

	if parentPrefixLength == requestedPrefixLength {
		return errors.New("requesting the entire parent prefix range is disallowed")
	} else if parentPrefixLength > requestedPrefixLength {
		return errors.New("requested prefix size must be smaller than the parent prefix size")
	} else if prefixFamily == int64(IPv4Family) && requestedPrefixLength > 32 {
		return errors.New("requested prefix length must be smaller than 32 for IPv4")
	}

	return nil
}

func (r *NetboxClient) GetAvailablePrefixesByParentPrefixSelector(prefixClaimSpec *netboxv1.PrefixClaimSpec) ([]*models.Prefix, error) {
	fieldEntries := make(map[string]string)

	if tenant, ok := prefixClaimSpec.ParentPrefixSelector["tenant"]; ok {
		details, err := r.GetTenantDetails(tenant)
		if err != nil {
			return nil, err
		}

		fieldEntries["tenant_id"] = strconv.Itoa(int(details.Id))
	}

	if site, ok := prefixClaimSpec.ParentPrefixSelector["site"]; ok {
		details, err := r.GetSiteDetails(site)
		if err != nil {
			return nil, err
		}

		fieldEntries["site_id"] = strconv.Itoa(int(details.Id))
	}

	if family, ok := prefixClaimSpec.ParentPrefixSelector["family"]; ok {
		switch family {
		case "IPv4":
			family = "4"
		case "IPv6":
			family = "6"
		default:
			return nil, ErrInvalidIpFamily
		}
		fieldEntries["family"] = family
	}

	parentPrefixSelectorCustomFields := make([]CustomFieldEntry, 0, len(prefixClaimSpec.ParentPrefixSelector))
	for k, v := range prefixClaimSpec.ParentPrefixSelector {
		switch k {
		case "tenant", "site", "family":
			// skip built in fields
		default:
			parentPrefixSelectorCustomFields = append(parentPrefixSelectorCustomFields, CustomFieldEntry{
				key:   k,
				value: v,
			})
		}
	}

	err := r.customFieldsExistsOrErr(parentPrefixSelectorCustomFields)
	if err != nil {
		return nil, err
	}

	conditions := newQueryFilterOperation(fieldEntries, parentPrefixSelectorCustomFields)

	list, err := r.Ipam.IpamPrefixesList(ipam.NewIpamPrefixesListParams(), nil, conditions)
	if err != nil {
		return nil, err
	}

	// TODO: find a better way?
	if list.Payload.Count != nil && *list.Payload.Count == 0 {
		return nil, nil
	}

	prefixes := make([]*models.Prefix, 0)
	for _, prefix := range list.Payload.Results {
		if prefix.Prefix != nil && r.isParentPrefixCandidate(prefixClaimSpec, *prefix.Prefix) {
			prefixes = append(prefixes, &models.Prefix{
				Prefix: *prefix.Prefix,
			})
		}
	}

	return prefixes, nil
}

func (r *NetboxClient) customFieldsExistsOrErr(customfieldFilterEntries []CustomFieldEntry) error {
	if len(customfieldFilterEntries) == 0 {
		// as the parent prefix selector does not filter for custom fields
		// the check can be skipped
		return nil
	}

	responseGetCustomFieldsList, err := r.Extras.ExtrasCustomFieldsList(extras.NewExtrasCustomFieldsListParams(), nil)
	if err != nil {
		return err
	}

	existingCustomFields := responseGetCustomFieldsList.Payload.Results
	if len(existingCustomFields) == 0 {
		return fmt.Errorf("netbox custom fields list is nil or empty")
	}

	customFieldNames := make([]string, len(existingCustomFields))
	for i, field := range existingCustomFields {
		if field.Name == nil {
			return fmt.Errorf("netbox custom field name is nil")
		}
		customFieldNames[i] = *field.Name
	}

	missingCustomFields := make([]string, 0)
	for _, entry := range customfieldFilterEntries {
		if !slices.Contains(customFieldNames, entry.key) {
			missingCustomFields = append(missingCustomFields, entry.key)
		}
	}

	if len(missingCustomFields) > 0 {
		return fmt.Errorf(
			"invalid parentPrefixSelector, netbox custom fields %s do not exist",
			strings.Join(missingCustomFields, ", "),
		)
	}

	return nil
}

func (r *NetboxClient) isParentPrefixCandidate(prefixClaimSpec *netboxv1.PrefixClaimSpec, prefix string) bool {
	// if we can allocate a prefix from it, we can take it as a parent prefix
	if _, err := r.GetAvailablePrefixByClaim(&models.PrefixClaim{
		ParentPrefix: prefix,
		PrefixLength: prefixClaimSpec.PrefixLength,
		Metadata: &models.NetboxMetadata{
			Tenant: prefixClaimSpec.Tenant,
			Site:   prefixClaimSpec.Site,
		},
	}); err == nil {
		return true
	}
	return false
}

// GetAvailablePrefixByClaim searches an available Prefix in Netbox matching PrefixClaim requirements
func (r *NetboxClient) GetAvailablePrefixByClaim(prefixClaim *models.PrefixClaim) (*models.Prefix, error) {
	_, err := r.GetTenantDetails(prefixClaim.Metadata.Tenant)
	if err != nil {
		return nil, err
	}

	// Don't assign an prefix if the requested site doesn't exist in netbox
	if prefixClaim.Metadata.Site != "" {
		_, err := r.GetSiteDetails(prefixClaim.Metadata.Site)
		if err != nil {
			return nil, err
		}
	}

	responseParentPrefix, err := r.GetPrefix(&models.Prefix{
		Prefix:   prefixClaim.ParentPrefix,
		Metadata: prefixClaim.Metadata,
	})
	if err != nil {
		return nil, err
	}
	if len(responseParentPrefix.Payload.Results) == 0 {
		return nil, ErrParentPrefixNotFound
	}

	if err := validatePrefixLengthOrError(prefixClaim, *responseParentPrefix.Payload.Results[0].Family.Value); err != nil {
		return nil, err
	}

	parentPrefixId := responseParentPrefix.Payload.Results[0].ID

	/* Notes regarding the available prefix returned by netbox

	The available prefixes API do NOT allow us to pass in the desired prefix size. And we observed the API's behavior as the following:
	- If the parent prefix currently doesn't have any child prefix associated with it, the available prefix API will return the parent prefix itself (as we disallow requesting for the entire parent prefix, this is the special case we need to handle).
	- If there is a child prefix (e.g. prefix of /28) associated with the parent prefix, the available prefixes API will likely return options containing /25 /26 /27 /28.
	- If there are multiple child prefixes associated with the parent prefix, the available prefixes API will likely suggest options containing prefix sizes all the way up to the smallest prefix size

	Thus, in some cases, we need to call CreateAvailablePrefixesByParentPrefix to explicitly request to the desired prefix size, e.g. when we have a never-used parent prefix.
	*/

	// step 1: we get available prefixes of the parent prefix from NetBox
	responseAvailablePrefixes, err := r.GetAvailablePrefixesByParentPrefix(parentPrefixId)
	if err != nil {
		return nil, err
	}

	// step 2: we get the prefix that has the prefix size >= the requested size
	matchingPrefix, IsMatchingPrefixSizeAsDesired, err := getSmallestMatchingPrefix(responseAvailablePrefixes, prefixClaim.PrefixLength)
	if err != nil {
		return nil, err
	}

	if !IsMatchingPrefixSizeAsDesired {
		// step 3-1: if the matchingPrefix size != the requested size, we will take the smallest matching prefix a.b.c.d/x, and modify it into a.b.c.d/target_size
		// Do NOT call CreateAvailablePrefixesByParentPrefix, as netbox uses a first-fit algorithm, where we would like to use the best-fit algorithm.
		// Also, we are using netbox client v3.4.5, which requires a hack for the call to work [1]. The hack was implemented here [2].
		// Reference:
		// [1] https://github.com/netbox-community/go-netbox/issues/131
		// [2] https://github.com/netbox-community/go-netbox/v3/commits/hack/v3.4.5-0/
		matchingPrefixSplit := strings.Split(matchingPrefix, "/")
		if len(matchingPrefixSplit) != 2 {
			return nil, ErrWrongMatchingPrefixSubnetFormat
		}
		matchingPrefix = matchingPrefixSplit[0] + prefixClaim.PrefixLength
	} // else {
	// step 3-2: if the matchingPrefix size == the requested size -> we return this one, thus no-op
	// }

	return &models.Prefix{
		Prefix: matchingPrefix,
	}, nil
}

func (r *NetboxClient) GetAvailablePrefixesByParentPrefix(parentPrefixId int64) (*ipam.IpamPrefixesAvailablePrefixesListOK, error) {
	requestAvailablePrefixes := ipam.NewIpamPrefixesAvailablePrefixesListParams().WithID(parentPrefixId)
	responseAvailablePrefixes, err := r.Ipam.IpamPrefixesAvailablePrefixesList(requestAvailablePrefixes, nil)
	if err != nil {
		return nil, err
	}
	if len(responseAvailablePrefixes.Payload) == 0 {
		return nil, ErrParentPrefixExhausted
	}
	return responseAvailablePrefixes, nil
}

func getSmallestMatchingPrefix(prefixList *ipam.IpamPrefixesAvailablePrefixesListOK, prefixClaimLengthString string) (string, bool, error) {
	// input validation
	if len(prefixClaimLengthString) == 0 {
		return "", false, errors.New("invalid prefixClaimLengthString: empty string")
	}

	if !strings.Contains(prefixClaimLengthString, "/") {
		return "", false, errors.New("invalid prefixClaimLengthString: no subnet given (no slash in the string)")
	}

	prefixClaimLength, err := strconv.Atoi(strings.TrimPrefix(prefixClaimLengthString, "/"))
	if err != nil {
		return "", false, err
	}

	candidateIdx := -1
	candidatePrefixSize := 0
	for i, prefix := range prefixList.Payload {
		_, sizeString, found := strings.Cut(prefix.Prefix, "/")
		if !found {
			// TODO: log error
			continue
		}
		prefixSize, err := strconv.Atoi(sizeString)
		if err != nil {
			// TODO: log error
			continue
		}
		if prefixSize == prefixClaimLength {
			return prefix.Prefix, true, nil
		}
		if candidatePrefixSize < prefixSize && prefixSize <= prefixClaimLength {
			candidateIdx = i
			candidatePrefixSize = prefixSize
		}
	}

	if candidateIdx == -1 {
		return "", false, fmt.Errorf("%w", ErrNoPrefixMatchsSizeCriteria)
	}

	return prefixList.Payload[candidateIdx].Prefix, false, nil
}
