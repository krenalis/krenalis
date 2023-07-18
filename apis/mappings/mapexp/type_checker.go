//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

import (
	"fmt"

	"chichi/connector/types"
)

// typeCheck type checks the expression expr. schema is the schema of the
// properties in the expression, dt is the destination type, and nullable
// indicates whether the result value of the evaluation can be nil.
func typeCheck(expr []part, schema, dt types.Type, nullable bool) error {

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
			t := schema
		Path:
			for j := 0; j < len(p.path); j++ {
				switch t.PhysicalType() {
				case types.PtJSON:
					break Path
				case types.PtObject:
					if p.path[j][0] == ':' {
						return fmt.Errorf("cannot use '[]' notation with %q (type %s)", t, stringifyPath(p.path[:j]))
					}
					property, ok := t.Property(p.path[j])
					if !ok {
						return fmt.Errorf("property %q does not exist", stringifyPath(p.path[:j+1]))
					}
					t = property.Type
				case types.PtMap:
					t = t.ValueType()
				default:
					if p.path[j][0] == ':' {
						return fmt.Errorf("cannot use '[]' notation with %q (type %s)", t, stringifyPath(p.path[:j]))
					}
					return fmt.Errorf("cannot use dot notation with %q (type %s)", t, stringifyPath(p.path[:j]))
				}
			}
			if concatenate && !convertibleTo(t, types.Text()) {
				return fmt.Errorf("cannot convert %s (type %s) to Text", stringifyPath(p.path), t)
			}
			expr[i].typ = t
			continue
		}
		// Check the function call
		var err error
		switch p.path[0] {
		case "and":
			expr[i].typ, err = checkAnd(p.args, schema, typ, n)
		case "coalesce":
			expr[i].typ, err = checkCoalesce(p.args, schema, typ, n)
		case "eq":
			expr[i].typ, err = checkEq(p.args, schema, typ, n)
		case "when":
			expr[i].typ, err = checkWhen(p.args, schema, typ, n)
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
		v, err := convert(p.value, p.typ, dt, nullable, false)
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
