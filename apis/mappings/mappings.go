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
	"strings"

	"chichi/apis/normalization"
	"chichi/apis/state"
	"chichi/apis/transformations"
	"chichi/connector/types"
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
	in struct {
		path types.Path
		typ  types.Type
	}
	out struct {
		path     types.Path
		typ      types.Type
		nullable bool
	}
}

// Mapping represents a mapping.
type Mapping struct {
	inSchema, outSchema types.Type
	properties          []propertyMapping
	pythonSource        string
	formatTime          bool
}

// New returns a new mapping that maps properties of inSchema to outSchema using
// the given mapping or, in case of a transformation function, the Python
// source.
// Panics if none or both the mapping and the Python source are provided.
func New(inSchema, outSchema types.Type, mapp map[string]string, pythonSource string, formatTime bool) (*Mapping, error) {

	if (mapp == nil) == (pythonSource == "") {
		panic("one and only one of mapping and Python source must be provided")
	}
	if !inSchema.Valid() || !outSchema.Valid() {
		panic("input and output schemas must be valid")
	}

	m := Mapping{inSchema: inSchema, outSchema: outSchema, formatTime: formatTime}

	// Mapping.
	if mapp != nil {
		properties := make([]propertyMapping, 0, len(mapp))
		for out, in := range mapp {
			var pm propertyMapping
			pm.in.path = strings.Split(in, ".")
			pm.out.path = strings.Split(out, ".")
			prop, err := inSchema.PropertyByPath(pm.in.path)
			if err != nil {
				return nil, err
			}
			pm.in.typ = prop.Type
			prop, err = outSchema.PropertyByPath(pm.out.path)
			if err != nil {
				return nil, err
			}
			pm.out.typ = prop.Type
			pm.out.nullable = prop.Nullable
			properties = append(properties, pm)
		}
		m.properties = properties
		return &m, nil
	}

	// Transformation function.
	m.pythonSource = pythonSource

	return &m, nil
}

// Apply applies the mapping to values and returns the mapped values or an error
// if values cannot be mapped.
func (m *Mapping) Apply(ctx context.Context, values map[string]any) (map[string]any, error) {

	// Map using properties mapping.
	if m.properties != nil {
		outValues := map[string]any{}
		for _, property := range m.properties {
			value, ok := readPropertyFrom(values, property.in.path)
			if !ok {
				continue
			}
			if value == nil && property.out.nullable {
				// Conversion is not necessary.
			} else {
				v, err := convert(value, property.in.typ, property.out.typ, property.out.nullable, m.formatTime)
				if err != nil {
					log.Printf("cannot convert %#v (type %s) to type %s for property at path %q", value, property.in.typ, property.out.typ, property.out.path)
					continue
				}
				value = v
			}
			writePropertyTo(outValues, property.out.path, value)
		}
		return outValues, nil
	}

	// Map using the transformation function.

	// Prepare the properties for the transformation.
	// TODO(Gianluca): this may be no longer necessary. Review when refactoring
	// the normalization of properties.
	inPropNames := m.inSchema.PropertiesNames()
	inProps := make(map[string]any, len(inPropNames))
	for _, name := range inPropNames {
		value, ok := values[name]
		if !ok {
			continue
		}
		inProps[name] = value
	}

	// Run the Python transformation function.
	pool := transformations.NewPool()
	outValues, err := pool.Run(ctx, m.pythonSource, inProps)
	if err != nil {
		return nil, fmt.Errorf("error while calling the transformation function: %s", err)
	}

	// Normalize the Python output according to the mapping's output schema.
	outValues, err = normalizePythonOutput(outValues, m.outSchema, m.formatTime)
	if err != nil {
		return nil, err
	}

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
