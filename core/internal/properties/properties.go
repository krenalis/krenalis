// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package properties

import (
	"github.com/krenalis/krenalis/tools/json"
)

// Read reads the property with the given path from m, returning its value (if
// found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func Read(m map[string]any, path []string) (any, bool) {
	last := len(path) - 1
	for i, name := range path {
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if i == last {
			return v, true
		}
		switch v := v.(type) {
		case map[string]any:
			m = v
		case json.Value:
			if v, ok := v.Get(path[i+1:]); ok {
				return v, true
			}
			return nil, false
		default:
			return nil, false
		}
	}
	panic("unreachable code")
}
