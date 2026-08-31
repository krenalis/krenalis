// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import (
	"github.com/krenalis/krenalis/core/internal/properties"
	"github.com/krenalis/krenalis/core/internal/state"
)

// SatisfiesEvent reports whether the consents carried by the given event
// satisfy the required consent purposes.
func SatisfiesEvent(purposes []*state.ConsentPurpose, matchAll bool, event map[string]any) bool {
	return satisfies(purposes, matchAll, func(purpose *state.ConsentPurpose) bool {
		// The consent can be given with the code of the purpose or with any of
		// its aliases, so every property path resolved for the purpose is read
		// until one of them grants the consent.
		for _, path := range purpose.EventPropertyPaths() {
			if granted(event, path) {
				return true
			}
		}
		return false
	})
}

// SatisfiesProfile reports whether the consents carried by the given profile
// satisfy the required consent purposes.
func SatisfiesProfile(purposes []*state.ConsentPurpose, matchAll bool, profile map[string]any) bool {
	return satisfies(purposes, matchAll, func(purpose *state.ConsentPurpose) bool {
		return granted(profile, purpose.ProfilePropertyPath())
	})
}

// satisfies reports whether the required consent purposes are satisfied, given
// that grants reports whether the consent for a purpose is given. If matchAll
// is true, the consent must be given for every required purpose; otherwise,
// one purpose is enough.
func satisfies(purposes []*state.ConsentPurpose, matchAll bool, grants func(*state.ConsentPurpose) bool) bool {
	if len(purposes) == 0 {
		return true
	}
	for _, purpose := range purposes {
		if grants(purpose) {
			if !matchAll {
				return true
			}
		} else if matchAll {
			return false
		}
	}
	return matchAll
}

// granted reports whether the property of the given attributes with the given
// path holds a granted consent.
func granted(attributes map[string]any, path []string) bool {
	v, _ := properties.Read(attributes, path)
	b, _ := v.(bool)
	return b
}
