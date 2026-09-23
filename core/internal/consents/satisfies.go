// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import (
	"github.com/krenalis/krenalis/core/internal/properties"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
)

// SatisfiesEvent reports whether the consents carried by the given event
// satisfy the required consent purposes.
func SatisfiesEvent(purposes []*state.ConsentPurpose, matchAll bool, event map[string]any) bool {
	return satisfies(purposes, matchAll, func(purpose *state.ConsentPurpose) bool {
		// Only missing keys are skipped; the first present key decides the consent.
		for _, path := range purpose.EventPropertyPaths() {
			value, exists := properties.Read(event, path)
			if !exists {
				continue
			}
			switch value := value.(type) {
			case bool:
				return value
			case json.Value:
				return value.Bool()
			}
			return false
		}
		return false
	})
}

// SatisfiesProfile reports whether the consents carried by the given profile
// satisfy the required consent purposes.
func SatisfiesProfile(purposes []*state.ConsentPurpose, matchAll bool, profile map[string]any) bool {
	return satisfies(purposes, matchAll, func(purpose *state.ConsentPurpose) bool {
		location := purpose.ProfileConsentLocation
		if location == nil {
			return false
		}
		return granted(profile, purpose.ProfilePropertyPath(), location.JSONKey)
	})
}

// granted reports whether the schema property or its explicit JSON key grants
// consent.
func granted(attributes map[string]any, path []string, key string) bool {

	if len(path) == 0 || properties.InJSON(attributes, path) {
		return false
	}
	v, ok := properties.Read(attributes, path)
	if !ok {
		return false
	}

	if key == "" {
		value, ok := v.(bool)
		return ok && value
	}
	object, ok := v.(json.Value)
	if !ok || !object.IsObject() {
		return false
	}
	value, exists := object.Get([]string{key})

	return exists && value.Bool()
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
