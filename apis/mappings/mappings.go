//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"chichi/apis/mappings/mapexp"
	"chichi/apis/normalization"
	"chichi/apis/state"
	"chichi/apis/transformations"
	"chichi/connector/types"

	"golang.org/x/exp/maps"
)

// ActionFilterApplies reports whether the action filter applies to the props,
// which can be an event or an user.
// Returns error if one of the properties of the filter are not found within
// props.
func ActionFilterApplies(filter *state.ActionFilter, props map[string]any) (bool, error) {
	if filter == nil {
		return true, nil
	}
	for _, cond := range filter.Conditions {
		value, ok := readPropertyFrom(props, strings.Split(cond.Property, "."))
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

// propertyMapping represents a property to map.
type propertyMapping struct {
	expression *mapexp.Expression
	outPath    types.Path
}

// Mapping represents a mapping.
type Mapping struct {
	inSchema, outSchema types.Type
	properties          []propertyMapping
	transformation      *state.Transformation
	formatTime          bool
}

// New returns a new mapping that maps properties of inSchema to outSchema using
// the given mapping and, in case a transformation function is provided, also
// uses such transformation.
func New(inSchema, outSchema types.Type, mappings map[string]string, transformation *state.Transformation, formatTime bool) (*Mapping, error) {

	if !inSchema.Valid() || !outSchema.Valid() {
		panic("input and output schemas must be valid")
	}

	m := Mapping{inSchema: inSchema, outSchema: outSchema,
		transformation: transformation, formatTime: formatTime}

	// Mapping.
	if mappings != nil {
		m.properties = make([]propertyMapping, 0, len(mappings))
		for path, expression := range mappings {
			property := propertyMapping{outPath: strings.Split(path, ".")}
			outProperty, err := outSchema.PropertyByPath(property.outPath)
			if err != nil {
				return nil, err
			}
			property.expression, err = mapexp.Compile(expression, inSchema, outProperty.Type, outProperty.Nullable)
			if err != nil {
				return nil, err
			}
			m.properties = append(m.properties, property)
		}
	}

	return &m, nil
}

// Apply applies the mapping to values and returns the mapped values or an error
// if values cannot be mapped.
func (m *Mapping) Apply(ctx context.Context, values map[string]any) (map[string]any, error) {

	outValues := map[string]any{}

	// Map using properties mapping.
	if m.properties != nil {
		for _, property := range m.properties {
			v, err := property.expression.Eval(values, m.formatTime)
			if err != nil {
				if err == mapexp.ErrVoid {
					continue
				}
				if err, ok := err.(*mapexp.InvalidConversionError); ok {
					log.Print(err)
					continue
				}
				return nil, err
			}
			writePropertyTo(outValues, property.outPath, v)
		}
	}

	if m.transformation == nil {
		return outValues, nil
	}

	// Map using the transformation function.

	// Prepare the properties for the transformation.
	// TODO(Gianluca): this may be no longer necessary. Review when refactoring
	// the normalization of properties.
	inProps := make(map[string]any, len(m.transformation.In))
	for _, name := range m.transformation.In {
		value, ok := values[name]
		if !ok {
			continue
		}
		inProps[name] = value
	}

	// Run the Python transformation function.
	pool := transformations.NewPool()
	transformationOutValues, err := pool.Run(ctx, m.transformation.Func, inProps)
	if err != nil {
		return nil, fmt.Errorf("error while calling the transformation function: %s", err)
	}

	// Verify that the transformation function hasn't returned property values
	// that are not present in the output schema.
	if len(m.transformation.Out) != len(transformationOutValues) {
		names := maps.Keys(transformationOutValues)
		sort.Strings(names)
		for _, got := range names {
			found := false
			for _, expected := range m.transformation.Out {
				if got == expected {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("transformation function has returned the unexpected property %q", got)
			}
		}
	}

	// Normalize the Python output according to the mapping's output schema.
	transformationOutValues, err = normalizePythonOutput(transformationOutValues, m.outSchema, m.formatTime)
	if err != nil {
		return nil, err
	}

	maps.Copy(outValues, transformationOutValues)

	return outValues, nil
}

// normalizePythonOutput normalizes the values returned by Python according to
// the given schema.
//
// formatTime reports whether DateTime and Date values should be formatted based
// on the layout, if any.
func normalizePythonOutput(values map[string]any, schema types.Type, formatTime bool) (map[string]any, error) {
	out := make(map[string]any, len(values))
	for name, value := range values {
		prop, ok := schema.Property(name)
		if !ok {
			return nil, fmt.Errorf("property %q not found", name)
		}
		v, err := normalization.NormalizePythonProperty(name, prop.Type, value, prop.Nullable, formatTime)
		if err != nil {
			return nil, err
		}
		out[name] = v
	}
	return out, nil
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, path types.Path) (any, bool) {
	name := path[0]
	v, ok := m[name]
	if !ok {
		return nil, false
	}
	if len(path) == 1 {
		return v, ok
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return readPropertyFrom(obj, path[1:])
}

// writePropertyTo writes the property value v into m at the given property
// path.
// m cannot be nil.
func writePropertyTo(m map[string]any, path types.Path, v any) {
	name := path[0]
	if len(path) == 1 {
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
	writePropertyTo(obj, path[1:], v)
}
