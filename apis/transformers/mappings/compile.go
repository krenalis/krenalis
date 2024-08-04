//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"
)

var jsonArrayType = types.Array(types.JSON())

// Expression represents a mapping expression used to transform data from a
// source to a destination. An Expression can contain strings, numbers, true,
// false, null, property paths and function calls.
type Expression struct {
	parts       []part     // expression parts.
	dt          types.Type // destination type.
	properties  []string   // properties used in the expression; see the documentation of the Properties method.
	timeLayouts *state.TimeLayouts
}

// Properties returns the properties found in the expression, sorted by their
// appearance order in the expression. The returned properties are guaranteed to
// be unique. If no property are present, it returns nil.
//
// If the expression contains a map or JSON indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (expr *Expression) Properties() []string {
	return slices.Clone(expr.properties)
}

// Compile parses an expression and returns an Expression value that can be used
// to execute the expression.
//
// schema is the schema of the paths in the expression, dt is the destination
// type, and layouts represents, if not nil, the layouts used to format
// DateTime, Date, and Time values as strings.
//
// An invalid schema can be passed to compile an expression without paths.
func Compile(expr string, schema, dt types.Type, layouts *state.TimeLayouts) (*Expression, error) {
	if expr == "" {
		return nil, errors.New("expression is empty")
	}
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.New("schema is non an Object")
	}
	if !dt.Valid() {
		return nil, errors.New("destination type is the invalid type")
	}
	parts, src, err := parse(expr)
	if err != nil {
		return nil, err
	}
	if src != "" {
		return nil, fmt.Errorf("unexpected character %v", strconv.QuoteRuneToGraphic(rune(src[0])))
	}
	properties := map[string]struct{}{}
	err = typeCheck(parts, schema, dt, true, properties)
	if err != nil {
		return nil, err
	}
	expression := &Expression{
		parts:       parts,
		dt:          dt,
		timeLayouts: layouts,
	}
	if len(properties) > 0 {
		expression.properties = make([]string, len(properties))
		i := 0
		for name := range properties {
			expression.properties[i] = name
			i++
		}
		slices.Sort(expression.properties)
	}
	return expression, nil
}

// checkAnd type checks a call to 'and' with the given arguments.
func checkAnd(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	if len(args) < 2 {
		return types.Type{}, errors.New("'and' function requires at least two argument")
	}
	booleanType := types.Boolean()
	for _, arg := range args {
		err := typeCheck(arg, schema, booleanType, true, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	return booleanType, nil
}

// checkArray type checks a call to 'array' with the given arguments.
func checkArray(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	for _, arg := range args {
		err := typeCheck(arg, schema, types.JSON(), false, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	return jsonArrayType, nil
}

// checkCoalesce type checks a call to 'coalesce' with the given arguments.
func checkCoalesce(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	if len(args) < 1 {
		return types.Type{}, errors.New("'coalesce' function requires at least one argument")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, dt, true, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	return types.Type{}, nil
}

// checkEq type checks a call to 'eq' with the given arguments.
func checkEq(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	if len(args) != 2 {
		return types.Type{}, errors.New("'eq' function requires two arguments")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, types.Type{}, true, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	t0 := typesOf(args[0])
	t1 := typesOf(args[1])
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
func checkIf(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n < 2 || n > 3 {
		return types.Type{}, errors.New("'if' function requires either two or three arguments")
	}
	err := typeCheck(args[0], schema, types.Boolean(), true, properties)
	if err != nil {
		return types.Type{}, err
	}
	err = typeCheck(args[1], schema, dt, nullable, properties)
	if err != nil {
		return types.Type{}, err
	}
	if n == 3 {
		err = typeCheck(args[2], schema, dt, nullable, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	return dt, nil
}

// checkInitCap type checks a call to 'initcap' with the given arguments.
func checkInitCap(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'initcap' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.Text(), true, properties)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// checkLower type checks a call to 'lower' with the given arguments.
func checkLower(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'lower' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.Text(), true, properties)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// checkNe type checks a call to 'ne' with the given arguments.
func checkNe(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	if len(args) != 2 {
		return types.Type{}, errors.New("'ne' function requires two arguments")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, types.Type{}, true, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	t0 := typesOf(args[0])
	t1 := typesOf(args[1])
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
func checkNot(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'not' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.Boolean(), true, properties)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// checkOr type checks a call to 'or' with the given arguments.
func checkOr(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	if len(args) < 2 {
		return types.Type{}, errors.New("'or' function requires at least two argument")
	}
	booleanType := types.Boolean()
	for _, arg := range args {
		err := typeCheck(arg, schema, booleanType, true, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	return booleanType, nil
}

// checkSubstring type checks a call to 'substring' with the given arguments.
func checkSubstring(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n < 2 || 3 < n {
		return types.Type{}, errors.New("'substring' function requires two or three arguments")
	}
	err := typeCheck(args[0], schema, types.Text(), true, properties)
	if err != nil {
		return types.Type{}, err
	}
	err = typeCheck(args[1], schema, types.Int(32), true, properties)
	if err != nil {
		return types.Type{}, err
	}
	if n == 3 {
		err = typeCheck(args[2], schema, types.Int(32), true, properties)
		if err != nil {
			return types.Type{}, err
		}
	}
	return dt, nil
}

// checkUpper type checks a call to 'upper' with the given arguments.
func checkUpper(args [][]part, schema, dt types.Type, nullable bool, properties map[string]struct{}) (types.Type, error) {
	n := len(args)
	if n != 1 {
		return types.Type{}, errors.New("'upper' function requires a single argument")
	}
	err := typeCheck(args[0], schema, types.Text(), true, properties)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}

// typeCheck type checks the expression expr. schema is the schema of the
// properties in the expression, dt is the destination type, and nullable
// indicates whether that value can be nil. An invalid schema can be passed to
// type check an expression without paths.
func typeCheck(expr []part, schema, dt types.Type, nullable bool, properties map[string]struct{}) error {

	typ := dt
	n := nullable
	concatenate := len(expr) > 1 || expr[0].value != nil
	if concatenate {
		typ = types.Text()
		n = true
	}

	for i, p := range expr {
		if p.path == nil {
			continue
		}
		// Check the path.
		if p.args == nil {
			var b strings.Builder
			t := schema
			for j := 0; j < len(p.path); j++ {
				name := p.path[j]
				switch t.Kind() {
				case types.JSONKind:
					p.path[j] = ":" + name
				case types.ObjectKind, types.InvalidKind:
					if name[len(name)-1] == '?' {
						return fmt.Errorf("invalid %s: operator '?' can be used only with JSON", stringifyPath(p.path[:j+1]))
					}
					if name[0] == '[' {
						name = name[1 : len(name)-1]
						if !types.IsValidPropertyName(name) {
							return fmt.Errorf("invalid %s: %q is not a valid property name", stringifyPath(p.path[:j+1]), name)
						}
					}
					var property types.Property
					var ok bool
					if t.Valid() {
						property, ok = t.Property(name)
					}
					if !ok {
						msg := fmt.Sprintf("property %q does not exist", name)
						if j > 0 {
							msg = fmt.Sprintf("invalid %s: %s", stringifyPath(p.path[:j+1]), msg)
						}
						return errors.New(msg)
					}
					if b.Len() > 0 {
						b.WriteByte('.')
					}
					b.WriteString(name)
					t = property.Type
				case types.MapKind:
					if name[len(name)-1] == '?' {
						return fmt.Errorf("invalid %s: operator '?' can be used only with JSON", stringifyPath(p.path[:j+1]))
					}
					p.path[j] = ":" + name
					t = t.Elem()
				default:
					return fmt.Errorf("invalid %s: %s (type %s) cannot have properties or keys", stringifyPath(p.path[:j+1]), stringifyPath(p.path[:j]), t)
				}
			}
			if concatenate && !convertibleTo(t, types.Text()) {
				return fmt.Errorf("cannot convert %s (type %s) to Text", stringifyPath(p.path), t)
			}
			properties[b.String()] = struct{}{}
			expr[i].typ = t
			continue
		}
		// Check the function call
		var err error
		switch p.path[0] {
		case "and":
			expr[i].typ, err = checkAnd(p.args, schema, typ, n, properties)
		case "array":
			expr[i].typ, err = checkArray(p.args, schema, typ, n, properties)
		case "coalesce":
			expr[i].typ, err = checkCoalesce(p.args, schema, typ, n, properties)
		case "eq":
			expr[i].typ, err = checkEq(p.args, schema, typ, n, properties)
		case "if":
			expr[i].typ, err = checkIf(p.args, schema, typ, n, properties)
		case "initcap":
			expr[i].typ, err = checkInitCap(p.args, schema, typ, n, properties)
		case "lower":
			expr[i].typ, err = checkLower(p.args, schema, typ, n, properties)
		case "ne":
			expr[i].typ, err = checkNe(p.args, schema, typ, n, properties)
		case "not":
			expr[i].typ, err = checkNot(p.args, schema, typ, n, properties)
		case "or":
			expr[i].typ, err = checkOr(p.args, schema, typ, n, properties)
		case "substring":
			expr[i].typ, err = checkSubstring(p.args, schema, typ, n, properties)
		case "upper":
			expr[i].typ, err = checkUpper(p.args, schema, typ, n, properties)
		default:
			panic(fmt.Errorf("unknown function %q", p.path[0]))
		}
		if err != nil {
			return err
		}
		if concatenate {
			if st := expr[i].typ; st.Valid() && !convertibleTo(st, types.Text()) {
				return fmt.Errorf("cannot convert %s(...) (type %s) to Text", stringifyPath(p.path), st)
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
	if len(expr) == 1 && p.path == nil {
		v, err := convert(p.value, p.typ, dt, nullable, nil, None)
		if err != nil {
			if p.value == nil {
				return fmt.Errorf("cannot convert null to %s", dt)
			}
			return fmt.Errorf("cannot convert %v (type %s) to %s", p.value, p.typ, dt)
		}
		expr[0].value = v
		expr[0].typ = dt
		return nil
	}
	st := types.Text()
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
func typesOf(expr []part) types.Type {
	p := expr[0]
	if len(expr) > 0 || p.value != nil && p.path != nil {
		return types.Text()
	}
	return p.typ
}
