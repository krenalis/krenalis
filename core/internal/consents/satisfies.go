// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

// Satisfies reports whether the given attributes satisfy the consent purposes
// with the given codes. If matchAll is true, the attributes must satisfy every
// purpose; otherwise, satisfying at least one is enough.
func Satisfies(codes []string, matchAll bool, attributes map[string]any) bool {
	if len(codes) == 0 {
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
	for _, code := range codes {
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
