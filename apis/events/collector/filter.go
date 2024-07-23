//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package collector

import (
	"fmt"
	"strings"

	"github.com/meergo/meergo/apis/state"
)

// filterApplies reports whether the filter applies to the provided properties.
// Returns an error if one of the properties of the filter is not found in the
// properties map.
func filterApplies(filter *state.Filter, properties map[string]any) (bool, error) {
	if filter == nil {
		return true, nil
	}
	for _, cond := range filter.Conditions {
		value, ok := readPropertyFrom(properties, cond.Property)
		if !ok {
			return false, fmt.Errorf("property %q not found", cond.Property)
		}
		var conditionApplies bool
		switch cond.Operator {
		case "is":
			conditionApplies = value == cond.Value
		case "is not":
			conditionApplies = value != cond.Value
		}
		if conditionApplies && filter.Logical == "any" {
			return true, nil
		}
		if !conditionApplies && filter.Logical == "all" {
			return false, nil
		}
	}
	if filter.Logical == "any" {
		return false, nil // none of the conditions applied.
	}
	// All the conditions applied.
	return true, nil
}

// readPropertyFrom retrieves the value at the specified path from the map m.
// It returns the value if found, otherwise nil, and a boolean indicating
// whether the path exists in m.
func readPropertyFrom(m map[string]any, path string) (any, bool) {
	var name string
	for {
		name, path, _ = strings.Cut(path, ".")
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if path == "" {
			return v, true
		}
		m, ok = v.(map[string]any)
		if !ok {
			return nil, false
		}
	}
}
