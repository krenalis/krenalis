//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// PathNotExistError is returned by PropertyByPath when the path does not exist.
type PathNotExistError struct {
	Path string
}

// Role represents a role.
type Role int

const (
	Source Role = 1 + iota
	Destination
)

func (err PathNotExistError) Error() string {
	return fmt.Sprintf("property path %q does not exist", err.Path)
}

// AsRole returns a copy of t with the ReadOptional, CreateRequired, and
// UpdateRequired fields of each property adjusted to ensure compatibility with
// the specified role:
//
// - If the role is Source, CreateRequired and UpdateRequired are set to false.
// - If the role is Destination, ReadOptional is set to false.
//
// If all properties of t are already compatible with the specified role, the
// function returns t unchanged. It panics if t is not of the object type or if
// the role is neither Source nor Destination.
func AsRole(t Type, role Role) Type {
	if !t.Valid() {
		panic("type is not valid")
	}
	if t.kind != ObjectKind {
		panic("cannot return type as role for non-object type")
	}
	if role != Source && role != Destination {
		panic("role is not valid")
	}
	t, _ = asRole(t, role)
	return t
}

// DecodeUUID returns the UUID corresponding to the given byte slice
// (representing the 128 bit of the UUID, so it must have length 16) it in the
// canonical string form without uppercase letters. The boolean return value
// reports whether s represent a UUID or not.
func DecodeUUID(s []byte) (string, bool) {
	id, err := uuid.FromBytes(s)
	if err != nil {
		return "", false
	}
	return id.String(), true
}

// Equal reports whether two types are equal.
func Equal(t1, t2 Type) bool {
	almostEqual := t1.kind == t2.kind && t1.size == t2.size && t1.generic == t2.generic && t1.unique == t2.unique && t1.real == t2.real && t1.p == t2.p && t1.s == t2.s
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
	case intRange, uintRange, floatRange, string:
		return vl1 == t2.vl
	case decimalRange:
		vl2 := t2.vl.(decimalRange)
		return vl1.min.Equal(vl2.min) && vl1.max.Equal(vl2.max)
	case Properties:
		pp1 := vl1.properties
		pp2 := t2.vl.(Properties).properties
		if len(pp1) != len(pp2) {
			return false
		}
		for i, p1 := range pp1 {
			p2 := (pp2)[i]
			if p1.Name != p2.Name ||
				p1.Prefilled != p2.Prefilled ||
				p1.CreateRequired != p2.CreateRequired ||
				p1.UpdateRequired != p2.UpdateRequired ||
				p1.ReadOptional != p2.ReadOptional ||
				p1.Nullable != p2.Nullable ||
				p1.Description != p2.Description ||
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

// Filter returns a subset of object t containing only the properties for which
// f returns true, preserving their original order in t.
// If f returns false for all properties, the result is an invalid schema.
//
// See also Prune if you need to filter beyond the top-level properties.
//
// It panics if t is not an object or if f is nil.
func Filter(t Type, f func(p Property) bool) Type {
	if t.kind != ObjectKind {
		panic("cannot get a subset of a non-object type")
	}
	var ps []Property
	pp := t.vl.(Properties).properties
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
	names := make(map[string]int, len(ps))
	for i, p := range ps {
		names[p.Name] = i
	}
	return Type{kind: ObjectKind, vl: Properties{properties: ps, names: names}}
}

// ParseUUID parses s as a UUID in the standard form xxxx-xxxx-xxxx-xxxxxxxxxxxx
// and returns it in the canonical form without uppercase letters. The boolean
// return value reports whether s is a UUID in the standard form.
func ParseUUID(s string) (string, bool) {
	if len(s) != 36 {
		return "", false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return "", false
	}
	return id.String(), true
}

// Prune removes from t all properties for which f returns false, keeping only
// those for which f returns true. f is evaluated only on non-object properties.
// An object property is removed if all of its subproperties are removed.
//
// See also Filter, which restricts only top-level properties.
//
// It panics if t is not an object or if f is nil.
func Prune(t Type, f func(path string) bool) Type {
	if t.kind != ObjectKind {
		panic("cannot prune a non-object type")
	}
	pp, ok := prune(t.vl.(Properties).properties, "", f)
	if !ok {
		return t
	}
	if pp == nil {
		return Type{}
	}
	names := make(map[string]int, len(pp))
	for i, p := range pp {
		names[p.Name] = i
	}
	return Type{kind: ObjectKind, vl: Properties{properties: pp, names: names}}
}

// PruneAtPath returns the subset of t that contains only the properties along
// the specified path and their parent hierarchy. If the path does not exist, it
// returns a PathNotExistError and the invalid Type.
// It panics if t is not an object or if path is not a valid property path.
func PruneAtPath(t Type, path string) (Type, error) {
	if t.kind != ObjectKind {
		panic("cannot get a subset of a non-object type")
	}
	if _, err := t.Properties().ByPath(path); err != nil {
		return Type{}, err
	}
	return pruneAtPath(t, path), nil
}

// asRole is a recursive function called by the AsRole method. t must be an
// object type, and role must be either Source or Destination. It returns the
// resulting type and a boolean indicating whether the returned type is
// different from t.
func asRole(t Type, role Role) (Type, bool) {
	pp := t.vl.(Properties).properties
	var ppc []Property
	for i := 0; i < len(pp); i++ {
		if pp[i].Type.Kind() == ObjectKind {
			if t, ok := asRole(pp[i].Type, role); ok {
				if ppc == nil {
					ppc = slices.Clone(pp)
				}
				ppc[i].Type = t
			}
		}
		switch role {
		case Source:
			if pp[i].CreateRequired || pp[i].UpdateRequired {
				if ppc == nil {
					ppc = slices.Clone(pp)
				}
				ppc[i].CreateRequired = false
				ppc[i].UpdateRequired = false
			}
		case Destination:
			if pp[i].ReadOptional {
				if ppc == nil {
					ppc = slices.Clone(pp)
				}
				ppc[i].ReadOptional = false
			}
		}
	}
	if ppc == nil {
		return t, false
	}
	names := make(map[string]int, len(ppc))
	for i, p := range ppc {
		names[p.Name] = i
	}
	return Type{kind: ObjectKind, vl: Properties{properties: ppc, names: names}}, true
}

// prune is a recursive helper called by Prune. It returns the pruned
// properties and a boolean indicating whether all properties were pruned.
// If no property is pruned, it returns nil and false. If all properties are
// pruned, it returns nil and true.
func prune(pp []Property, path string, f func(string) bool) ([]Property, bool) {
	var ps []Property
	unchanged := true
	for i := 0; i < len(pp); i++ {
		if t := &pp[i].Type; t.kind == ObjectKind {
			if properties, ok := prune(t.vl.(Properties).properties, path+pp[i].Name+".", f); ok {
				// Almost one property of the object has been pruned.
				if properties == nil {
					// Prune the entire property.
					if unchanged {
						if i > 0 {
							ps = append([]Property(nil), pp[:i]...)
						}
						unchanged = false
					}
					continue
				}
				// Prune some property of the object.
				if unchanged {
					if i > 0 {
						ps = append([]Property(nil), pp[:i]...)
					}
					unchanged = false
				}
				names := make(map[string]int, len(properties))
				for i, p := range properties {
					names[p.Name] = i
				}
				p := pp[i]
				p.Type.vl = Properties{properties: properties, names: names}
				ps = append(ps, p)
				continue
			}
		} else if !f(path + pp[i].Name) {
			// Prune the property.
			if unchanged {
				if i > 0 {
					ps = append([]Property(nil), pp[:i]...)
				}
				unchanged = false
			}
			continue
		}
		// Don't prune.
		if !unchanged {
			ps = append(ps, pp[i])
		}
	}
	if unchanged {
		return nil, false
	}
	if ps == nil {
		return nil, true
	}
	return ps, true
}

// pruneAtPath is a recursive function called by the PruneAtPath function. t
// must be an object type and path must exist in t.
func pruneAtPath(t Type, path string) Type {
	name, rest, found := strings.Cut(path, ".")
	property, _ := t.Properties().ByName(name)
	if !found {
		return Object([]Property{property})
	}
	property.Type = pruneAtPath(property.Type, rest)
	return Object([]Property{property})
}
