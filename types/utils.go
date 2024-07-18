//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"iter"
	"regexp"
	"slices"
	"strings"
)

// AsRole returns a new type with properties of t that are compatible with role.
// If all properties of t are compatible with role, it returns t. If no
// properties of t are compatible with role, it returns an invalid type.
//
// It panics if t or role are not valid types, or if t is not an object type or
// role is 'Both'.
func AsRole(t Type, role Role) Type {
	if !t.Valid() {
		panic("type is not valid")
	}
	if t.kind != ObjectKind {
		panic("cannot return type as role for non-Object type")
	}
	if role < BothRole || role > DestinationRole {
		panic("role is not valid")
	}
	if role == BothRole {
		return t
	}
	last := 0
	var roleProperties []Property
	properties := t.vl.([]Property)
	for i, p := range properties {
		if p.Role == role {
			continue
		}
		if p.Role == BothRole && (role == SourceRole && !p.CreateRequired && !p.UpdateRequired ||
			role == DestinationRole && !p.ReadOptional) {
			continue
		}
		if last < i {
			roleProperties = append(roleProperties, properties[last:i]...)
		}
		if p.Role == BothRole {
			switch role {
			case SourceRole:
				p.CreateRequired = false
				p.UpdateRequired = false
			case DestinationRole:
				p.ReadOptional = false
			}
			roleProperties = append(roleProperties, p)
		}
		last = i + 1
	}
	if last == 0 {
		return t
	}
	if last < len(properties) {
		roleProperties = append(roleProperties, properties[last:]...)
	}
	if roleProperties == nil {
		return Type{}
	}
	return Type{kind: ObjectKind, vl: roleProperties}
}

// Equal reports whether two types are equal.
func Equal(t1, t2 Type) bool {
	almostEqual := t1.kind == t2.kind && t1.size == t2.size && t1.unique == t2.unique && t1.real == t2.real && t1.p == t2.p && t1.s == t2.s
	if !almostEqual {
		return false
	}
	if t1.vl == nil && t2.vl == nil {
		return true
	}
	if (t1.vl == nil) != (t2.vl == nil) {
		return false
	}
	switch vl1 := t1.vl.(type) {
	case Type:
		return Equal(vl1, t2.vl.(Type))
	case intRange, uintRange, floatRange, decimalRange, string:
		return vl1 == t2.vl
	case []Property:
		vl2 := t2.vl.([]Property)
		if len(vl1) != len(vl2) {
			return false
		}
		for i, p1 := range vl1 {
			p2 := (vl2)[i]
			if p1.Name != p2.Name ||
				p1.Label != p2.Label ||
				p1.Placeholder != p2.Placeholder ||
				p1.Role != p2.Role ||
				p1.CreateRequired != p2.CreateRequired ||
				p1.UpdateRequired != p2.UpdateRequired ||
				p1.ReadOptional != p2.ReadOptional ||
				p1.Nullable != p2.Nullable ||
				p1.Note != p2.Note ||
				!Equal(p1.Type, p2.Type) {
				return false
			}
		}
		return true
	case []string:
		vl2, ok := t2.vl.([]string)
		return ok && slices.Equal(vl1, vl2)
	case *regexp.Regexp:
		vl2, ok := t2.vl.(*regexp.Regexp)
		return ok && vl1.String() == vl2.String()
	}
	panic("unreachable code")
}

// IsValidPropertyName reports whether name is a valid property name.
// A property name must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_]
func IsValidPropertyName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !('a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}

// IsValidPropertyPath reports whether path is a valid property path.
// A property path is formed by property names separated by periods.
func IsValidPropertyPath(path string) bool {
	for path != "" {
		i := strings.IndexByte(path, '.')
		if i == -1 {
			i = len(path)
		}
		if !IsValidPropertyName(path[:i]) {
			return false
		}
		if i == len(path) {
			return true
		}
		path = path[i+1:]
	}
	return false
}

// NumProperties returns the count of properties in t.
// Panics if t is not an Object type.
func NumProperties(t Type) int {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-Object type")
	}
	return len(t.vl.([]Property))
}

// Properties returns the properties of the Object type t.
// Panics if t is not an Object type.
func Properties(t Type) []Property {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-Object type")
	}
	return slices.Clone(t.vl.([]Property))
}

// PropertyByPath returns the property with the given path in the Object t, or
// a PathNotExistError error if the property does not exist.
//
// Unlike Walk, it does not traverse through Arrays and Maps. If path is "x.y"
// and the type of "x" is not an Object, it returns a PathNotExistError error.
//
// It panics if t is not of type Object or if path is not a valid path.
func PropertyByPath(t Type, path string) (Property, error) {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-Object type")
	}
	name, rest := "", path
Rest:
	for {
		name, rest, _ = strings.Cut(rest, ".")
		if t.kind != ObjectKind {
			break
		}
		properties := t.vl.([]Property)
		for j := 0; j < len(properties); j++ {
			if properties[j].Name != name {
				continue
			}
			if rest == "" {
				return properties[j], nil
			}
			t = properties[j].Type
			continue Rest
		}
		break
	}
	if !IsValidPropertyPath(path) {
		panic("invalid property path")
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, rest), ".")
	return Property{}, PathNotExistError{path}
}

// PropertyExists reports whether property with the given path exists in the
// Object t.
//
// If path is "x.y" and the property "x" has type Array(T) or Map(T), it reports
// whether T has the property "y".
//
// It panics if t is not of type Object or if path is not a valid path.
func PropertyExists(t Type, path string) bool {
	if t.kind != ObjectKind {
		panic("cannot check the properties of a non-Object type")
	}
	if !IsValidPropertyPath(path) {
		panic("invalid property path")
	}
	var name string
Object:
	for {
		name, path, _ = strings.Cut(path, ".")
		properties := t.vl.([]Property)
		for i := 0; i < len(properties); i++ {
			if properties[i].Name != name {
				continue
			}
			if path == "" {
				return true
			}
			t = properties[i].Type
			for {
				switch t.kind {
				case ObjectKind:
					continue Object
				case ArrayKind, MapKind:
					t = t.Elem()
				default:
					return false
				}
			}
		}
		return false
	}
}

// PropertyNames returns the names of the properties of the Object t.
// Panics if t is not an Object type.
func PropertyNames(t Type) []string {
	if t.kind != ObjectKind {
		panic("cannot get the property names of a non-Object type")
	}
	pp := t.vl.([]Property)
	names := make([]string, len(pp))
	for i := 0; i < len(pp); i++ {
		names[i] = pp[i].Name
	}
	return names
}

// SubsetFunc returns a subset of the object t, including only the properties
// for which f returns true, maintaining their original order in type t.
// If f returns false for all properties, it returns an invalid schema.
// It panics if t is not an object, or f is nil.
func SubsetFunc(t Type, f func(p Property) bool) Type {
	if t.kind != ObjectKind {
		panic("cannot get a subset of a non-Object type")
	}
	var ps []Property
	pp := t.vl.([]Property)
	all := true
	for i := 0; i < len(pp); i++ {
		if f(pp[i]) {
			if !all {
				ps = append(ps, pp[i])
			}
		} else if all {
			if i > 0 {
				ps = append(pp[:0:0], pp[:i]...)
			}
			all = false
		}
	}
	if all {
		return t
	}
	if ps == nil {
		return Type{}
	}
	return Type{kind: ObjectKind, vl: ps}
}

// Walk returns an iterator over all the properties in t in a depth-first order.
//
// For example:
//
//	for path, property := range Walk(t) {
//	    fmt.Printf("%s: %s\n", path, property.Type.Kind)
//	}
//
// If a property "x" has type Array(T) or Map(T) and T has the property "y", its
// path is "x.y".
//
// It panics if t is not an Object.
func Walk(t Type) iter.Seq2[string, Property] {
	if t.kind != ObjectKind {
		panic("cannot iterate over a non-Object type")
	}
	return func(yield func(path string, property Property) bool) {
		type entry struct {
			base string
			prop *Property
		}
		properties := t.vl.([]Property)
		n := len(properties)
		pp := make([]entry, n)
		for i := 0; i < n; i++ {
			pp[i].prop = &properties[n-1-i]
		}
		for len(pp) > 0 {
			var e entry
			n := len(pp)
			e, pp = pp[n-1], pp[:n-1]
			t := e.prop.Type
			for t.kind == MapKind || t.kind == ArrayKind {
				t = t.Elem()
			}
			if t.kind == ObjectKind {
				properties := t.vl.([]Property)
				for i := len(properties) - 1; i >= 0; i-- {
					pp = append(pp, entry{base: e.base + e.prop.Name + ".", prop: &properties[i]})
				}
			}
			if !yield(e.base+e.prop.Name, *e.prop) {
				return
			}
		}
	}
}
