// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import "github.com/krenalis/krenalis/core/internal/state"

// Satisfies reports whether the given attributes satisfy the required consent
// purposes. If matchAll is true, the attributes must satisfy every required
// purpose; otherwise, satisfying at least one is enough.
func Satisfies(purposes []*state.ConsentPurpose, matchAll bool, attributes map[string]any) bool {
	if len(purposes) == 0 {
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
	for _, purpose := range purposes {
		granted, _ := consents[purpose.Code].(bool)
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
