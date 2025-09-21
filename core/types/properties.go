//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package types

import (
	"iter"
	"strings"
)

// Properties holds the properties of an object.
type Properties struct {
	properties []Property
	names      map[string]int
}

// All returns an iterator over the properties.
func (pp Properties) All() iter.Seq2[int, Property] {
	n := len(pp.properties)
	return func(yield func(i int, property Property) bool) {
		for i := 0; i < n; i++ {
			if !yield(i, pp.properties[i]) {
				return
			}
		}
	}
}

const (
	invalidNameMsg = "invalid property name"
	invalidPathMsg = "invalid property path"
)

// ByName returns the property with the given name and a boolean indicating
// whether it exists. If the name is invalid or not found, it returns the zero
// Property and false.
func (pp Properties) ByName(name string) (Property, bool) {
	if i, ok := pp.names[name]; ok {
		return pp.properties[i], true
	}
	return Property{}, false
}

// ByPath returns the property with the given path. If the property does not
// exist, it returns the last valid property found (if any), and a
// PathNotExistError error.
//
// Unlike Walk, it does not traverse through arrays and maps. If path is "x.y"
// and the type of "x" is not an object, it returns a PathNotExistError error.
//
// It panics if path is not a valid property path.
func (pp Properties) ByPath(path string) (Property, error) {
	var p *Property
	name, rest := "", path
	for {
		name, rest, _ = strings.Cut(rest, ".")
		i, ok := pp.names[name]
		if !ok {
			break
		}
		if rest == "" {
			return pp.properties[i], nil
		}
		p = &pp.properties[i]
		if p.Type.kind != ObjectKind {
			_, rest, _ = strings.Cut(rest, ".")
			break
		}
		pp = p.Type.vl.(Properties)
	}
	if !IsValidPropertyPath(path) {
		panic(invalidPathMsg)
	}
	err := PathNotExistError{strings.TrimSuffix(strings.TrimSuffix(path, rest), ".")}
	if p == nil {
		return Property{}, err
	}
	return *p, err
}

// ByPathSlice is like ByPath but takes a slice of property names as the path.
// It also panics if path is empty.
func (pp Properties) ByPathSlice(path []string) (Property, error) {
	last := len(path) - 1
	if last == -1 {
		panic("path is empty")
	}
	var p *Property
	var i int
	var name string
	for i, name = range path {
		j, ok := pp.names[name]
		if !ok {
			break
		}
		if i == last {
			return pp.properties[j], nil
		}
		p = &pp.properties[j]
		if p.Type.kind != ObjectKind {
			i++
			break
		}
		pp = p.Type.vl.(Properties)
	}
	for _, name := range path {
		if !IsValidPropertyName(name) {
			panic(invalidPathMsg)
		}
	}
	err := PathNotExistError{strings.Join(path[:i+1], ".")}
	if p == nil {
		return Property{}, err
	}
	return *p, err
}

// ContainsName reports whether a property with the given name exists.
// If the name is invalid or not found, it returns the zero Property and false.
func (pp Properties) ContainsName(name string) bool {
	_, ok := pp.names[name]
	return ok
}

// ContainsPath reports whether property with the given path exists.
//
// If path is "x.y" and the property "x" has type array(T) or map(T), it reports
// whether T has the property "y".
//
// It panics if path is not a valid property path.
func (pp Properties) ContainsPath(path string) bool {
	var name string
	var found bool
Object:
	for {
		name, path, found = strings.Cut(path, ".")
		i, ok := pp.names[name]
		if !ok {
			if !IsValidPropertyName(name) {
				panic(invalidPathMsg)
			}
			return false
		}
		if path == "" {
			if found {
				panic(invalidPathMsg)
			}
			return true
		}
		t := pp.properties[i].Type
		for {
			switch t.kind {
			case ObjectKind:
				pp = t.vl.(Properties)
				continue Object
			case ArrayKind, MapKind:
				t = t.Elem()
			default:
				if !IsValidPropertyPath(path) {
					panic(invalidPathMsg)
				}
				return false
			}
		}
	}
}

// Count returns the number of properties.
func (pp Properties) Count() int {
	return len(pp.properties)
}

// Names returns a copy of the property names.
func (pp Properties) Names() []string {
	properties := pp.properties
	names := make([]string, len(properties))
	for i := 0; i < len(properties); i++ {
		names[i] = properties[i].Name
	}
	return names
}

// Slice returns a copy of the properties.
func (pp Properties) Slice() []Property {
	properties := make([]Property, len(pp.properties))
	copy(properties, pp.properties)
	return properties
}

// WalkAll returns an iterator over all properties in depth-first order.
//
// Example:
//
//	for path, property := range properties.WalkAll() {
//	    fmt.Printf("%s: %s\n", path, property.Type.Kind)
//	}
//
// Unlike WalkObjects, WalkAll also descends into array and map elements.
// For a property "x" of type array(T) or map(T) with a sub-property "y",
// the resulting path is "x.y".
func (pp Properties) WalkAll() iter.Seq2[string, Property] {
	return pp.walk(true)
}

// WalkObjects returns an iterator over all properties in depth-first order.
//
// Example:
//
//	for path, property := range properties.WalkObjects() {
//	    fmt.Printf("%s: %s\n", path, property.Type.Kind)
//	}
//
// Unlike WalkAll, WalkObjects does not descend into array or map elements.
// Iteration is limited to object properties only.
func (pp Properties) WalkObjects() iter.Seq2[string, Property] {
	return pp.walk(false)
}

// walk is the shared implementation of WalkAll and WalkObjects.
// If traverseArrayMap is true, iteration also descends into array and map
// elements. If false, iteration is limited to object properties only.
func (pp Properties) walk(traverseArrayMap bool) iter.Seq2[string, Property] {
	return func(yield func(path string, property Property) bool) {
		type entry struct {
			base string
			prop *Property
		}
		n := len(pp.properties)
		entries := make([]entry, n)
		for i := 0; i < n; i++ {
			entries[i].prop = &pp.properties[n-1-i]
		}
		for len(entries) > 0 {
			var e entry
			n := len(entries)
			e, entries = entries[n-1], entries[:n-1]
			t := e.prop.Type
			if traverseArrayMap {
				for t.kind == MapKind || t.kind == ArrayKind {
					t = t.Elem()
				}
			}
			if t.kind == ObjectKind {
				properties := t.vl.(Properties).properties
				for i := len(properties) - 1; i >= 0; i-- {
					entries = append(entries, entry{base: e.base + e.prop.Name + ".", prop: &properties[i]})
				}
			}
			if !yield(e.base+e.prop.Name, *e.prop) {
				return
			}
		}
	}
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
