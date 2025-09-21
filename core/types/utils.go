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
	for path, property := range t.Properties().WalkObjects() {
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
			names := make(map[string]int, len(vl))
			for i, p := range vl {
				names[p.Name] = i
			}
			property.Type.vl = Properties{properties: vl, names: names}
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

// asRole is a recursive function called by the Type.AsRole method. t must be an
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
