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

package api

import (
	"regexp"
	"strings"
)

var slugInvalidCharsRegex = regexp.MustCompile(`[^-a-zA-Z0-9_]+`)

// slugify converts name into a value that satisfies NetBox's slug field
// constraints (matches ^[-a-zA-Z0-9_]+$, max 100 characters). If name yields
// an empty slug, fallback is returned instead.
func slugify(name, fallback string) string {
	slug := slugInvalidCharsRegex.ReplaceAllString(strings.TrimSpace(name), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 100 {
		slug = strings.Trim(slug[:100], "-")
	}
	if slug == "" {
		slug = fallback
	}
	return slug
}
