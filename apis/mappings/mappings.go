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
	"strconv"
	"strings"

	"chichi/apis/state"
	"chichi/apis/transformers"
	"chichi/apis/transformers/mapexp"
	"chichi/connector/types"
)

// Error represents an error resulting from a mapping or transformation
// function, such as a syntax error in the function, or the use of a
// non-existent property.
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

// Mapping represents a mapping.
type Mapping struct {
	inSchema, outSchema types.Type
	mapTransformer      *mapexp.Transformer
	transformation      *state.Transformation
	transformer         transformers.Function
	action              int
}

// New returns a new mapping that maps properties of inSchema to outSchema using
// the given mapping and, in case a transformation is provided, also uses such
// transformation. layouts represents, if not null, the layouts used to format
// DateTime, Date, and Time values as strings.
func New(inSchema, outSchema types.Type, mappings map[string]string, transformation *state.Transformation, action int, transformer transformers.Function, layouts *state.Layouts) (*Mapping, error) {

	if !outSchema.Valid() {
		return nil, errors.New("output schema is not valid")
	}

	m := Mapping{
		inSchema:       inSchema,
		outSchema:      outSchema,
		transformation: transformation,
		transformer:    transformer,
		action:         action,
	}

	// Mapping.
	if mappings != nil {
		var err error
		m.mapTransformer, err = mapexp.New(mappings, inSchema, outSchema, layouts)
		if err != nil {
			return nil, err
		}
	}

	return &m, nil
}

// Apply applies the mapping to values and returns the mapped values or an error
// if values cannot be mapped.
func (m *Mapping) Apply(ctx context.Context, values map[string]any) (map[string]any, error) {

	// Map using properties mapping.
	if m.mapTransformer != nil {
		return m.mapTransformer.Transform(values)
	}

	// Map using the transformation.
	funcName := transformationFunctionName(m.action, m.transformation.Language)
	results, err := m.transformer.Call(ctx, funcName, m.transformation.Version, m.inSchema, m.outSchema, []map[string]any{values})
	if err != nil {
		if err, ok := err.(*transformers.ExecutionError); ok {
			return nil, errorf("%s: %s ", m.transformation.Language.String(), err.Msg)
		}
		return nil, fmt.Errorf("error while execution the transformation: %s", err)
	}
	if err := results[0].Error; err != nil {
		if err, ok := err.(*transformers.ExecutionError); ok {
			return nil, errorf("%s: %s ", m.transformation.Language.String(), err.Msg)
		}
		return nil, errorf("%s", err)
	}
	out := results[0].Value

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
