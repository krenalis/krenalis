// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/krenalis/krenalis/tools/types"
)

// requiredProperties retains only properties required for one purpose and the
// containers needed to reach them.
type requiredProperties struct {
	kind          types.Kind
	properties    []requiredProperty
	width         int  // property count of the complete object
	hasChildren   bool // object properties have descendant checks
	assumePresent bool // an external matching value creates this object
	elem          *requiredProperties
}

type requiredProperty struct {
	name     string
	required bool
	children *requiredProperties
}

// newRequiredProperties builds required-property checks for t and purpose.
// If source is valid, it skips checks guaranteed by its read constraints.
// A valid source must have a compatible value representation.
// It returns nil if no checks are needed.
func newRequiredProperties(t, source types.Type, purpose Purpose) *requiredProperties {
	switch t.Kind() {
	case types.ObjectKind:
		var properties []requiredProperty
		hasChildren := false
		for _, p := range t.Properties().All() {
			required := purpose == Create && p.CreateRequired || purpose == Update && p.UpdateRequired
			var sourceType types.Type
			if source.IsObject() {
				if sp, ok := source.Properties().ByName(p.Name); ok {
					sourceType = sp.Type
					if !sp.ReadOptional {
						required = false
					}
				}
			}
			children := newRequiredProperties(p.Type, sourceType, purpose)
			if required || children != nil {
				properties = append(properties, requiredProperty{name: p.Name, required: required, children: children})
				hasChildren = hasChildren || children != nil
			}
		}
		if len(properties) == 0 {
			return nil
		}
		return &requiredProperties{
			kind:        types.ObjectKind,
			properties:  properties,
			width:       t.Properties().Len(),
			hasChildren: hasChildren,
		}
	case types.ArrayKind, types.MapKind:
		var sourceElem types.Type
		if source.Kind() == t.Kind() {
			sourceElem = source.Elem()
		}
		elem := newRequiredProperties(t.Elem(), sourceElem, purpose)
		if elem == nil {
			return nil
		}
		return &requiredProperties{kind: t.Kind(), elem: elem}
	default:
		return nil
	}
}

// maxRequiredPropertyKeyBytes limits the length of map keys included in
// validation errors.
const maxRequiredPropertyKeyBytes = 128

// missingPath returns a path to a missing required property, or an empty string
// if none is missing. Object property names are prefixed with a dot so callers
// can prepend a property name, array index, or map key without inspecting the
// returned path.
// v must use the value representation of the schema used to build r, and
// object keys must be declared in that schema.
func (r *requiredProperties) missingPath(v any) string {
	if r == nil || v == nil && !r.assumePresent {
		return ""
	}
	switch r.kind {
	case types.ObjectKind:
		var object map[string]any
		if v != nil {
			object = v.(map[string]any)
		}
		full := len(object) == r.width
		if full && !r.hasChildren {
			return ""
		}
		for _, p := range r.properties {
			if full && p.children == nil {
				continue
			}
			value, ok := object[p.name]
			if !ok {
				if p.required {
					return "." + p.name
				}
				if p.children == nil || !p.children.assumePresent {
					continue
				}
			}
			if p.children != nil {
				if path := p.children.missingPath(value); path != "" {
					return "." + p.name + path
				}
			}
		}
	case types.ArrayKind:
		flatObject := r.elem.kind == types.ObjectKind && !r.elem.hasChildren
		for i, value := range v.([]any) {
			if flatObject && value != nil && len(value.(map[string]any)) == r.elem.width {
				continue
			}
			if path := r.elem.missingPath(value); path != "" {
				return "[" + strconv.Itoa(i) + "]" + path
			}
		}
	case types.MapKind:
		for key, value := range v.(map[string]any) {
			if path := r.elem.missingPath(value); path != "" {
				if len(key) > maxRequiredPropertyKeyBytes {
					key = key[:maxRequiredPropertyKeyBytes] + "…"
				}
				return "[" + strconv.Quote(key) + "]" + path
			}
		}
	default:
		panic(fmt.Sprintf("core/internal/transformers/mappings: unexpected required properties kind %v", r.kind))
	}
	return ""
}

// providePath removes the check for path and treats its ancestors as present
// even when they are missing from the output.
// path must identify a scalar property in the schema used to build r.
func (r *requiredProperties) providePath(path []string) *requiredProperties {
	if r == nil {
		return nil
	}
	for i := range r.properties {
		p := &r.properties[i]
		if p.name != path[0] {
			continue
		}
		p.required = false
		if len(path) == 1 {
			p.children = nil
		} else {
			p.children = p.children.providePath(path[1:])
			if p.children != nil {
				p.children.assumePresent = true
			}
		}
		if p.children == nil {
			r.properties = slices.Delete(r.properties, i, i+1)
		}
		if len(r.properties) == 0 {
			return nil
		}
		r.hasChildren = false
		for _, p := range r.properties {
			r.hasChildren = r.hasChildren || p.children != nil
		}
		return r
	}
	return r
}

// removeGuaranteedChecks removes checks below path for properties whose
// presence is guaranteed by value-compatible inputs. Paths absent from the
// plan require no checks.
func (r *requiredProperties) removeGuaranteedChecks(path []string, t, source types.Type, purpose Purpose) *requiredProperties {
	if r == nil {
		return nil
	}
	for i := range r.properties {
		p := &r.properties[i]
		if p.name != path[0] {
			continue
		}
		if len(path) == 1 {
			p.children = newRequiredProperties(t, source, purpose)
		} else {
			p.children = p.children.removeGuaranteedChecks(path[1:], t, source, purpose)
		}
		if !p.required && p.children == nil {
			r.properties = slices.Delete(r.properties, i, i+1)
		}
		if len(r.properties) == 0 {
			return nil
		}
		r.hasChildren = false
		for _, p := range r.properties {
			r.hasChildren = r.hasChildren || p.children != nil
		}
		return r
	}
	return r
}
