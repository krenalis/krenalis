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

// filterApplies reports whether the filter applies to props, which can be an
// event or a user. Returns error if one of the properties of the filter are not
// found within props.
func filterApplies(filter *state.Filter, props map[string]any) (bool, error) {
	if filter == nil {
		return true, nil
	}
	for _, cond := range filter.Conditions {
		value, ok := readPropertyFrom(props, cond.Property)
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

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
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
