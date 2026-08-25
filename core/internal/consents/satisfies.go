// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import "github.com/krenalis/krenalis/core/internal/state"

// Satisfies reports whether the given consents satisfy the required consent
// purposes. If matchAll is true, the given consents must satisfy every required
// purpose; otherwise, satisfying at least one is enough.
func Satisfies(purposes []*state.ConsentPurpose, matchAll bool, given map[string]any) bool {
	if len(purposes) == 0 {
		return true
	}
	for _, purpose := range purposes {
		granted, _ := given[purpose.Code].(bool)
		if granted {
			if !matchAll {
				return true
			}
		} else if matchAll {
			return false
		}
	}
	return matchAll
}

// SatisfiesEvent reports whether the consents carried by the given event
// satisfy the required consent purposes. The consents are read from the context
// of the event, keyed by the code of the purpose.
func SatisfiesEvent(purposes []*state.ConsentPurpose, matchAll bool, event map[string]any) bool {
	if len(purposes) == 0 {
		return true
	}
	context, ok := event["context"].(map[string]any)
	if !ok {
		return false
	}
	given, ok := context["consents"].(map[string]any)
	if !ok {
		return false
	}
	return Satisfies(purposes, matchAll, given)
}

// SatisfiesProfile reports whether the consents carried by the given profile
// satisfy the required consent purposes.
//
// TODO(Andrea): the property holding the consents of a purpose will be given by
// the path configured on the purpose itself, which does not exist yet. Until
// then, the consents are assumed to be in a "consents" property of the profile,
// keyed by the code of the purpose.
func SatisfiesProfile(purposes []*state.ConsentPurpose, matchAll bool, profile map[string]any) bool {
	if len(purposes) == 0 {
		return true
	}
	given, ok := profile["consents"].(map[string]any)
	if !ok {
		return false
	}
	return Satisfies(purposes, matchAll, given)
}
