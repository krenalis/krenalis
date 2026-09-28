// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import (
	"strings"

	"github.com/krenalis/krenalis/core/internal/properties"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
)

// SatisfiesEvent reports whether the given event satisfies the required
// consent purposes.
func SatisfiesEvent(purposes []*state.ConsentPurpose, matchAll bool, event map[string]any) bool {

	if len(purposes) == 0 {
		return true
	}

	for _, purpose := range purposes {
		var granted bool
		// Only missing locations are skipped; the first location found determines the consent.
		for _, loc := range purpose.EventConsentLocations {
			value, exists := properties.Read(event, []string{"context", "consents", loc.PurposeCode})
			if !exists {
				continue
			}
			switch value := value.(type) {
			case bool:
				granted = value
			case json.Value:
				granted = value.Bool()
			}
			break
		}
		if matchAll {
			if !granted {
				return false
			}
		} else if granted {
			return true
		}
	}

	return matchAll
}

// SatisfiesProfile reports whether the given profile satisfies the required
// consent purposes.
func SatisfiesProfile(purposes []*state.ConsentPurpose, matchAll bool, profile map[string]any) bool {

	if len(purposes) == 0 {
		return true
	}

	for _, purpose := range purposes {
		var granted bool
		loc := purpose.ProfileConsentLocation
		if loc != nil {
			var value any
			var found bool
			path := loc.Property
			attributes := profile
			// Traverse nested maps only; JSON values require an explicit key.
			for {
				name, rest, hasMore := strings.Cut(path, ".")
				value, found = attributes[name]
				if !found || !hasMore {
					break
				}
				next, ok := value.(map[string]any)
				if !ok {
					found = false
					break
				}
				attributes = next
				path = rest
			}
			if found {
				if loc.JSONKey == "" {
					granted, _ = value.(bool)
				} else if object, ok := value.(json.Value); ok && object.IsObject() {
					v, exists := object.Get([]string{loc.JSONKey})
					granted = exists && v.Bool()
				}
			}
		}
		if matchAll {
			if !granted {
				return false
			}
		} else if granted {
			return true
		}
	}

	return matchAll
}
