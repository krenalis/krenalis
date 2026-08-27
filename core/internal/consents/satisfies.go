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
	return satisfies(purposes, matchAll, event, func(purpose *state.ConsentPurpose) []string {
		return purpose.EventPropertyPath()
	})
}

// SatisfiesProfile reports whether the consents carried by the given profile
// satisfy the required consent purposes.
func SatisfiesProfile(purposes []*state.ConsentPurpose, matchAll bool, profile map[string]any) bool {
	return satisfies(purposes, matchAll, profile, func(purpose *state.ConsentPurpose) []string {
		return purpose.ProfilePropertyPath()
	})
}

// satisfies reports whether the consents carried by the given attributes
// satisfy the required consent purposes. The consent given for a purpose is
// read from the property of the attributes whose path is returned by property.
// If matchAll is true, the given consents must satisfy every required purpose;
// otherwise, satisfying at least one is enough.
func satisfies(purposes []*state.ConsentPurpose, matchAll bool, attributes map[string]any,
	property func(*state.ConsentPurpose) []string) bool {
	if len(purposes) == 0 {
		return true
	}
	for _, purpose := range purposes {
		v, _ := properties.Read(attributes, property(purpose))
		granted, _ := v.(bool)
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
