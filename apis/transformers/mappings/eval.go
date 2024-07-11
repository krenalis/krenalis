//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

// Void represents the void value.
var Void = struct{}{}

// errVoid is returned by the 'if' function when it has only two arguments and
// the first argument is false. In this case, the destination property is not
// changed.
var errVoid = errors.New("void")

// invalidConversionError is the error returned by the Eval and Transform
// methods of when a value resulted from an evaluation cannot be converted to
// the destination type.
type invalidConversionError struct {
	value           any
	sourceType      types.Type
	destinationType types.Type
}

func (err *invalidConversionError) Error() string {
	switch err.value {
	case nil:
		return "cannot convert null to a non-nullable value"
	case Void:
		return "expression is required, but the evaluation returned no value"
	}
	return fmt.Sprintf("cannot convert %#v (type %s) to type %s", err.value, err.sourceType, err.destinationType)
}

// Eval evaluates the expression using the provided values for the properties,
// which must conform to the expression's source schema, and returns the result
// that conforms to the expression's destination type or Void if the result is
// void.
func (expr *Expression) Eval(values map[string]any) (any, error) {
	v, st, err := eval(expr.parts, values, expr.timeLayouts)
	if err != nil {
		if err == errVoid {
			if !expr.required {
				return Void, nil
			}
			err = &invalidConversionError{Void, st, expr.dt}
		}
		return nil, err
	}
	if v != nil || !expr.nullable {
		c, err := convert(v, st, expr.dt, expr.nullable, expr.timeLayouts)
		if err != nil {
			if err == errVoid {
				if !expr.required {
					return Void, nil
				}
				err = errInvalidConversion
			}
			if err == errInvalidConversion {
				err = &invalidConversionError{v, st, expr.dt}
			}
			return nil, err
		}
		v = c
	}
	return v, err
}

// appendAsString appends v to b after converting it to a string.
// Calling appendAsString(b, v, t) is the same of calling
// convert(v, t, types.Text(), false, false) and appending the result to b.
func appendAsString(b []byte, v any, t types.Type) ([]byte, error) {
	if v == nil {
		return b, nil
	}
	if s, ok := v.(string); ok {
		return append(b, s...), nil
	}
	switch t.Kind() {
	case types.BooleanKind:
		strconv.AppendBool(b, v.(bool))
	case types.IntKind, types.YearKind:
		return strconv.AppendInt(b, int64(v.(int)), 10), nil
	case types.UintKind:
		return strconv.AppendUint(b, uint64(v.(uint)), 10), nil
	case types.FloatKind:
		return strconv.AppendFloat(b, v.(float64), 'g', -1, t.BitSize()), nil
	case types.DecimalKind:
		return append(b, v.(decimal.Decimal).String()...), nil
	case types.DateTimeKind:
		return v.(time.Time).AppendFormat(b, time.RFC3339Nano), nil
	case types.DateKind:
		return v.(time.Time).AppendFormat(b, time.DateOnly), nil
	case types.TimeKind:
		return v.(time.Time).AppendFormat(b, "15:04:05.999999999"), nil
	case types.JSONKind:
		switch v := v.(type) {
		case float64:
			return strconv.AppendFloat(b, v, 'g', -1, 64), nil
		case json.Number:
			return append(b, v...), nil
		case bool:
			strconv.AppendBool(b, v)
		case json.RawMessage:
			s, err := jsonToText(v)
			if err == nil {
				return append(b, s...), nil
			}
		}
	}
	return b, errInvalidConversion
}

// eval evaluates expression and returns its value and type. values contains the
// property values. layouts represents, if not nil, the layouts used to format
// DateTime, Date, and Time values as strings.
//
// If the result of the evaluation is void, it returns the errVoid error.
func eval(expression []part, values map[string]any, layouts *state.TimeLayouts) (any, types.Type, error) {

	// Evaluate the most common cases that does not require a buffer.
	if len(expression) == 1 {
		p := expression[0]
		if p.path == nil {
			return p.value, p.typ, nil
		}
		if p.value == nil {
			if len(p.path) == 1 {
				if p.args == nil {
					v, ok := values[p.path[0]]
					if !ok {
						return nil, types.Type{}, errVoid
					}
					return v, p.typ, nil
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
			if err != nil && err != errVoid {
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

// evalCall evaluates p representing a function call, and returns its value and
// type. values contains the property values. timeLayouts represents, if not
// nil, the timeLayouts used to format DateTime, Date, and Time values as
// strings.
func evalCall(p part, values map[string]any, layouts *state.TimeLayouts) (any, types.Type, error) {
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
		if !types.Equal(t0, t1) {
			v0, err = convert(v0, t0, t1, true, layouts)
			if err != nil {
				if err == errInvalidConversion {
					return false, types.Boolean(), nil
				}
				return nil, types.Type{}, err
			}
		}
		return reflect.DeepEqual(v0, v1), types.Boolean(), nil
	case "if":
		v0, _, err := eval(p.args[0], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v0.(bool) {
			return eval(p.args[1], values, layouts)
		}
		if len(p.args) == 3 {
			return eval(p.args[2], values, layouts)
		}
		return nil, types.Type{}, errVoid
	}
	panic(fmt.Errorf("unknown function %q", p.path[0]))
}

// valueOf returns the value at the specified path in values. It returns errVoid
// if the path does not exist, including keys in a map and properties of a JSON
// object.
//
// For non-object JSON values, accessing a key returns errVoid if the key is
// followed by "?"; otherwise, it returns an error.
func valueOf(path path, values map[string]any) (any, error) {
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
			return nil, errVoid
		}
		if i != last {
			values, ok = v.(map[string]any)
			if !ok {
				if name := path[i+1]; name[len(name)-1] == '?' {
					return nil, errVoid
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
