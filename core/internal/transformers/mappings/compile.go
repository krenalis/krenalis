// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

var jsonArrayType = types.Array(types.JSON())

// Expression represents a mapping expression used to transform data from a
// source to a destination. An Expression can contain strings, numbers, true,
// false, null, property paths and function calls.
type Expression struct {
	parts  []part // parts.
	source string // source code used for error messages.
}

// Compile parses the given expression and returns an Expression that can be
// used for evaluation, along with the property paths referenced in the
// expression.
//
// The schema defines the structure of the paths used in the expression, while
// dt specifies the destination type.
//
// If schema is invalid, the expression will be compiled without path resolution.
func Compile(expr string, schema, dt types.Type) (*Expression, []string, error) {
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, nil, errors.New("schema is not an object")
	}
	if !dt.Valid() {
		return nil, nil, errors.New("destination type is the invalid type")
	}
	parts, src, err := parse(expr, 0, len(expr))
	if err != nil {
		return nil, nil, err
	}
	if src != "" {
		return nil, nil, fmt.Errorf("unexpected character %v", strconv.QuoteRuneToGraphic(rune(src[0])))
	}
	if parts == nil {
		return nil, nil, errors.New("expression is empty")
	}
	props := map[string]struct{}{}
	err = typeCheck(parts, schema, dt, true, props)
	if err != nil {
		return nil, nil, err
	}
	expression := &Expression{
		parts:  parts,
		source: expr,
	}
	var properties []string
	if len(props) > 0 {
		properties = make([]string, len(props))
		i := 0
		for name := range props {
			properties[i] = name
			i++
		}
		slices.Sort(properties)
	}
	return expression, properties, nil
}

// checkAnd type checks a call to 'and' with the given arguments.
func checkAnd(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) < 2 {
		return types.Type{}, errors.New("'and' function requires at least two argument")
	}
	booleanType := types.Boolean()
	for _, arg := range args {
		err := typeCheck(arg, schema, booleanType, true, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	return booleanType, nil
}

// checkArray type checks a call to 'array' with the given arguments.
func checkArray(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	// Ensure that all arguments can be converted to the element type of the destination array.
	// If the destination type (dt) is not an array, no checks are performed,
	// and it is left to the caller to fail later.
	et := types.JSON()
	if dt.Kind() == types.ArrayKind {
		et = dt.Elem()
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, et, false, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	return jsonArrayType, nil
}

// checkCoalesce type checks a call to 'coalesce' with the given arguments.
func checkCoalesce(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) < 1 {
		return types.Type{}, errors.New("'coalesce' function requires at least one argument")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, dt, true, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	return types.Type{}, nil
}

// checkEq type checks a call to 'eq' with the given arguments.
func checkEq(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) != 2 {
		return types.Type{}, errors.New("'eq' function requires two arguments")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, types.Type{}, true, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	t0 := typeOf(args[0])
	t1 := typeOf(args[1])
	if !t0.Valid() || !t1.Valid() {
		return types.Boolean(), nil
	}
	if !convertibleTo(t0, t1) {
		return types.Type{}, errors.New("first argument of 'eq(...)' is not convertible to the type of the second")
	}
	if !convertibleTo(t1, t0) {
		return types.Type{}, errors.New("second argument of 'eq(...)' is not convertible to the type of the first")
	}
	return types.Boolean(), nil
}

// checkIf type checks a call to 'if' with the given arguments.
func checkIf(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n < 2 || n > 3 {
		return types.Type{}, errors.New("'if' function requires either two or three arguments")
	}
	err := typeCheck(args[0], schema, types.Boolean(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	err = typeCheck(args[1], schema, dt, nullable, attributes)
	if err != nil {
		return types.Type{}, err
	}
	if n == 3 {
		err = typeCheck(args[2], schema, dt, nullable, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	return dt, nil
}

// checkInitCap type checks a call to 'initcap' with the given arguments.
func checkInitCap(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'initcap' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// checkJSONParse type checks a call to 'json_parse' with the given arguments.
func checkJSONParse(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) != 1 {
		return types.Type{}, errors.New("'json_parse' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return types.JSON(), nil
}

// checkLen type checks a call to 'len' with the given arguments.
func checkLen(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) != 1 {
		return types.Type{}, errors.New("'len' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.Type{}, true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return types.Int(32), nil
}

// checkLower type checks a call to 'lower' with the given arguments.
func checkLower(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'lower' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// checkLTrim type checks a call to 'ltrim' with the given arguments.
func checkLTrim(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) != 1 {
		return types.Type{}, errors.New("'ltrim' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return types.String(), nil
}

// checkMap type checks a call to 'map' with the given arguments.
func checkMap(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args)%2 != 0 {
		return types.Type{}, errors.New("'map' function requires an even number of arguments")
	}

	keys := make(map[string]struct{}, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if len(args[i]) != 1 || args[i][0].path.elements != nil {
			return types.Type{}, errors.New("'map' key is not constant")
		}
		if args[i][0].typ.Kind() != types.StringKind {
			return types.Type{}, errors.New("'map' key is not string")
		}

		if err := typeCheck(args[i], schema, types.String(), false, attributes); err != nil {
			return types.Type{}, err
		}
		if err := typeCheck(args[i+1], schema, types.JSON(), false, attributes); err != nil {
			return types.Type{}, err
		}

		key := args[i][0].value.(string)
		if _, ok := keys[key]; ok {
			return types.Type{}, errors.New("duplicate key in 'map' function")
		}
		keys[key] = struct{}{}
	}

	return types.Map(types.JSON()), nil
}

// checkNe type checks a call to 'ne' with the given arguments.
func checkNe(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) != 2 {
		return types.Type{}, errors.New("'ne' function requires two arguments")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, types.Type{}, true, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	t0 := typeOf(args[0])
	t1 := typeOf(args[1])
	if !t0.Valid() || !t1.Valid() {
		return types.Boolean(), nil
	}
	if !convertibleTo(t0, t1) {
		return types.Type{}, errors.New("first argument of 'ne(...)' is not convertible to the type of the second")
	}
	if !convertibleTo(t1, t0) {
		return types.Type{}, errors.New("second argument of 'ne(...)' is not convertible to the type of the first")
	}
	return types.Boolean(), nil
}

// checkNot type checks a call to 'not' with the given arguments.
func checkNot(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'not' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.Boolean(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// checkOr type checks a call to 'or' with the given arguments.
func checkOr(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) < 2 {
		return types.Type{}, errors.New("'or' function requires at least two argument")
	}
	booleanType := types.Boolean()
	for _, arg := range args {
		err := typeCheck(arg, schema, booleanType, true, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	return booleanType, nil
}

// checkRTrim type checks a call to 'rtrim' with the given arguments.
func checkRTrim(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) != 1 {
		return types.Type{}, errors.New("'rtrim' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return types.String(), nil
}

// checkSubstring type checks a call to 'substring' with the given arguments.
func checkSubstring(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n < 2 || n > 3 {
		return types.Type{}, errors.New("'substring' function requires two or three arguments")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	err = typeCheck(args[1], schema, types.Int(32), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	if n == 3 {
		err = typeCheck(args[2], schema, types.Int(32), true, attributes)
		if err != nil {
			return types.Type{}, err
		}
	}
	return dt, nil
}

// checkTrim type checks a call to 'trim' with the given arguments.
func checkTrim(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	if len(args) != 1 {
		return types.Type{}, errors.New("'trim' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return types.String(), nil
}

// checkUpper type checks a call to 'upper' with the given arguments.
func checkUpper(args [][]part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'upper' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.String(), true, attributes)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// typeCheck type checks the expression expr. schema is the schema of the
// properties in the expression, dt is the destination type, and nullable
// indicates whether that value can be nil. An invalid schema can be passed to
// type check an expression without paths.
func typeCheck(expr []part, schema, dt types.Type, nullable bool, attributes map[string]struct{}) error {

	typ := dt
	n := nullable
	concatenate := len(expr) > 1 || expr[0].value != nil
	if concatenate {
		typ = types.String()
		n = true
	}

	for i, p := range expr {
		if p.path.elements == nil {
			continue
		}
		// Check the path.
		if p.args == nil {
			var b strings.Builder
			t := schema
			for j := 0; j < len(p.path.elements); j++ {
				name := p.path.elements[j]
				decorators := p.path.decorators[j]
				switch t.Kind() {
				case types.JSONKind:
				case types.ObjectKind, types.InvalidKind:
					if decorators.optional() {
						return fmt.Errorf("invalid %s: operator '?' can be used only with json", p.path.slice(0, j+1))
					}
					if decorators.indexing() {
						if !types.IsValidPropertyName(name) {
							return fmt.Errorf("invalid %s: %q is not a valid property name", p.path.slice(0, j+1), name)
						}
					}
					var property types.Property
					var ok bool
					if t.Valid() {
						property, ok = t.Properties().ByName(name)
					}
					if !ok {
						msg := fmt.Sprintf("property %q does not exist", name)
						if j > 0 {
							msg = fmt.Sprintf("invalid %s: %s", p.path.slice(0, j+1), msg)
						}
						return errors.New(msg)
					}
					if b.Len() > 0 {
						b.WriteByte('.')
					}
					b.WriteString(name)
					t = property.Type
				case types.MapKind:
					if p.path.decorators[j].optional() {
						return fmt.Errorf("invalid %s: operator '?' can be used only with json", p.path.slice(0, j+1))
					}
					t = t.Elem()
				default:
					return fmt.Errorf("invalid %s: %s (type %s) cannot have properties or keys", p.path.slice(0, j+1), p.path.slice(0, j), t)
				}
			}
			if concatenate && !convertibleTo(t, types.String()) {
				return fmt.Errorf("cannot convert %s (type %s) to string", p.path, t)
			}
			attributes[b.String()] = struct{}{}
			expr[i].typ = t
			continue
		}
		// Check the function call
		expr[i].path.decorators = nil
		var err error
		switch p.path.elements[0] {
		case "and":
			expr[i].typ, err = checkAnd(p.args, schema, typ, n, attributes)
		case "array":
			expr[i].typ, err = checkArray(p.args, schema, typ, n, attributes)
		case "coalesce":
			expr[i].typ, err = checkCoalesce(p.args, schema, typ, n, attributes)
		case "eq":
			expr[i].typ, err = checkEq(p.args, schema, typ, n, attributes)
		case "if":
			expr[i].typ, err = checkIf(p.args, schema, typ, n, attributes)
		case "initcap":
			expr[i].typ, err = checkInitCap(p.args, schema, typ, n, attributes)
		case "json_parse":
			expr[i].typ, err = checkJSONParse(p.args, schema, typ, n, attributes)
		case "len":
			expr[i].typ, err = checkLen(p.args, schema, typ, n, attributes)
		case "lower":
			expr[i].typ, err = checkLower(p.args, schema, typ, n, attributes)
		case "ltrim":
			expr[i].typ, err = checkLTrim(p.args, schema, typ, n, attributes)
		case "map":
			expr[i].typ, err = checkMap(p.args, schema, typ, n, attributes)
		case "ne":
			expr[i].typ, err = checkNe(p.args, schema, typ, n, attributes)
		case "not":
			expr[i].typ, err = checkNot(p.args, schema, typ, n, attributes)
		case "or":
			expr[i].typ, err = checkOr(p.args, schema, typ, n, attributes)
		case "rtrim":
			expr[i].typ, err = checkRTrim(p.args, schema, typ, n, attributes)
		case "substring":
			expr[i].typ, err = checkSubstring(p.args, schema, typ, n, attributes)
		case "trim":
			expr[i].typ, err = checkTrim(p.args, schema, typ, n, attributes)
		case "upper":
			expr[i].typ, err = checkUpper(p.args, schema, typ, n, attributes)
		default:
			panic(fmt.Errorf("unknown function %q", p.path.elements[0]))
		}
		if err != nil {
			return err
		}
		if concatenate {
			if st := expr[i].typ; st.Valid() && !convertibleTo(st, types.String()) {
				return fmt.Errorf("cannot convert %s(...) (type %s) to string", p.path, st)
			}
		}
	}

	if dt.Valid() {
		return asType(expr, dt, nullable)
	}
	return nil
}

// asType reports whether expr can be converted to type dt. If expr contains
// only a value, it is converted to dt.
func asType(expr []part, dt types.Type, nullable bool) error {
	p := expr[0]
	if len(expr) == 1 && p.path.elements == nil {
		if s, ok := p.value.(string); ok && s == "" {
			nullable = false
		}
		v, err := convert(p.value, p.typ, dt, nullable, false, nil, None)
		if err != nil {
			if p.value == nil {
				return fmt.Errorf("null is not convertible to the %s type", dt)
			}
			var msg string
			switch err {
			case errRangeConversion:
				msg = fmt.Sprintf("number %s is not a %s value", p.value, dt)
			case errMinConversion:
				var n any
				switch dt.Kind() {
				case types.IntKind:
					n, _ = dt.IntRange()
				case types.UintKind:
					n, _ = dt.UintRange()
				case types.FloatKind:
					n, _ = dt.FloatRange()
				case types.DecimalKind:
					n, _ = dt.DecimalRange()
				}
				msg = fmt.Sprintf("number %s is less than %v", p.value, n)
			case errMaxConversion:
				var n any
				switch dt.Kind() {
				case types.IntKind:
					_, n = dt.IntRange()
				case types.UintKind:
					_, n = dt.UintRange()
				case types.FloatKind:
					_, n = dt.FloatRange()
				case types.DecimalKind:
					_, n = dt.DecimalRange()
				}
				msg = fmt.Sprintf("number %s is greater than %v", p.value, n)
			case errEnumConversion:
				msg = fmt.Sprintf("%q is not one of the allowed values", p.value)
			case errRegexpConversion:
				msg = fmt.Sprintf("%q does not match /%s/", p.value, dt.Regexp())
			case errMaxByteLengthConversion:
				n, _ := dt.MaxByteLength()
				msg = fmt.Sprintf("%q exceeds the %d-byte limit", p.value, n)
			case errMaxLengthConversion:
				n, _ := dt.MaxLength()
				msg = fmt.Sprintf("%q exceeds the %d-char limit", p.value, n)
			default:
				var s string
				switch v := p.value.(type) {
				case string:
					s = strconv.Quote(v)
				case int:
					s = strconv.Itoa(v)
				case decimal.Decimal:
					s = v.String()
				case bool:
					s = strconv.FormatBool(v)
				case json.Value:
					s = "null"
				}
				msg = fmt.Sprintf("%s is not convertible to the %s type", s, dt.String())
			}
			return errors.New(msg)
		}
		expr[0].value = v
		expr[0].typ = dt
		return nil
	}
	st := types.String()
	if len(expr) == 1 && p.value == nil {
		st = p.typ
		// If it is not valid, it should not be validated.
		if !st.Valid() {
			return nil
		}
	}
	if !convertibleTo(st, dt) {
		return fmt.Errorf("cannot convert expression (type %s) to %s", st, dt)
	}
	return nil
}

// typeOf returns the type of the expression expr.
func typeOf(expr []part) types.Type {
	p := expr[0]
	if len(expr) > 1 || p.value != nil && p.path.elements != nil {
		return types.String()
	}
	return p.typ
}
