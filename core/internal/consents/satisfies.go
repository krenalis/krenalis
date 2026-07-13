// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import "github.com/krenalis/krenalis/core/internal/state"

// Satisfies reports whether the given attributes satisfy the required consents.
func Satisfies(requiredConsents []string, attributes map[string]any) bool {
	if len(requiredConsents) == 0 {
		return true
	}
	context, ok := attributes["context"].(map[string]any)
	if !ok {
		return false
	}
	consent, ok := context["consent"].(map[string]any)
	if !ok {
		return false
	}
	for _, code := range requiredConsents {
		v, ok := consent[code].(bool)
		if !ok || !v {
			return false
		}
	}
	return true
}

// SatisfiesByIDs reports whether the given attributes satisfy the required
// consents, given the consent's IDs.
func SatisfiesByIDs(ws *state.Workspace, requiredConsentIDs []string, attributes map[string]any) bool {
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
	return Satisfies(codes, attributes)
}
