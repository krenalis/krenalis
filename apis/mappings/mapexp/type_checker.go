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

	t := dt
	n := nullable
	concatenate := len(expr) > 1 || expr[0].value != nil
	if concatenate {
		t = types.Text()
		n = true
	}

	for i, p := range expr {
		if p.path == nil {
			continue
		}
		// Check the path.
		if p.args == nil {
			property, err := schema.PropertyByPath(p.path)
			if err != nil {
				return fmt.Errorf("property %q does not exist", p.path)
			}
			if concatenate && !convertibleTo(property.Type, types.Text()) {
				return fmt.Errorf("cannot convert property %s (type %s) to Text", p.path, property.Type)
			}
			expr[i].typ = property.Type
			continue
		}
		// Check the function call
		var err error
		switch p.path[0] {
		case "coalesce":
			expr[i].typ, err = checkCoalesce(p.args, schema, t, n)
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

	return asType(expr, dt, nullable)
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
