//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"chichi/apis/mappings/mapexp"
	"chichi/apis/normalization"
	"chichi/apis/state"
	"chichi/apis/transformers"
	"chichi/connector/types"
)

// Error represents an error resulting from a mapping or transformation
// function, such as a syntax error in the function, the use of a non-existent
// property, or a property being used as an identifier.
type Error string

func (err Error) Error() string { return string(err) }

func errorf(format string, a ...any) error {
	return Error(fmt.Sprintf(format, a...))
}

// FilterApplies reports whether the filter applies to props, which can be an
// event or a user. Returns error if one of the properties of the filter are not
// found within props.
func FilterApplies(filter *state.Filter, props map[string]any) (bool, error) {
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
	required            int
	transformation      *state.Transformation
	transformer         transformers.Transformer
	action              int
	formatTime          bool
}

// New returns a new mapping that maps properties of inSchema to outSchema using
// the given mapping and, in case a transformation is provided, also uses such
// transformation.
func New(inSchema, outSchema types.Type, mappings map[string]string, transformation *state.Transformation, action int, transformer transformers.Transformer, formatTime bool) (*Mapping, error) {

	if !inSchema.Valid() || !outSchema.Valid() {
		return nil, errors.New("input or output schema is not valid")
	}

	m := Mapping{
		inSchema:       inSchema,
		outSchema:      outSchema,
		transformation: transformation,
		transformer:    transformer,
		action:         action,
		formatTime:     formatTime,
	}

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

	if transformation != nil {
		for _, p := range m.outSchema.Properties() {
			if p.Required {
				m.required++
			}
		}
	}

	return &m, nil
}

// Apply applies the mapping to values and returns the mapped values or an error
// if values cannot be mapped.
func (m *Mapping) Apply(ctx context.Context, values map[string]any) (map[string]any, error) {

	out := map[string]any{}

	// Map using properties mapping.
	if m.properties != nil {
		for _, property := range m.properties {
			v, err := property.expression.Eval(values, m.formatTime)
			if err != nil {
				if err == mapexp.ErrVoid {
					continue
				}
				if err, ok := err.(*mapexp.InvalidConversionError); ok {
					slog.Info("cannot convert property", "err", err)
					continue
				}
				return nil, err
			}
			writePropertyTo(out, property.outPath, v)
		}
	}

	if m.transformation == nil {
		return out, nil
	}

	// Map using the transformation.

	// Transform values.
	funcName := transformationFunctionName(m.action, m.transformation.Language)
	results, err := m.transformer.CallFunction(ctx, funcName, m.transformation.Version, m.inSchema, []map[string]any{values})
	if err != nil {
		if err, ok := err.(*transformers.ExecutionError); ok {
			return nil, errorf("%s: %s ", m.transformation.Language.String(), err.Msg)
		}
		return nil, fmt.Errorf("error while execution the transformation: %s", err)
	}
	values = results[0].Value
	if values == nil {
		return nil, errorf("%s: %s ", m.transformation.Language.String(), results[0].Error)
	}

	// Validate and normalize the transformed values.
	missingRequired := m.required
	for name, value := range values {
		if _, ok := out[name]; ok {
			return nil, errorf("property %q is already used as identifier", name)
		}
		prop, ok := m.outSchema.Property(name)
		if !ok {
			return nil, errorf("property %q does not exist", name)
		}
		if prop.Required {
			missingRequired--
		}
		v, err := normalization.NormalizeTransformationProperty(name, prop.Type, value, prop.Nullable, m.formatTime)
		if err != nil {
			return nil, err
		}
		out[name] = v
	}
	if missingRequired > 0 {
		for _, p := range m.outSchema.Properties() {
			if p.Required {
				if _, ok := out[p.Name]; !ok {
					return nil, errorf("required property %q is missing", p.Name)
				}
			}
		}
		panic("unreachable code")
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

// transformationFunctionName returns the name the transformation function for
// an action in the specified language.
//
// Keep in sync with the function having the same name in the apis package.
func transformationFunctionName(action int, language state.Language) string {
	var ext string
	switch language {
	case state.JavaScript:
		ext = ".js"
	case state.Python:
		ext = ".py"
	default:
		panic("unexpected language")
	}
	return "action-" + strconv.Itoa(action) + ext
}
