//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/go-json-experiment/json/jsontext"
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
		v := v.(json.Value)
		switch v.Kind() {
		case json.Array, json.Object:
		case json.String:
			return jsontext.AppendUnquote(b, v)
		default:
			return append(b, v...), nil
		}
	}
	return b, errInvalidConversion
}

// digitCountInt returns the number of decimal digits in n, including the sign
// for negative numbers.
func digitCountInt(n int64) int {
	if n == 0 {
		return 1
	}
	sign := 0
	if n < 0 {
		if n == math.MinInt64 {
			return 20
		}
		sign = 1
		n = -n
	}
	return sign + int(math.Log10(float64(n))) + 1
}

// digitCountUint returns the number of decimal digits in n.
func digitCountUint(n uint64) int {
	if n == 0 {
		return 1
	}
	return int(math.Log10(float64(n))) + 1
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
		var null bool
		for _, arg := range p.args {
			v, _, err := eval(arg, properties, layouts, purpose)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v == nil {
				null = true
				continue
			}
			if !v.(bool) {
				return false, types.Boolean(), nil
			}
		}
		if null {
			return nil, types.Boolean(), nil
		}
		return true, types.Boolean(), nil
	case "array":
		var b bytes.Buffer
		enc := stdjson.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		arr := make([]any, len(p.args))
		for i, arg := range p.args {
			v, _, err := eval(arg, properties, layouts, purpose)
			if err != nil {
				return nil, types.Type{}, err
			}
			switch v.(type) {
			case nil:
				v = json.Value("null")
			case json.Value:
			default:
				b.Reset()
				_ = enc.Encode(v)
				v = json.Value(bytes.Clone(b.Bytes()))
			}
			arr[i] = v
		}
		return arr, types.Array(types.JSON()), nil
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
		if v0 == nil {
			return nil, types.Boolean(), nil
		}
		v1, t1, err := eval(p.args[1], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v1 == nil {
			return nil, types.Boolean(), nil
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
		if v0 != nil && v0.(bool) {
			return eval(p.args[1], properties, layouts, purpose)
		}
		if len(p.args) == 3 {
			return eval(p.args[2], properties, layouts, purpose)
		}
		return nil, types.JSON(), nil
	case "initcap":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.Text(), nil
		}
		return strings.Title(v.(string)), types.Text(), nil
	case "len":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v2, ok := v.(json.Value); ok {
			dec := stdjson.NewDecoder(bytes.NewReader(v2))
			dec.UseNumber()
			v = nil
			_ = dec.Decode(&v)
		}
		var length int
		switch v := v.(type) {
		case nil:
		case bool:
			length = 5
			if v {
				length = 4
			}
		case int:
			length = digitCountInt(int64(v))
		case uint:
			length = digitCountUint(uint64(v))
		case float64:
			bitSize := 64
			if t := typesOf(p.args[0]); t.Kind() == types.FloatKind && t.BitSize() == 32 {
				bitSize = 32
			}
			length = len(strconv.FormatFloat(v, 'g', -1, bitSize))
		case decimal.Decimal:
			length = len(v.String())
		case string:
			length = utf8.RuneCountInString(v)
		case time.Time:
			t := typesOf(p.args[0])
			switch t.Kind() {
			case types.DateTimeKind:
				length = 20
			case types.DateKind:
				length = 10
			case types.TimeKind:
				length = len(v.Format("15:04:05.999999999"))
			}
		case []any:
			length = len(v)
		case map[string]any:
			length = len(v)
		case stdjson.Number:
			length = len(v)
		}
		return length, types.Int(32), nil
	case "lower":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.Text(), nil
		}
		return strings.ToLower(v.(string)), types.Text(), nil
	case "ltrim":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.Text(), nil
		}
		return strings.TrimLeftFunc(v.(string), unicode.IsSpace), types.Text(), nil
	case "ne":
		v0, t0, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v0 == nil {
			return nil, types.Boolean(), nil
		}
		v1, t1, err := eval(p.args[1], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v1 == nil {
			return nil, types.Boolean(), nil
		}
		if !types.Equal(t0, t1) {
			v0, err = convert(v0, t0, t1, true, layouts, purpose)
			if err != nil {
				if err == errInvalidConversion {
					return true, types.Boolean(), nil
				}
				return nil, types.Type{}, err
			}
		}
		return !reflect.DeepEqual(v0, v1), types.Boolean(), nil
	case "not":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.Boolean(), nil
		}
		return !v.(bool), types.Boolean(), nil
	case "or":
		var null bool
		for _, arg := range p.args {
			v, _, err := eval(arg, properties, layouts, purpose)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v == nil {
				null = true
				continue
			}
			if v.(bool) {
				return true, types.Boolean(), nil
			}
		}
		if null {
			return nil, types.Boolean(), nil
		}
		return false, types.Boolean(), nil
	case "rtrim":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.Text(), nil
		}
		return strings.TrimRightFunc(v.(string), unicode.IsSpace), types.Text(), nil
	case "substring":
		v0, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v0 == nil {
			return nil, types.Text(), nil
		}
		v1, _, err := eval(p.args[1], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v1 == nil {
			return nil, types.Text(), nil
		}
		start := v1.(int)
		if start < 1 {
			start = 1
		}
		length := -1
		if len(p.args) == 3 {
			v2, _, err := eval(p.args[2], properties, layouts, purpose)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v2 == nil {
				return nil, types.Text(), nil
			}
			length = v2.(int)
			if length < 0 {
				return nil, types.Type{}, errors.New("negative substring length is not allowed")
			}
		}
		return substring(v0.(string), start, length), types.Text(), nil
	case "trim":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.Text(), nil
		}
		return strings.TrimSpace(v.(string)), types.Text(), nil
	case "upper":
		v, _, err := eval(p.args[0], properties, layouts, purpose)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.Text(), nil
		}
		return strings.ToUpper(v.(string)), types.Text(), nil
	}
	panic(fmt.Errorf("unknown function %q", p.path[0]))
}

// substring returns a substring of s starting from the rune at position
// start-1, with start > 0, for a length in rune of length. If length is
// negative, it returns all the runes from s to the end of the string.
func substring(s string, start, length int) string {
	if length == 0 {
		return ""
	}
	n := 0
	var i int
	if start > 1 {
		for i = range s {
			n += 1
			if n == start {
				break
			}
		}
		if n < start {
			return ""
		}
	}
	s = s[i:]
	if length < 0 {
		return s
	}
	n = 0
	var r rune
	for i, r = range s {
		n += 1
		if n == length {
			break
		}
	}
	i += utf8.RuneLen(r)
	return s[:i]
}

// valueOf returns the value at the specified path in properties. It returns nil
// if the path does not exist, including keys in a map and properties of a JSON
// object.
//
// For non-object JSON values, accessing a key returns nil if the key is
// followed by "?"; otherwise, it returns an error.
func valueOf(path path, properties map[string]any) (any, error) {
	var v any
	var ok bool
	last := len(path) - 1
	var i int
	for i = 0; i < len(path); i++ {
		name := path[i]
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
		if i == last {
			return v, nil
		}
		properties, ok = v.(map[string]any)
		if !ok {
			break
		}
	}
	v2 := v.(json.Value)
	for i++; i < len(path); i++ {
		name := path[i]
		if !v2.IsObject() {
			if name[len(name)-1] == '?' {
				return nil, nil
			}
			var t string
			switch v2.Kind() {
			case json.Null:
				t = "JSON null"
			case json.False, json.True:
				t = "a JSON boolean"
			case json.Number:
				t = "a JSON number"
			case json.String:
				t = "a JSON string"
			default:
				t = "a JSON array"
			}
			return nil, &invalidConversionError{msg: fmt.Sprintf("invalid %s: %s is not a JSON object, it is %s", path[:i+1], path[:i], t)}
		}
		if name[0] == ':' {
			name = name[1:]
			if n := len(name) - 1; name[n] == '?' {
				name = name[:n]
			}
		}
		if name[0] == '[' {
			name = name[1 : len(name)-1]
		}
		v2, ok = v2.Lookup(name)
		if !ok {
			return nil, nil
		}
		if i == last {
			return v2, nil
		}
	}
	return v, nil
}
