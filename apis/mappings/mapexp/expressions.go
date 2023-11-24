//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"chichi/apis/state"
	"chichi/connector/types"
)

// ErrVoid is returned by the 'when' function when the first argument is false,
// and in this case the destination property is not changed.
var ErrVoid = errors.New("void")

// InvalidConversionError is the error returned by the Eval method of Expression
// when a value resulted from an evaluation cannot be converted to the
// destination type.
type InvalidConversionError struct {
	Value           any
	SourceType      types.Type
	DestinationType types.Type
}

func (err *InvalidConversionError) Error() string {
	if err.Value == nil {
		return fmt.Sprintf("cannot convert null to a non-nullable value")
	}
	return fmt.Sprintf("cannot convert %#v (type %s) to type %s", err.Value, err.SourceType, err.DestinationType)
}

// Expression represents a mapping expression used to transform data from a
// source to a destination. An Expression can contain strings, numbers, true,
// false, null, property paths and function calls.
type Expression struct {
	parts    []part     // expression parts.
	dt       types.Type // destination type.
	nullable bool       // reports whether the resulting value can be nil.
	layouts  *state.Layouts
}

// part represents an expression part within an Expression. An expression part
// can take different forms:
//
//   - value             example: "foo"
//   - path              example: x
//   - path(args)        example: add(x, 5)
//   - value path        example: 5 a.b
//   - value path(args)  example: "foo" coalesce(a.b, c)
//
// For instance, the Expression `"foo" x " " true a.b` is parsed as `"foo" x`,
// `" true" a.b`.
type part struct {
	// Value. If there is a path, value, if present, can only be of type Text.
	value any

	// Path or function name.
	// If it represents a function name, it consists of only the function name.
	// Otherwise, path elements follow these rules:
	//   - If it represents a map or JSON key, it starts with ':'.
	//   - If it was denoted with an indexing (e.g., a["b"]), it is enclosed in '[' and ']'.
	//   - If it was denoted by '?', it ends with '?'.
	// Examples of path elements: "x", "[x]" ":x", ":[$a]", ":x?", ":[x]?".
	path []string

	// Function call arguments.
	args [][]part

	// If there is a path, it represents the type of the property or the type of the function call.
	// Otherwise, it represents the type of the value. For some function calls, as coalesce, it is
	// the invalid type, indicating that the call can return different types.
	typ types.Type
}

// Compile parses a map expression and returns an Expression value that can be
// used to execute the expression. schema is the schema of the paths in the
// expression, dt is the destination type, nullable indicates whether the result
// value of the evaluation can be nil, and layouts represents, if not null, the
// layouts used to format DateTime, Date, and Time values as strings.
// An invalid schema can be passed to compile an expression without paths.
func Compile(expr string, schema types.Type, dt types.Type, nullable bool, layouts *state.Layouts) (*Expression, error) {
	if expr == "" {
		return nil, errors.New("expression is empty")
	}
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.New("schema is non an object")
	}
	if !dt.Valid() {
		return nil, errors.New("destination type is not valid")
	}
	parts, src, err := parseExpression(expr)
	if err != nil {
		return nil, err
	}
	if src != "" {
		return nil, fmt.Errorf("unexpected character %v", strconv.QuoteRuneToGraphic(rune(src[0])))
	}
	err = typeCheck(parts, schema, dt, nullable)
	if err != nil {
		return nil, err
	}
	expression := &Expression{
		parts:    parts,
		dt:       dt,
		nullable: nullable,
		layouts:  layouts,
	}
	return expression, nil
}

// Eval evaluates the map expression on the given values and returns the result.
// If the evaluation succeeds but cannot be converted to the destination type,
// it returns an InvalidConversionError error.
func (expr *Expression) Eval(values map[string]any) (any, error) {
	v, st, err := eval(expr.parts, values, expr.layouts)
	if err != nil {
		return nil, err
	}
	if v != nil || !expr.nullable {
		c, err := convert(v, st, expr.dt, expr.nullable, expr.layouts)
		if err != nil {
			if err == errInvalidConversion {
				err = &InvalidConversionError{v, st, expr.dt}
			}
			return nil, err
		}
		v = c
	}
	return v, err
}

// Properties returns the properties found in the expression, sorted by their
// appearance order in the expression. The returned properties are guaranteed to
// be unique. If no property are present, it returns nil.
//
// If the expression contains a map or JSON indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {{"x"}} if x is a
// JSON object, and returns {{"x", "z"}} if x is a map of objects.
func (expr *Expression) Properties() []types.Path {
	properties := appendProperties(nil, expr.parts)
	if properties == nil {
		return nil
	}
	if len(properties) == 1 {
		return properties
	}
	uniqueProperties := make([]types.Path, 0, len(properties))
	for _, property := range properties {
		var exists bool
		for _, p := range uniqueProperties {
			if p.Equals(property) {
				exists = true
				break
			}
		}
		if !exists {
			uniqueProperties = append(uniqueProperties, property)
		}
	}
	return uniqueProperties
}

// appendProperties appends the properties in expression to properties.
func appendProperties(properties []types.Path, expression []part) []types.Path {
	for _, expr := range expression {
		if expr.path == nil {
			continue
		}
		if expr.args == nil {
			path := make(types.Path, 0, len(expr.path))
			for _, name := range expr.path {
				if name[0] != ':' {
					if name[0] == '[' {
						name = name[1 : len(name)-1]
					}
					path = append(path, name)
				}
			}
			properties = append(properties, path)
			continue
		}
		for _, arg := range expr.args {
			properties = appendProperties(properties, arg)
		}
	}
	return properties
}

// eval evaluates expression and returns its value and type. values contains the
// property values. layouts represents, if not null, the layouts used to format
// DateTime, Date, and Time values as strings.
func eval(expression []part, values map[string]any, layouts *state.Layouts) (any, types.Type, error) {

	// Evaluate the most common cases that does not require a buffer.
	if len(expression) == 1 {
		p := expression[0]
		if p.path == nil {
			return p.value, p.typ, nil
		}
		if p.value == nil {
			if len(p.path) == 1 {
				if p.args == nil {
					return values[p.path[0]], p.typ, nil
				}
				return evalCall(p, values, layouts)
			}
			v, err := valueOf(p.path, values)
			if err != nil {
				return nil, types.Type{}, err
			}
			return v, p.typ, nil
		}
	}

	var v any
	var err error
	var vt types.Type
	var buf []byte

	for _, p := range expression {
		if s, _ := p.value.(string); s != "" {
			buf = append(buf, s...)
		}
		if p.path == nil {
			continue
		}
		if p.args == nil {
			v, err = valueOf(p.path, values)
			if err != nil {
				return nil, types.Type{}, err
			}
			vt = p.typ
		} else {
			v, vt, err = evalCall(p, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
		}
		buf, err = appendAsString(buf, v, vt)
		if err != nil {
			return nil, types.Type{}, err
		}
	}

	return string(buf), types.Text(), nil
}

// valueOf returns the value at the given path in values.
// It returns an error if the path does not exist.
func valueOf(path types.Path, values map[string]any) (any, error) {
	var v any
	var ok bool
	last := len(path) - 1
	for i, name := range path {
		if name[0] == ':' {
			name = name[1:]
			if n := len(name) - 1; name[n] == '?' {
				name = name[:n]
			}
		}
		if name[0] == '[' {
			name = name[1 : len(name)-1]
		}
		v, ok = values[name]
		if !ok {
			return nil, nil
		}
		if i != last {
			values, ok = v.(map[string]any)
			if !ok {
				if name := path[i+1]; name[len(name)-1] == '?' {
					return nil, ErrVoid
				}
				var t string
				switch v.(type) {
				case nil:
					t = "JSON null"
				case bool:
					t = "a JSON boolean"
				case float64, json.Number:
					t = "a JSON number"
				case string:
					t = "a JSON string"
				default:
					t = "a JSON array"
				}
				return nil, fmt.Errorf("invalid %s: %s is not a JSON object, it is %s", stringifyPath(path[:i+2]), stringifyPath(path[:i+1]), t)
			}
		}
	}
	return v, nil
}

// stringifyPath returns path as a string.
func stringifyPath(path []string) string {
	s := path[0]
	for _, name := range path[1:] {
		if name[0] == ':' {
			name = name[1:]
		}
		question := name[len(name)-1] == '?'
		if question {
			name = name[:len(name)-1]
		}
		if name[0] == '[' {
			s += "[" + strconv.Quote(name[1:len(name)-1]) + "]"
		} else {
			s += "." + name
		}
		if question {
			s += "?"
		}
	}
	return s
}

// evalCall evaluates p representing a function call, and returns its value and
// type. values contains the property values. layouts represents, if not null,
// the layouts used to format DateTime, Date, and Time values as strings.
func evalCall(p part, values map[string]any, layouts *state.Layouts) (any, types.Type, error) {
	switch name := p.path[0]; name {
	case "and":
		for _, arg := range p.args {
			v, _, err := eval(arg, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
			if !v.(bool) {
				return false, types.Boolean(), nil
			}
		}
		return true, types.Boolean(), nil
	case "array":
		a := make([]any, len(p.args))
		for i, arg := range p.args {
			v, _, err := eval(arg, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
			a[i] = v
		}
		return a, types.Array(types.JSON()), nil
	case "coalesce":
		for _, arg := range p.args {
			v, vt, err := eval(arg, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v != nil {
				return v, vt, nil
			}
		}
		return nil, p.typ, nil
	case "eq":
		v0, t0, err := eval(p.args[0], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		v1, t1, err := eval(p.args[1], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		if !t0.EqualTo(t1) {
			v0, err = convert(v0, t0, t1, true, layouts)
			if err != nil {
				if err == errInvalidConversion {
					return false, types.Boolean(), nil
				}
				return nil, types.Type{}, err
			}
		}
		return reflect.DeepEqual(v0, v1), types.Boolean(), nil
	case "when":
		v0, _, err := eval(p.args[0], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		if !v0.(bool) {
			return nil, types.Type{}, ErrVoid
		}
		v1, t1, err := eval(p.args[1], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		return v1, t1, nil
	}
	panic(fmt.Errorf("unknown function %q", p.path[0]))
}
