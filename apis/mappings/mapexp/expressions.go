//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

import (
	"errors"
	"fmt"
	"strconv"

	"chichi/connector/types"
)

// ErrNotConvertible is the error returned by the Compile method of Expression
// when the type of the expression cannot convert to the destination type.
var ErrNotConvertible = errors.New("expression is not convertible")

// InvalidConversionError is the error returned by the Eval method of Expression
// when a value resulted from an evaluation cannot be converted to the
// destination type.
type InvalidConversionError struct {
	Value           any
	SourceType      types.Type
	DestinationType types.Type
}

func (err *InvalidConversionError) Error() string {
	return fmt.Sprintf("cannot convert %#v (type %s) to type %s", err.Value, err.SourceType, err.DestinationType)
}

type Expression struct {
	e        []expr
	dt       types.Type
	nullable bool
}

type expr struct {
	text  string     // Text that starts the expression
	value any        // Value, can be true, false, null or a decimal.Decimal value. Is nil is path is not nil.
	path  types.Path // Property path or function name. Is nil is value is not nil.
	args  [][]expr   // Function call arguments. It is nil if Path is nil or refers to a property
	typ   types.Type // Type of the value or the type of the property at path.
}

// TextOnly reports whether the expression contains only text.
func (e expr) TextOnly() bool {
	return !e.typ.Valid() && e.args == nil
}

// ValueOnly reports whether the expression contains only a value.
func (e expr) ValueOnly() bool {
	return e.typ.Valid() && e.path == nil
}

// PathOnly reports whether the expression contains only a path.
func (e expr) PathOnly() bool {
	return e.path != nil && e.text == ""
}

// Compile parses a map expression and returns an Expression value that can be
// used to execute the expression. schema is the schema of the paths in the
// expression, dt is the destination type, and nullable indicates whether
// the result value of the evaluation can be nil.
func Compile(expr string, schema types.Type, dt types.Type, nullable bool) (*Expression, error) {
	if expr == "" {
		return nil, errors.New("expression is empty")
	}
	if schema.PhysicalType() != types.PtObject {
		return nil, errors.New("schema is non an object")
	}
	if !dt.Valid() {
		return nil, errors.New("destination type is not valid")
	}
	e, src, err := parseExpression(expr, schema)
	if err != nil {
		return nil, err
	}
	if src != "" {
		return nil, fmt.Errorf("unexpected character %v", strconv.QuoteRuneToGraphic(rune(src[0])))
	}
	if !convertible(e, dt.PhysicalType()) {
		return nil, ErrNotConvertible
	}
	expression := &Expression{
		e:        e,
		dt:       dt,
		nullable: nullable,
	}
	return expression, nil
}

// Eval evaluates the map expression on the given values and returns the result.
// formatTime reports whether DateTime and Date values should be formatted based
// on the layout of the destination type, if any.
// If the evaluation succeeds but cannot be converted to the destination type,
// it returns an InvalidConversionError error.
func (expr *Expression) Eval(values map[string]any, formatTime bool) (any, error) {
	v, st, err := eval(expr.e, values)
	if err != nil {
		return nil, err
	}
	if v != nil || !expr.nullable {
		c, err := convert(v, st, expr.dt, expr.nullable, formatTime)
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

// PropertyPaths returns the property paths found in the expression, sorted by
// their appearance order in the expression. The returned paths are guaranteed
// to be unique. If no property paths are present, it returns nil.
func (expr *Expression) PropertyPaths() []types.Path {
	paths := appendPropertyPaths(nil, expr.e)
	if paths == nil {
		return nil
	}
	if len(paths) == 1 {
		return paths
	}
	uniquePaths := make([]types.Path, 0, len(paths))
	for _, path := range paths {
		var exists bool
		for _, p := range uniquePaths {
			if equalPaths(path, p) {
				exists = true
				break
			}
		}
		if !exists {
			uniquePaths = append(uniquePaths, path)
		}
	}
	return uniquePaths
}

func equalPaths(p1 types.Path, p2 types.Path) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i, name := range p1 {
		if name != p2[i] {
			return false
		}
	}
	return true
}

// appendPropertyPaths appends to the property paths in expression to paths.
func appendPropertyPaths(paths []types.Path, expression []expr) []types.Path {
	for _, expr := range expression {
		if expr.path == nil {
			continue
		}
		if expr.args == nil {
			paths = append(paths, expr.path)
			continue
		}
		for _, arg := range expr.args {
			paths = appendPropertyPaths(paths, arg)
		}
	}
	return paths
}

// eval evaluates expr and returns its value and type. values contains the
// property values.
func eval(expr []expr, values map[string]any) (any, types.Type, error) {

	// Evaluate the most common cases that does not require a buffer.
	if len(expr) == 1 {
		expr := expr[0]
		if expr.PathOnly() {
			if len(expr.path) == 1 {
				name := expr.path[0]
				if expr.args == nil {
					return values[name], expr.typ, nil
				}
				return evalCall(name, expr.args, values)
			}
			value, err := valueOf(expr.path, values)
			if err != nil {
				return nil, types.Type{}, err
			}
			return value, expr.typ, nil
		}
		if expr.ValueOnly() {
			return expr.value, expr.typ, nil
		}
		if expr.TextOnly() {
			return expr.text, types.Text(), nil
		}
	}

	var v any
	var err error
	var vt types.Type
	var buf []byte

	for _, exp := range expr {
		buf = append(buf, exp.text...)
		if exp.path == nil {
			continue
		}
		if args := exp.args; args == nil {
			v, err = valueOf(exp.path, values)
			if err != nil {
				return nil, types.Type{}, err
			}
			vt = exp.typ
		} else {
			v, vt, err = evalCall(exp.path[0], args, values)
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
		v, ok = values[name]
		if !ok {
			return nil, fmt.Errorf("cannot find value for property path %q", path[:i+1])
		}
		if i != last {
			values, ok = v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("cannot find value for property path %q (%q has type %T)", path[:i+2], path[:i+1], v)
			}
		}
	}
	return v, nil
}

// evalCall evaluates a call to the function named name with arguments args, and
// returns its value and type. values contains the property values.
func evalCall(name string, args [][]expr, values map[string]any) (any, types.Type, error) {
	switch name {
	case "coalesce":
		for _, arg := range args {
			v, vt, err := eval(arg, values)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v != nil {
				return v, vt, nil
			}
		}
		return nil, types.JSON(), nil
	}
	panic("unknown function")
}

// convertible reports whether expression is convertible to a type with physical
// type dt.
func convertible(expression []expr, dt types.PhysicalType) bool {
	if len(expression) > 1 || expression[0].text != "" {
		return convertibleTo(types.PtText, dt)
	}
	exp := expression[0]
	if exp.args == nil {
		if exp.path == nil {
			return convertibleTo(types.PtDecimal, dt)
		}
		return convertibleTo(exp.typ.PhysicalType(), dt)
	}
	switch exp.path[0] {
	case "coalesce":
		for _, arg := range exp.args {
			if !convertible(arg, dt) {
				return false
			}
		}
		return true
	}
	panic("unknown function")
}
