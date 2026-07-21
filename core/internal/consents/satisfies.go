// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import "github.com/krenalis/krenalis/core/internal/state"

// Satisfies reports whether the given attributes satisfy the required consents.
// If matchAll is true, the attributes must satisfy every required consent;
// otherwise, satisfying at least one is enough.
func Satisfies(requiredConsents []string, matchAll bool, attributes map[string]any) bool {
	if len(requiredConsents) == 0 {
		return true
	}
	context, ok := attributes["context"].(map[string]any)
	if !ok {
		return false
	}
	consents, ok := context["consents"].(map[string]any)
	if !ok {
		return false
	}
	for _, code := range requiredConsents {
		granted, _ := consents[code].(bool)
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

// SatisfiesByIDs reports whether the given attributes satisfy the required
// consents, given the consent's IDs. If matchAll is true, the attributes must
// satisfy every required consent; otherwise, satisfying at least one is enough.
func SatisfiesByIDs(ws *state.Workspace, requiredConsentIDs []string, matchAll bool, attributes map[string]any) bool {
	if len(requiredConsentIDs) == 0 {
		return true
	}
	codes := make([]string, len(requiredConsentIDs))
	for i, id := range requiredConsentIDs {
		cp, ok := ws.ConsentPurpose(id)
		if !ok {
			return false
		}
		codes[i] = cp.Code
	}
	return Satisfies(codes, matchAll, attributes)
}
