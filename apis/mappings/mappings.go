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
	"log"
	"strings"

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
		path []string
		typ  types.Type
	}
	out struct {
		path     []string
		typ      types.Type
		nullable bool
	}
}

// Mapping represents a mapping.
type Mapping struct {
	properties     []propertyMapping
	transformation *state.Transformation
}

// New returns a new mapping that maps properties of inSchema to outSchema using
// the given mapping or transformation.
func New(inSchema, outSchema types.Type, mapp map[string]string, transf *state.Transformation) (*Mapping, error) {
	if mapp != nil {
		var err error
		properties := make([]propertyMapping, 0, len(mapp))
		for out, in := range mapp {
			var pm propertyMapping
			pm.in.path = strings.Split(in, ".")
			pm.out.path = strings.Split(out, ".")
			pm.in.typ, _, err = propertyByPath(pm.in.path, inSchema)
			if err != nil {
				return nil, err
			}
			pm.out.typ, pm.out.nullable, err = propertyByPath(pm.out.path, outSchema)
			if err != nil {
				return nil, err
			}
			properties = append(properties, pm)
		}
		return &Mapping{properties: properties}, nil
	}
	return &Mapping{transformation: transf}, nil
}

// propertyByPath returns the property at the given path of the Object t.
// If the property does not exist, it returns an error.
func propertyByPath(path []string, t types.Type) (types.Type, bool, error) {
	last := len(path) - 1
	for i, name := range path {
		p, ok := t.Property(name)
		if !ok {
			return types.Type{}, false, fmt.Errorf("property %s does not exist", strings.Join(path[:i+1], "."))
		}
		if i == last {
			return p.Type, p.Nullable, nil
		}
		if t.PhysicalType() != types.PtObject {
			return types.Type{}, false, fmt.Errorf("property %s is not an object", strings.Join(path[:i+1], "."))
		}
		t = p.Type
	}
	panic("unreachable")
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
			if value == nil {
				if !property.out.nullable {
					log.Printf("property path %s is not nullable", strings.Join(property.out.path, "."))
				}
			} else {
				v, err := convert(value, property.in.typ, property.out.typ)
				if err != nil {
					path := strings.Join(property.out.path, ".")
					log.Printf("cannot convert %#v (type %s) to type %s for property at path %q", value, property.in.typ, property.out.typ, path)
					continue
				}
				value = v
			}
			writePropertyTo(outValues, property.out.path, value)
		}
		return outValues, nil
	}

	// Map using the transformation function.
	t := m.transformation

	// Prepare the properties for the transformation.
	inPropsNames := t.In.PropertiesNames()
	inProps := make(map[string]any, len(inPropsNames))
	for _, name := range inPropsNames {
		value, ok := values[name]
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
	outValues, err := pool.Run(ctx, t.PythonSource, inProps)
	if err != nil {
		return nil, fmt.Errorf("error while calling the transformation function: %s", err)
	}

	// Validate the properties returned by Python according to the
	// transformation's output schema.
	err = validateProps(outValues, t.Out)
	if err != nil {
		return nil, fmt.Errorf("output schema validation failed: %s", err)
	}

	return outValues, nil
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
