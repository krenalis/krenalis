//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package connectors

import (
	"fmt"

	"github.com/open2b/chichi/types"
)

// checkSchemaAlignment checks whether the schema t1 is aligned with t2 and
// returns a *SchemaError error if it is not aligned.
// It panics if a schema is not valid.
func checkSchemaAlignment(t1, t2 types.Type) error {
	return checkTypeAlignment("", t1, t2)
}

// checkTypeAlignment is called by checkSchemaAlignment to check if t1 is
// aligned with t2.
func checkTypeAlignment(name string, t1, t2 types.Type) error {
	k1 := t1.Kind()
	k2 := t2.Kind()
	switch {
	// Types Int, Uint, Float, Decimal, and Year are aligned.
	case k1 >= types.IntKind && k1 <= types.DecimalKind || k1 == types.YearKind:
		if k2 >= types.IntKind && k2 <= types.DecimalKind || k2 == types.YearKind {
			return nil
		}
	// Types Text and UUID are aligned.
	// Types Text and Inet are aligned.
	case k1 == types.TextKind || k1 == types.InetKind || k1 == types.UUIDKind:
		if k2 == k1 || k2 == types.TextKind {
			return nil
		}
	// An Array type is aligned with another Array type if its item type is aligned with the other item type.
	case k1 == types.ArrayKind:
		if k2 == types.ArrayKind {
			return checkTypeAlignment(name, t1.Elem(), t2.Elem())
		}
	// An Object type is aligned with another Object type if its property names are also present in the other Object
	// and the types of the properties are aligned with the types of the respective properties.
	case k1 == types.ObjectKind:
		if k2 == types.ObjectKind {
			for _, p1 := range t1.Properties() {
				path := p1.Name
				if name != "" {
					path = name + "." + path
				}
				p2, ok := t2.Property(p1.Name)
				if !ok {
					return &SchemaError{Msg: fmt.Sprintf(`%q property no longer exists`, path)}
				}
				err := checkTypeAlignment(path, p1.Type, p2.Type)
				if err != nil {
					return err
				}
			}
			return nil
		}
	// A Map type is aligned with another Map type if its value type is aligned with the other value type.
	case k1 == types.MapKind:
		if k2 == types.MapKind {
			return checkTypeAlignment(name, t1.Elem(), t2.Elem())
		}
	// Apart from the previous cases, if two types have the same kind, they are aligned.
	case k1 == k2:
		return nil
	}
	return &SchemaError{Msg: fmt.Sprintf("type of the %q property has changed from %s to %s", name, t1, t2)}
}
