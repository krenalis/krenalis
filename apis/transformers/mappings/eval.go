//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

// invalidConversionError is the error returned by the Eval and Transform
// methods of when a value resulted from an evaluation cannot be converted to
// the destination type.
type invalidConversionError struct {
	v   any
	st  types.Type
	dt  types.Type
	msg string
}

func (err *invalidConversionError) Error() string {
	if err.msg != "" {
		return err.msg
	}
	switch err.v {
	case nil:
		return "cannot convert null to a non-nullable value"
	}
	return fmt.Sprintf("cannot convert %#v (type %s) to type %s", err.v, err.st, err.dt)
}

// Eval evaluates the expression using the provided properties which must
// conform to the expression's source schema, and returns the result that
// conforms to the expression's destination type.
//
// purpose specifies the reason for the evaluation. If Create or Update, then
// all the properties required for creation or the update must be present in the
// returned value.
//
// Eval might replace JSON properties in the properties map with their
// unmarshalled values.
func (expr *Expression) Eval(properties map[string]any, purpose Purpose) (any, error) {
	v, st, err := eval(expr.parts, properties, expr.timeLayouts, purpose)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return v, err
	}
	c, err := convert(v, st, expr.dt, true, expr.timeLayouts, purpose)
	if err != nil {
		if err == errInvalidConversion {
			err = &invalidConversionError{v, st, expr.dt, ""}
		}
		return nil, err
	}
	return c, nil
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

// eval evaluates expression and returns its value and type. properties are the
// property values. layouts represents, if not nil, the layouts used to format
// DateTime, Date, and Time values as strings, and purpose specifies the reason
// for the evaluation.
func eval(expression []part, properties map[string]any, layouts *state.TimeLayouts, purpose Purpose) (any, types.Type, error) {

	// Evaluate the most common cases that does not require a buffer.
	if len(expression) == 1 {
		p := expression[0]
		if p.path == nil {
			return p.value, p.typ, nil
		}
		if p.value == nil {
			if len(p.path) == 1 {
				if p.args == nil {
					v, ok := properties[p.path[0]]
					if !ok {
						return nil, types.Type{}, nil
					}
					return v, p.typ, nil
				}
				return evalCall(p, properties, layouts, purpose)
			}
			v, err := valueOf(p.path, properties)
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
			v, err = valueOf(p.path, properties)
			if err != nil {
				return nil, types.Type{}, err
			}
			vt = p.typ
		} else {
			v, vt, err = evalCall(p, properties, layouts, purpose)
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
// type. properties contains the property values. timeLayouts represents, if not
// nil, the timeLayouts used to format DateTime, Date, and Time values as
// strings. purpose specifies the reason for the evaluation.
func evalCall(p part, properties map[string]any, layouts *state.TimeLayouts, purpose Purpose) (any, types.Type, error) {
	switch name := p.path[0]; name {
	case "and":
		for _, arg := range p.args {
			v, _, err := eval(arg, properties, layouts, purpose)
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
			v, _, err := eval(arg, properties, layouts, purpose)
			if err != nil {
				return nil, types.Type{}, err
			}
			a[i] = v
		}
		return a, types.Array(types.JSON()), nil
	case "coalesce":
		for _, arg := range p.args {
			v, vt, err := eval(arg, properties, layouts, purpose)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v != nil {
				return v, vt, nil
			}
		}
		return nil, p.typ, nil
	case "eq":
		v0, t0, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		v1, t1, err := eval(p.args[1], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if !types.Equal(t0, t1) {
			v0, err = convert(v0, t0, t1, true, layouts, purpose)
			if err != nil {
				if err == errInvalidConversion {
					return false, types.Boolean(), nil
				}
				return nil, types.Type{}, err
			}
		}
		return reflect.DeepEqual(v0, v1), types.Boolean(), nil
	case "if":
		v0, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v0.(bool) {
			return eval(p.args[1], properties, layouts, purpose)
		}
		if len(p.args) == 3 {
			return eval(p.args[2], properties, layouts, purpose)
		}
		return nil, types.Type{}, nil
	}
	panic(fmt.Errorf("unknown function %q", p.path[0]))
}

// valueOf returns the value at the specified path in properties. It returns nil
// if the path does not exist, including keys in a map and properties of a JSON
// object.
//
// For non-object JSON values, accessing a key returns nil if the key is
// followed by "?"; otherwise, it returns an error.
//
// For a JSON property of type json.RawMessage, the function unmarshals the
// value and replaces it with the unmarshalled value in the properties map.
func valueOf(path path, properties map[string]any) (any, error) {
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
		v, ok = properties[name]
		if !ok {
			return nil, nil
		}
		if i != last {
			var ok bool
			switch v2 := v.(type) {
			case map[string]any:
				properties, ok = v2, true
			case json.RawMessage:
				if v2[0] == '{' {
					dec := json.NewDecoder(bytes.NewReader(v2))
					dec.UseNumber()
					v = nil
					_ = dec.Decode(&v)
					properties[name] = v
					properties, ok = v.(map[string]any), true
				}
			}
			if !ok {
				if name := path[i+1]; name[len(name)-1] == '?' {
					return nil, nil
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
				return nil, &invalidConversionError{msg: fmt.Sprintf("invalid %s: %s is not a JSON object, it is %s", stringifyPath(path[:i+2]), stringifyPath(path[:i+1]), t)}
			}
		}
	}
	return v, nil
}
