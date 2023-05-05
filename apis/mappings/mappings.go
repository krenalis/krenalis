//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"chichi/apis/state"
	"chichi/apis/transformations"
	"chichi/apis/types"
)

// ActionFilterApplies reports whether the action filter applies to the event.
// Returns error if one of the properties of the filter are not found within the
// event.
func ActionFilterApplies(filter *state.ActionFilter, event map[string]any) (bool, error) {
	if filter == nil {
		return true, nil
	}
	for _, cond := range filter.Conditions {
		value, ok := readPropertyFrom(event, strings.Split(cond.Property, "."))
		if !ok {
			return false, fmt.Errorf("property %q not found in event", cond.Property)
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

// Apply applies the mapping or the transformation to the properties.
// If outSchema is a valid schema, the resulting properties are validated
// through that schema before being returned.
func Apply(ctx context.Context, action *state.Action, properties map[string]any, outSchema types.Type) (map[string]any, error) {

	var mappedEvent map[string]any

	// Map using properties mapping.
	if action.Mapping != nil {

		mappedEvent = map[string]any{}
		for out, in := range action.Mapping {
			inputPropPath := strings.Split(in, ".")
			value, ok := readPropertyFrom(properties, inputPropPath)
			if !ok {
				continue
			}
			// TODO(Gianluca): handle conversions of values here, when the type
			// checking rules will be defined.
			outputPropPath := strings.Split(out, ".")
			writePropertyTo(mappedEvent, outputPropPath, value)
		}

	} else if action.Transformation != nil {

		// Map using the transformation function.
		t := action.Transformation

		// Prepare the properties for the transformation.
		inPropsNames := t.In.PropertiesNames()
		inProps := make(map[string]any, len(inPropsNames))
		for _, name := range inPropsNames {
			value, ok := properties[name]
			if !ok {
				continue
			}
			inProps[name] = value
		}

		// Validate the input properties according to the transformation's input
		// schema.
		err := validateProps(inProps, t.In)
		if err != nil {
			return nil, fmt.Errorf("input schema validation failed: %s", err)
		}

		// Run the Python transformation function.
		pool := transformations.NewPool()
		mappedEvent, err = pool.Run(ctx, t.PythonSource, inProps)
		if err != nil {
			return nil, fmt.Errorf("error while calling the transformation function: %s", err)
		}

		// Validate the properties returned by Python according to the
		// transformation's output schema.
		err = validateProps(mappedEvent, t.Out)
		if err != nil {
			return nil, fmt.Errorf("output schema validation failed: %s", err)
		}

	}

	// If outSchema is a valid schema, then validate the mapped properties with
	// it.
	if outSchema.Valid() {
		err := validateProps(mappedEvent, outSchema)
		if err != nil {
			return nil, fmt.Errorf("mapped properties validation failed: %s", err)
		}
	}

	return mappedEvent, nil
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, propPath []string) (any, bool) {
	name := propPath[0]
	v, ok := m[name]
	if !ok {
		return nil, false
	}
	if len(propPath) == 1 {
		return v, ok
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return readPropertyFrom(obj, propPath[1:])
}

// validateProps validate the given properties using schema, returning error if
// the validation fails.
func validateProps(props map[string]any, schema types.Type) error {
	data, err := json.Marshal(props)
	if err != nil {
		return err
	}
	_, err = decode(bytes.NewReader(data), schema)
	return err
}

// writePropertyTo writes the property value v into m at the given property
// path.
// m cannot be nil.
func writePropertyTo(m map[string]any, propPath []string, v any) {
	name := propPath[0]
	if len(propPath) == 1 {
		m[name] = v
		return
	}
	_, ok := m[name]
	if !ok {
		m[name] = map[string]any{}
	}
	obj, ok := m[name].(map[string]any)
	if !ok {
		return
	}
	writePropertyTo(obj, propPath[1:], v)
}
