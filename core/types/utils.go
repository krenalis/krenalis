//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"fmt"
	"iter"
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
	case []Property:
		vl2 := t2.vl.([]Property)
		if len(vl1) != len(vl2) {
			return false
		}
		for i, p1 := range vl1 {
			p2 := (vl2)[i]
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
// Panics if t is not an object type.
func NumProperties(t Type) int {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-object type")
	}
	return len(t.vl.([]Property))
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

// Properties returns the properties of the object type t.
// Panics if t is not an object type.
func Properties(t Type) []Property {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-object type")
	}
	properties := t.vl.([]Property)
	pp := make([]Property, len(properties))
	copy(pp, properties)
	return pp
}

// PropertyByPath returns the property with the given path in the object t.
// If the property does not exist, it returns the last valid property found (if
// any), and a PathNotExistError error.
//
// Unlike Walk, it does not traverse through arrays and maps. If path is "x.y"
// and the type of "x" is not an object, it returns a PathNotExistError error.
//
// It panics if t is not of type object or if path is not a valid path.
func PropertyByPath(t Type, path string) (Property, error) {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-object type")
	}
	var p *Property
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
			p = &properties[j]
			t = p.Type
			continue Rest
		}
		break
	}
	if !IsValidPropertyPath(path) {
		panic("invalid property path")
	}
	err := PathNotExistError{strings.TrimSuffix(strings.TrimSuffix(path, rest), ".")}
	if p == nil {
		return Property{}, err
	}
	return *p, err
}

// PropertyByPathSlice is like PropertyByPath but takes a slice of property
// names as the path.
func PropertyByPathSlice(t Type, path []string) (Property, error) {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-object type")
	}
	var p *Property
	last := len(path) - 1
	i := 0
	var name string
Rest:
	for i, name = range path {
		if t.kind != ObjectKind {
			break
		}
		properties := t.vl.([]Property)
		for j := 0; j < len(properties); j++ {
			if properties[j].Name != name {
				continue
			}
			if i == last {
				return properties[j], nil
			}
			p = &properties[j]
			t = p.Type
			continue Rest
		}
		break
	}
	for _, name := range path {
		if !IsValidPropertyName(name) {
			panic("invalid property path")
		}
	}
	err := PathNotExistError{strings.Join(path[:i+1], ".")}
	if p == nil {
		return Property{}, err
	}
	return *p, err
}

// PropertyExists reports whether property with the given path exists in the
// object t.
//
// If path is "x.y" and the property "x" has type array(T) or map(T), it reports
// whether T has the property "y".
//
// It panics if t is not of type object or if path is not a valid path.
func PropertyExists(t Type, path string) bool {
	if t.kind != ObjectKind {
		panic("cannot check the properties of a non-object type")
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

// PropertyNames returns the names of the properties of the object t.
// Panics if t is not an object type.
func PropertyNames(t Type) []string {
	if t.kind != ObjectKind {
		panic("cannot get the property names of a non-object type")
	}
	pp := t.vl.([]Property)
	names := make([]string, len(pp))
	for i := 0; i < len(pp); i++ {
		names[i] = pp[i].Name
	}
	return names
}

// SubsetByPathFunc returns a subset of the object t, including the properties
// for which f returns true and their upper hierarchy, maintaining their
// original order in type t. The properties of t are navigated recursively by
// traversing inside the object properties.
// If the function f does not return true for any property, an invalid Type is
// returned.
// It panics if t is not an object, or f is nil.
func SubsetByPathFunc(t Type, f func(path string) bool) Type {
	if t.kind != ObjectKind {
		panic("cannot get a subset of a non-object type")
	}

	// depthOf returns the depth of path, starting at 1 (for top-level property
	// paths, such as "a"). So, for example, the path "x.y.z" has a depth of 3.
	depthOf := func(path string) int {
		return strings.Count(path, ".") + 1
	}

	// Creates a list of all property paths that must actually result in the
	// returned object, which are ordered by position (for properties with the
	// same depth). The list also includes properties from the upper hierarchy,
	// for which the f function did not return true, since they must also be
	// returned, having to maintain the structure.
	toAdd := []string{}
	propByPath := map[string]Property{} // every property in t.
	var maxDepth int                    // 1 means: top level property, such as "a".
	fReturnedTrue := map[string]struct{}{}
	for path, property := range WalkObjects(t) {
		propByPath[path] = property
		if !f(path) {
			continue
		}
		fReturnedTrue[path] = struct{}{}
		maxDepth = max(maxDepth, depthOf(path))
		components := strings.Split(path, ".")
		// In addition to adding the property for which f returned true, also
		// add its upper hierarchy.
		for i := range components {
			hierarchy := strings.Join(components[:i+1], ".")
			if !slices.Contains(toAdd, hierarchy) {
				toAdd = append(toAdd, hierarchy)
			}
		}
	}

	// Populate the hierarchy of properties that will be returned, starting with
	// the deepest level properties and gradually moving up the surface, up to
	// the top-level ones.
	for depth := maxDepth; depth >= 1; depth-- {
		for _, path := range toAdd {
			// Only take the properties of this level, ignoring the others.
			if depthOf(path) != depth {
				continue
			}
			property := propByPath[path]
			// Properties that are not objects do not need to be managed, as
			// they have no descendants to populate.
			if property.Type.kind != ObjectKind {
				continue
			}
			// If the function f returned true for this path, then such property
			// is taken as is from the initial schema, and its descendants are
			// preserved, so there is no need to do anything.
			if _, ok := fReturnedTrue[path]; ok {
				continue
			}
			// This is the case to handle, that is a property of type object,
			// with no children, that must be populated with descendants
			// according to the order indicated in orderedToAdd.
			var vl []Property
			for _, p := range toAdd {
				isDescendant := strings.HasPrefix(p, path+".")
				if isSon := isDescendant && depthOf(p) == depth+1; isSon {
					vl = append(vl, propByPath[p])
				}
			}
			property.Type.vl = vl
			propByPath[path] = property
		}
	}

	// Of all the properties to be returned, set aside the top-level ones, which
	// will then be returned in an object.
	var topLevelProps []Property
	for _, path := range toAdd {
		if depthOf(path) == 1 {
			topLevelProps = append(topLevelProps, propByPath[path])
		}
	}

	// No properties to return, so return the invalid Type.
	if len(topLevelProps) == 0 {
		return Type{}
	}

	return Object(topLevelProps)
}

// SubsetFunc returns a subset of the object t, including only the properties
// for which f returns true, maintaining their original order in type t.
// If f returns false for all properties, it returns an invalid schema.
// It panics if t is not an object, or f is nil.
func SubsetFunc(t Type, f func(p Property) bool) Type {
	if t.kind != ObjectKind {
		panic("cannot get a subset of a non-object type")
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

// WalkAll returns an iterator over all the properties in t in a depth-first
// order.
//
// For example:
//
//	for path, property := range WalkAll(t) {
//	    fmt.Printf("%s: %s\n", path, property.Type.Kind)
//	}
//
// WalkAll - unlike WalkObjects - navigates into array and maps, so if a
// property "x" has type array(T) or map(T) and T has the property "y", its path
// is "x.y".
//
// It panics if t is not an object.
func WalkAll(t Type) iter.Seq2[string, Property] {
	return walk(t, true)
}

// WalkObjects returns an iterator over all the object properties in t in a
// depth-first order.
//
// For example:
//
//	for path, property := range WalkObjects(t) {
//	    fmt.Printf("%s: %s\n", path, property.Type.Kind)
//	}
//
// WalkObjects - unlike WalkAll - does not navigate through array or map
// properties, navigating only through object properties.
//
// It panics if t is not an object.
func WalkObjects(t Type) iter.Seq2[string, Property] {
	return walk(t, false)
}

// asRole is a recursive function called by the Type.AsRole method. t must be an
// object type, and role must be either Source or Destination. It returns the
// resulting type and a boolean indicating whether the returned type is
// different from t.
func asRole(t Type, role Role) (Type, bool) {
	var pp = t.vl.([]Property)
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
	return Type{kind: ObjectKind, vl: ppc}, true
}

// walk is the internal function underlying the exported functions WalkAll and
// WalkObjects. descendIntoArrayMap determines whether navigation should descend
// inside the array and map properties, thus navigating inside them, or not do
// so and limit navigation to objects only.
func walk(t Type, descendIntoArrayMap bool) iter.Seq2[string, Property] {
	if t.kind != ObjectKind {
		panic("cannot iterate over a non-object type")
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
			if descendIntoArrayMap {
				for t.kind == MapKind || t.kind == ArrayKind {
					t = t.Elem()
				}
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
