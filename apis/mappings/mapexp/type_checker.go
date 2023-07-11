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

			s := 0
			j := 0
			t := schema
			for j < len(p.path) {
				for ; j < len(p.path); j++ {
					if p.path[j][0] == ':' {
						break
					}
				}
				if s < j {
					property, err := t.PropertyByPath(p.path[s:j])
					if err != nil {
						return fmt.Errorf("property %q does not exist", p.path[:j])
					}
					t = property.Type
				}
				if j == len(p.path) {
					break
				}
				if t.PhysicalType() != types.PtMap {
					return fmt.Errorf("cannot access to property %q (type %s) as a map", t, p.path[:j])
				}
				t = t.ValueType()
				j++
				s = j
			}

			if concatenate && !convertibleTo(t, types.Text()) {
				return fmt.Errorf("cannot convert property %s (type %s) to Text", p.path, t)
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
				return fmt.Errorf("cannot convert %s(...) (type %s) to Text", p.path, st)
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
