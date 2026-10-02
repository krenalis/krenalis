// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import (
	"strings"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
)

// SatisfiesEvent reports whether the given event satisfies the required
// consent purposes. Each purpose should have at least one event consent
// location, and each location's purpose code should be non-empty.
func SatisfiesEvent(op state.ConsentPurposesOperator, purposes []*state.ConsentPurpose, event map[string]any) bool {

	if len(purposes) == 0 {
		return true
	}

	context, ok := event["context"].(map[string]any)
	if !ok {
		return false
	}
	consents, ok := context["consents"].(map[string]any)
	if !ok {
		return false
	}

	for _, purpose := range purposes {
		var granted bool
		// Only missing locations are skipped; the first location found determines the consent.
		for _, loc := range purpose.EventConsentLocations {
			value, exists := consents[loc.PurposeCode]
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
		if op == state.PurposesAnd {
			if !granted {
				return false
			}
		} else if op == state.PurposesOr && granted {
			return true
		}
	}

	return op == state.PurposesAnd
}

// SatisfiesProfile reports whether the given profile attributes satisfy the
// required consent purposes. Each purpose must have a profile consent location.
func SatisfiesProfile(op state.ConsentPurposesOperator, purposes []*state.ConsentPurpose, attributes map[string]any) bool {

	if len(purposes) == 0 {
		return true
	}

	for _, purpose := range purposes {
		var granted bool
		loc := purpose.ProfileConsentLocation
		var value any
		var found bool
		path := loc.Property
		current := attributes
		// Traverse nested maps only; JSON values require an explicit key.
		for {
			name, rest, hasMore := strings.Cut(path, ".")
			value, found = current[name]
			if !found || !hasMore {
				break
			}
			next, ok := value.(map[string]any)
			if !ok {
				found = false
				break
			}
			current = next
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
		if op == state.PurposesAnd {
			if !granted {
				return false
			}
		} else if op == state.PurposesOr && granted {
			return true
		}
	}

	return op == state.PurposesAnd
}
