// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// encodeSorted encodes JSON values with their object keys sorted.
// It is set to true during tests to ensure deterministic output.
var encodeSorted = false

// Eval evaluates the expression using attributes, which must conform to the
// source schema, and returns the value and its actual type. A nil result may
// have an invalid type when no result type is known.
//
// Literals have already been converted by Compile. Dynamic arguments are
// converted when required by the function receiving them. Calls to if and
// coalesce return their selected argument with its actual type. Eval does not
// convert the final result to Compile's destination type; Mapping.Transform
// does that.
//
// If a property transformation fails, Eval returns a TransformationError.
// If an array element cannot be converted to its contextual type, it returns a
// ValidationError.
func (expr *Expression) Eval(attributes map[string]any) (any, types.Type, error) {
	v, st, err := eval(expr.parts, expr.source, attributes)
	if err != nil {
		if err == errInvalidConversion {
			return nil, types.Type{}, TransformationError{err.Error()}
		}
		return nil, types.Type{}, err
	}
	return v, st, nil
}

// appendAsString appends v to b after converting it to a string.
// Calling appendAsString(b, v, t) is equivalent to calling
// convert(v, t, types.String(), false, false, nil, None) and appending the
// result to b.
func appendAsString(b []byte, v any, t types.Type) ([]byte, error) {
	if v == nil {
		return b, nil
	}
	if s, ok := v.(string); ok {
		return append(b, s...), nil
	}
	switch t.Kind() {
	case types.BooleanKind:
		return strconv.AppendBool(b, v.(bool)), nil
	case types.IntKind:
		if t.IsUnsigned() {
			return strconv.AppendUint(b, uint64(v.(uint)), 10), nil
		}
		return strconv.AppendInt(b, int64(v.(int)), 10), nil
	case types.FloatKind:
		return strconv.AppendFloat(b, v.(float64), 'g', -1, t.BitSize()), nil
	case types.DecimalKind:
		return v.(decimal.Decimal).Append(b), nil
	case types.DateTimeKind:
		return v.(time.Time).AppendFormat(b, time.RFC3339Nano), nil
	case types.DateKind:
		return v.(time.Time).AppendFormat(b, time.DateOnly), nil
	case types.TimeKind:
		return v.(time.Time).AppendFormat(b, "15:04:05.999999999"), nil
	case types.YearKind:
		return strconv.AppendInt(b, int64(v.(int)), 10), nil
	case types.JSONKind:
		v := v.(json.Value)
		switch v.Kind() {
		case json.Array, json.Object:
		case json.String:
			return v.AppendUnquote(b), nil
		default:
			return append(b, v...), nil
		}
	}
	return b, errInvalidConversion
}

// digitCountInt returns the number of decimal digits in n, including the sign
// for negative numbers.
func digitCountInt(n int64) int {
	return len(strconv.FormatInt(n, 10))
}

// digitCountUint returns the number of decimal digits in n.
func digitCountUint(n uint64) int {
	return len(strconv.FormatUint(n, 10))
}

// eval evaluates the expression and returns its value and type. source is the
// source code of the expression, used in error messages, and attributes are the
// attributes.
//
// Errors may include errInvalidConversion, TransformationError, and
// ValidationError.
func eval(expression []part, source string, attributes map[string]any) (any, types.Type, error) {

	// Evaluate the most common cases that do not require a buffer.
	if len(expression) == 1 {
		p := expression[0]
		if p.path.elements == nil {
			return p.value, p.typ, nil
		}
		if p.value == nil {
			if len(p.path.elements) == 1 {
				if p.args == nil {
					v, ok := attributes[p.path.elements[0]]
					if !ok {
						return nil, types.Type{}, nil
					}
					return v, p.typ, nil
				}
				return evalCall(p, source, attributes)
			}
			v, err := valueOf(p.path, attributes)
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
		if p.path.elements == nil {
			continue
		}
		if p.args == nil {
			v, err = valueOf(p.path, attributes)
			if err != nil {
				return nil, types.Type{}, err
			}
			vt = p.typ
		} else {
			v, vt, err = evalCall(p, source, attributes)
			if err != nil {
				return nil, types.Type{}, err
			}
		}
		buf, err = appendAsString(buf, v, vt)
		if err != nil {
			return nil, types.Type{}, err
		}
	}

	return string(buf), types.String(), nil
}

// evalCall evaluates p representing a function call, and returns its value and
// type. source is the source code of the expression, used in error messages,
// and attributes are the attributes.
//
// Errors may include errInvalidConversion, TransformationError, and
// ValidationError.
func evalCall(p part, source string, attributes map[string]any) (any, types.Type, error) {
	switch name := p.path.elements[0]; name {
	case "and":
		var null bool
		for _, arg := range p.args {
			v, vt, err := eval(arg, source, attributes)
			if err == nil && v != nil && vt.Kind() != types.BooleanKind {
				v, err = convert(v, vt, types.Boolean(), true, false, nil, None)
				if err != nil {
					err = errBooleanConversion("and", code(source, arg...), v, vt)
				}
			}
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
		arr := make([]any, len(p.args))
		et := p.typ.Elem()
		for i, arg := range p.args {
			v, vt, err := eval(arg, source, attributes)
			if err != nil {
				return nil, types.Type{}, err
			}
			arr[i], err = convert(v, vt, et, false, false, nil, None)
			if err != nil {
				// The element type comes from the destination schema, so a conversion
				// failure is a validation error even though it occurs during evaluation.
				return nil, types.Type{}, errValidationConversion(err, code(source, arg...), et)
			}
		}
		return arr, p.typ, nil
	case "coalesce":
		for _, arg := range p.args {
			v, vt, err := eval(arg, source, attributes)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v != nil {
				return v, vt, nil
			}
		}
		return nil, p.typ, nil
	case "eq", "ne":
		v0, t0, err := eval(p.args[0], source, attributes)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v0 == nil {
			return nil, types.Boolean(), nil
		}
		v1, t1, err := eval(p.args[1], source, attributes)
		if err != nil {
			return nil, types.Type{}, err
		}
		if v1 == nil {
			return nil, types.Boolean(), nil
		}
		equal := equalValues(v0, v1, t0, t1)
		if p.path.elements[0] == "ne" {
			equal = !equal
		}
		return equal, types.Boolean(), nil
	case "if":
		v0, vt0, err := eval(p.args[0], source, attributes)
		if err == nil && v0 != nil && vt0.Kind() != types.BooleanKind {
			v0, err = convert(v0, vt0, types.Boolean(), true, false, nil, None)
			if err != nil {
				err = errBooleanConversion("if", code(source, p.args[0]...), v0, vt0)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v0 != nil && v0.(bool) {
			return eval(p.args[1], source, attributes)
		}
		if len(p.args) == 3 {
			return eval(p.args[2], source, attributes)
		}
		return nil, types.JSON(), nil
	case "initcap":
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil && vt.Kind() != types.StringKind {
			v, err = convert(v, vt, types.String(), true, false, nil, None)
			if err != nil {
				err = errStringConversion("initcap", code(source, p.args[0]...), v)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.String(), nil
		}
		// Keep strings.Title to preserve the existing word boundaries.
		return strings.Title(strings.ToLower(v.(string))), types.String(), nil
	case "json_parse":
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil {
			k := vt.Kind()
			if k != types.StringKind && k != types.JSONKind {
				return nil, types.Type{}, fmt.Errorf(
					"«%s» has type %s and cannot be passed as a string value to the «json_parse» function", code(source, p.args[0]...), k)
			}
			if k == types.JSONKind {
				value := v.(json.Value)
				if k := value.Kind(); k != json.String {
					return nil, types.Type{}, fmt.Errorf(
						"«%s» is a JSON %s and cannot be passed as a string value to the «json_parse» function", code(source, p.args[0]...), k)
				}
				v = value.String()
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.JSON(), nil
		}
		jv := json.Value(v.(string))
		if !json.Valid(jv) {
			err = fmt.Errorf("«%s» cannot be parsed by «json_parse» because it is not valid JSON", code(source, p.args[0]...))
			return nil, types.Type{}, err
		}
		return jv, types.JSON(), nil
	case "len":
		v, vt, err := eval(p.args[0], source, attributes)
		if err != nil {
			return nil, types.Type{}, err
		}
		var length int
		switch v := v.(type) {
		case nil:
		case string:
			length = utf8.RuneCountInString(v)
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
			if vt.Kind() == types.FloatKind && vt.BitSize() == 32 {
				bitSize = 32
			}
			length = len(strconv.FormatFloat(v, 'g', -1, bitSize))
		case decimal.Decimal:
			length = len(v.String())
		case time.Time:
			switch vt.Kind() {
			case types.DateTimeKind:
				length = len(v.Format(time.RFC3339Nano))
			case types.DateKind:
				length = 10
			case types.TimeKind:
				length = len(v.Format("15:04:05.999999999"))
			}
		case json.Value:
			switch v.Kind() {
			case json.Null:
			case json.True, json.False, json.Number:
				length = len(json.TrimSpace(v))
			case json.String:
				length = utf8.RuneCountInString(v.String())
			case json.Object:
				length = v.NumProperty()
			case json.Array:
				length = v.NumElement()
			}
		case []any:
			length = len(v)
		case map[string]any:
			length = len(v)
		}
		if length > types.MaxInt32 {
			msg := fmt.Sprintf("length of «%s» exceeds int(32) maximum %d", code(source, p.args[0]...), types.MaxInt32)
			return nil, types.Type{}, TransformationError{msg}
		}
		return length, types.Int(32), nil
	case "lower":
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil && vt.Kind() != types.StringKind {
			v, err = convert(v, vt, types.String(), true, false, nil, None)
			if err != nil {
				if err != nil {
					err = errStringConversion("lower", code(source, p.args[0]...), v)
				}
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.String(), nil
		}
		return strings.ToLower(v.(string)), types.String(), nil
	case "ltrim":
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil && vt.Kind() != types.StringKind {
			v, err = convert(v, vt, types.String(), true, false, nil, None)
			if err != nil {
				err = errStringConversion("ltrim", code(source, p.args[0]...), v)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.String(), nil
		}
		return strings.TrimLeftFunc(v.(string), unicode.IsSpace), types.String(), nil
	case "map":
		m := make(map[string]any, len(p.args)/2)
		for i := 0; i < len(p.args); i += 2 {
			key := p.args[i][0].value.(string)

			valExpr := p.args[i+1]
			v, vt, err := eval(valExpr, source, attributes)
			if err != nil {
				return nil, types.Type{}, err
			}

			if v == nil && len(valExpr) == 1 && valExpr[0].value == nil && valExpr[0].args == nil &&
				valExpr[0].path.elements != nil {
				// Single property path with no value: omit the key.
				continue
			}

			switch v.(type) {
			case nil:
				v = json.Value("null")
			case json.Value:
			default:
				v = formatJSONTimes(v, vt)
				if encodeSorted {
					var b json.Buffer
					err = b.EncodeSorted(v)
					if err != nil {
						return nil, types.Type{}, err
					}
					v, err = b.Value()
				} else {
					v, err = json.Marshal(v)
				}
				if err != nil {
					return nil, types.Type{}, err
				}
			}
			m[key] = v
		}
		return m, types.Map(types.JSON()), nil
	case "not":
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil && vt.Kind() != types.BooleanKind {
			v, err = convert(v, vt, types.Boolean(), true, false, nil, None)
			if err != nil {
				err = errBooleanConversion("not", code(source, p.args[0]...), v, vt)
			}
		}
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
			v, vt, err := eval(arg, source, attributes)
			if err == nil && v != nil && vt.Kind() != types.BooleanKind {
				v, err = convert(v, vt, types.Boolean(), true, false, nil, None)
				if err != nil {
					err = errBooleanConversion("or", code(source, arg...), v, vt)
				}
			}
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
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil && vt.Kind() != types.StringKind {
			v, err = convert(v, vt, types.String(), true, false, nil, None)
			if err != nil {
				err = errStringConversion("rtrim", code(source, p.args[0]...), v)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.String(), nil
		}
		return strings.TrimRightFunc(v.(string), unicode.IsSpace), types.String(), nil
	case "substring":
		v0, vt0, err := eval(p.args[0], source, attributes)
		if err == nil && v0 != nil && vt0.Kind() != types.StringKind {
			v0, err = convert(v0, vt0, types.String(), true, false, nil, None)
			if err != nil {
				err = errStringConversion("substring", code(source, p.args[0]...), v0)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v0 == nil {
			return nil, types.String(), nil
		}
		v1, vt1, err := eval(p.args[1], source, attributes)
		if err == nil && v1 != nil &&
			(vt1.Kind() != types.IntKind || vt1.BitSize() > 32 || vt1.IsUnsigned()) {
			v1, err = convert(v1, vt1, types.Int(32), true, false, nil, None)
			if err != nil {
				err = errInt32Conversion("substring", code(source, p.args[1]...), v1, vt1)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v1 == nil {
			return nil, types.String(), nil
		}
		start := max(v1.(int), 1)
		length := -1
		if len(p.args) == 3 {
			v2, vt2, err := eval(p.args[2], source, attributes)
			if err == nil && v2 != nil &&
				(vt2.Kind() != types.IntKind || vt2.BitSize() > 32 || vt2.IsUnsigned()) {
				v2, err = convert(v2, vt2, types.Int(32), true, false, nil, None)
				if err != nil {
					err = errInt32Conversion("substring", code(source, p.args[2]...), v2, vt2)
				}
			}
			if err != nil {
				return nil, types.Type{}, err
			}
			if v2 == nil {
				return nil, types.String(), nil
			}
			length = v2.(int)
			if length < 0 {
				return nil, types.Type{}, TransformationError{"substring: negative substring length is not allowed"}
			}
		}
		return substring(v0.(string), start, length), types.String(), nil
	case "trim":
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil && vt.Kind() != types.StringKind {
			v, err = convert(v, vt, types.String(), true, false, nil, None)
			if err != nil {
				err = errStringConversion("trim", code(source, p.args[0]...), v)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.String(), nil
		}
		return strings.TrimSpace(v.(string)), types.String(), nil
	case "upper":
		v, vt, err := eval(p.args[0], source, attributes)
		if err == nil && v != nil && vt.Kind() != types.StringKind {
			v, err = convert(v, vt, types.String(), true, false, nil, None)
			if err != nil {
				err = errStringConversion("upper", code(source, p.args[0]...), v)
			}
		}
		if err != nil {
			return nil, types.Type{}, err
		}
		if v == nil {
			return nil, types.String(), nil
		}
		return strings.ToUpper(v.(string)), types.String(), nil
	}
	panic(fmt.Errorf("unknown function %q", p.path.elements[0]))
}

// substring returns the substring of s starting at rune start-1. If length is
// non-negative, it returns at most length runes; otherwise, it returns the
// remainder of the string.
func substring(s string, start, length int) string {
	if s == "" || length == 0 {
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
	for i = range s {
		n += 1
		if n == length {
			break
		}
	}
	// Decode only the last rune: range consumes one byte for malformed UTF-8.
	_, size := utf8.DecodeRuneInString(s[i:])
	return s[:i+size]
}

// errBooleanConversion returns an error explaining the failure that occurred
// when converting v, of type t, to a nullable boolean type while passing it
// to the fn function. code is the source code of the passed expression.
func errBooleanConversion(fn string, code string, v any, t types.Type) error {
	switch t.Kind() {
	case types.StringKind:
		return fmt.Errorf("«%s» (type string) does not represent a boolean when passed to the «%s» function", code, fn)
	case types.JSONKind:
		k := v.(json.Value).Kind()
		return fmt.Errorf("«%s», of type JSON %s, cannot be passed as boolean to the «%s» function", code, k, fn)
	}
	return fmt.Errorf("«%s», of type %s, cannot be passed as boolean to the «%s» function", code, t.Kind(), fn)
}

// errInt32Conversion returns an error explaining the failure that occurred when
// converting v, of type t, to a nullable int(32) type while passing it to the
// fn function. code is the source code of the passed expression.
func errInt32Conversion(fn string, code string, v any, t types.Type) error {
	switch t.Kind() {
	case types.IntKind, types.FloatKind, types.DecimalKind:
		return fmt.Errorf("«%s», with a value of %v, cannot be passed as a 32-bit int to the «%s» function",
			code, v, fn)
	case types.JSONKind:
		k := v.(json.Value).Kind()
		if k == json.Number {
			return fmt.Errorf("«%s», with a value of %s, cannot be passed as a 32-bit int to the «%s» function", code, v, fn)
		}
		return fmt.Errorf("«%s», of type JSON %s, cannot be passed as an int to the «%s» function", code, k, fn)
	}
	return fmt.Errorf("«%s», of type %s, cannot be passed as int to the «%s» function", code, t.Kind(), fn)
}

// errStringConversion returns an error explaining the failure that occurred
// when converting v to a nullable string type while passing it to the fn
// function. code is the source code of the passed expression.
//
// Note that the conversion fails at execution time only if v is a JSON object
// or JSON array.
func errStringConversion(fn string, code string, v any) error {
	k := v.(json.Value).Kind()
	return fmt.Errorf("«%s» (a JSON %s) cannot be converted to a string value to be passed to the «%s» function", code, k, fn)
}

// errValidationConversion describes a failed conversion to dt as a
// ValidationError. code identifies the expression; Mapping.Transform adds the
// destination path. convert does not retain the failing type inside a
// container, so constraints are described only when they can be read from dt
// itself.
func errValidationConversion(err error, code string, dt types.Type) ValidationError {

	msg := fmt.Sprintf("«%s» is not convertible to the «%s» type", code, dt)
	switch err {
	case errRangeConversion:
		msg = fmt.Sprintf("number «%s» is not a «%s» value", code, dt)
	case errMinConversion:
		var n any
		switch dt.Kind() {
		case types.IntKind:
			if dt.IsUnsigned() {
				n, _ = dt.UnsignedRange()
			} else {
				n, _ = dt.IntRange()
			}
		case types.FloatKind:
			n, _ = dt.FloatRange()
		case types.DecimalKind:
			n, _ = dt.DecimalRange()
		}
		if n != nil {
			msg = fmt.Sprintf("number «%s» is less than %v", code, n)
		}
	case errMaxConversion:
		var n any
		switch dt.Kind() {
		case types.IntKind:
			if dt.IsUnsigned() {
				_, n = dt.UnsignedRange()
			} else {
				_, n = dt.IntRange()
			}
		case types.FloatKind:
			_, n = dt.FloatRange()
		case types.DecimalKind:
			_, n = dt.DecimalRange()
		}
		if n != nil {
			msg = fmt.Sprintf("number «%s» is greater than %v", code, n)
		}
	case errParseConversion:
		var to string
		switch dt.Kind() {
		case types.DateTimeKind:
			to = "a date time in ISO 8601 format"
		case types.DateKind:
			to = "a date in ISO 8601 format"
		case types.TimeKind:
			to = "a time in ISO 8601 format"
		case types.UUIDKind:
			to = "a UUID"
		case types.IPKind:
			to = "an IP address"
		}
		if to != "" {
			msg = fmt.Sprintf("«%s» is not parsable as %s", code, to)
		}
	case errYearRangeConversion:
		msg = fmt.Sprintf("year of «%s» is not in range [1,9999]", code)
	case errUnixNanoConversion:
		msg = fmt.Sprintf("«%s» is outside the range supported by the «unixnano» datetime format", code)
	case errEnumConversion:
		msg = fmt.Sprintf("«%s» is not one of the allowed values", code)
	case errPatternConversion:
		if dt.Kind() == types.StringKind {
			msg = fmt.Sprintf("«%s» does not match «/%s/»", code, dt.Pattern())
		}
	case errMaxBytesConversion:
		if dt.Kind() == types.StringKind {
			n, _ := dt.MaxBytes()
			msg = fmt.Sprintf("«%s» exceeds the %d-byte limit", code, n)
		}
	case errMaxLengthConversion:
		if dt.Kind() == types.StringKind {
			n, _ := dt.MaxLength()
			msg = fmt.Sprintf("«%s» exceeds the %d-char limit", code, n)
		}
	}

	return ValidationError{msg}
}

// valueOf returns the value at the specified path in attributes. It returns nil
// if the path does not exist, including keys in a map and properties of a JSON
// object, or if an intermediate value is nil.
//
// For non-object JSON values, accessing a key returns nil if the key is
// optional; otherwise, it returns an error.
//
// It returns a TransformationError if traversal encounters a non-object value,
// except for optional keys in non-object JSON values as described above.
func valueOf(path path, attributes map[string]any) (any, error) {
	last := len(path.elements) - 1
	var i int
	for i = 0; i < len(path.elements); i++ {
		name := path.elements[i]
		v, ok := attributes[name]
		if !ok || v == nil {
			return nil, nil
		}
		if i == last {
			return v, nil
		}
		switch v := v.(type) {
		case map[string]any:
			attributes = v
		case json.Value:
			i += 1
			v, err := v.Lookup(path.elements[i:])
			if err != nil {
				e := err.(json.NotExistError)
				if e.Kind == json.Object || path.decorators[i+e.Index].optional() {
					return nil, nil
				}
				msg := fmt.Sprintf("invalid %s: %s is not JSON object, it is %s",
					path.slice(0, i+e.Index+1), path.slice(0, i+e.Index), e.Kind)
				return nil, TransformationError{msg}
			}
			return v, nil
		default:
			msg := fmt.Sprintf("invalid %s: %s is not an object", path.slice(0, i+2), path.slice(0, i+1))
			return nil, TransformationError{msg}
		}
	}
	panic("unreachable code")
}
