// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package schemas

import (
	"fmt"
	"slices"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
)

// Error represents an error with a schema.
type Error struct {
	Msg string
}

func (err *Error) Error() string {
	return err.Msg
}

// CheckAlignment checks whether schema t1 is aligned with schema t2. It
// returns a *Error error if t1 is not aligned with t2.
//
// Alignment means that all properties of t1 must be present in t2 with the
//   - same Type and Nullable.
//   - same ReadOptional, if exportMode is nil.
//   - same CreateRequired, if exportMode is CreateOnly or CreateOrUpdate.
//   - same UpdateRequired, if exportMode is UpdateOnly or CreateOrUpdate.
//
// Furthermore, if exportMode is not nil, t1 must include all properties in t2
// that are required for creation or the update based on exportMode.
//
// An invalid schema is handled as if it were an object without properties.
func CheckAlignment(t1, t2 types.Type, exportMode *state.ExportMode) error {
	if t1.Kind() == types.InvalidKind {
		if t2.Kind() == types.InvalidKind {
			return nil
		}
		if exportMode == nil {
			return nil
		}
		for path, p := range t2.Properties().WalkAll() {
			if *exportMode != state.UpdateOnly && p.CreateRequired {
				return &Error{Msg: fmt.Sprintf("%q property is required for creation", path)}
			}
			if *exportMode != state.CreateOnly && p.UpdateRequired {
				return &Error{Msg: fmt.Sprintf("%q property is required for update", path)}
			}
		}
		return nil
	}
	if t2.Kind() == types.InvalidKind {
		for _, p := range t1.Properties().All() {
			return &Error{Msg: fmt.Sprintf("%q property no longer exists", p.Name)}
		}
	}
	return checkTypeAlignment("", t1, t2, exportMode)
}

// checkTypeAlignment is called by CheckAlignment to check if t1 is aligned
// with t2.
func checkTypeAlignment(name string, t1, t2 types.Type, exportMode *state.ExportMode) error {
	k1 := t1.Kind()
	k2 := t2.Kind()
	if k1 == types.ObjectKind {
		if k2 != types.ObjectKind {
			return &Error{Msg: fmt.Sprintf("%q property's type has changed from object to %s", name, t2)}
		}
		properties1 := t1.Properties()
		properties2 := t2.Properties()
		for _, p1 := range properties1.All() {
			path := p1.Name
			if name != "" {
				path = name + "." + path
			}
			p2, ok := properties2.ByName(p1.Name)
			if !ok {
				return &Error{Msg: fmt.Sprintf("%q property no longer exists", path)}
			}
			if exportMode != nil {
				if p1.CreateRequired != p2.CreateRequired && *exportMode != state.UpdateOnly {
					if p1.CreateRequired {
						return &Error{Msg: fmt.Sprintf("%q property was previously required for creation but is no longer", path)}
					}
					return &Error{Msg: fmt.Sprintf("%q property was not previously required for creation but it is now required", path)}
				}
				if p1.UpdateRequired != p2.UpdateRequired && *exportMode != state.CreateOnly {
					if p1.UpdateRequired {
						return &Error{Msg: fmt.Sprintf("%q property was previously required for the update but is no longer", path)}
					}
					return &Error{Msg: fmt.Sprintf("%q property was not previously required for the update but it is now required", path)}
				}
			}
			if exportMode == nil && p1.ReadOptional != p2.ReadOptional {
				if p1.ReadOptional {
					return &Error{Msg: fmt.Sprintf("%q property was previously optional but it is now non-optional", path)}
				}
				return &Error{Msg: fmt.Sprintf("%q property was previously non-optional but it is now optional", path)}
			}
			if p1.Nullable != p2.Nullable {
				if p1.Nullable {
					return &Error{Msg: fmt.Sprintf("%q property was previously nullable but it is no longer nullable", path)}
				}
				return &Error{Msg: fmt.Sprintf("%q property was previously non-nullable but is now nullable", path)}
			}
			err := checkTypeAlignment(path, p1.Type, p2.Type, exportMode)
			if err != nil {
				return err
			}
		}
		if exportMode == nil {
			return nil
		}
		for _, p := range properties2.All() {
			checkCreate := p.CreateRequired && *exportMode != state.UpdateOnly
			checkUpdate := p.UpdateRequired && *exportMode != state.CreateOnly
			if !checkCreate && !checkUpdate {
				continue
			}
			if properties1.ContainsName(p.Name) {
				continue
			}
			if checkCreate {
				return &Error{Msg: fmt.Sprintf(`"%s.%s" property is required for creation but is not present in the schema`, name, p.Name)}
			}
			return &Error{Msg: fmt.Sprintf(`"%s.%s" property is required for update but is not present in the schema`, name, p.Name)}
		}
		return nil
	}
	if types.Equal(t1, t2) {
		return nil
	}
	if k1 != k2 {
		return &Error{Msg: fmt.Sprintf("%q property's type has changed from %s to %s", name, t1, t2)}
	}
	switch k1 {
	case types.StringKind:
		c1, ok1 := t1.MaxLength()
		c2, ok2 := t2.MaxLength()
		if c1 != c2 || ok1 != ok2 {
			var v1, v2 any = "unbounded", "unbounded"
			if ok1 {
				v1 = c1
			}
			if ok2 {
				v2 = c2
			}
			return &Error{Msg: fmt.Sprintf("character length of the %q property's type has changed from %v to %v", name, v1, v2)}
		}
		b1, ok1 := t1.MaxBytes()
		b2, ok2 := t2.MaxBytes()
		if b1 != b2 || ok1 != ok2 {
			var v1, v2 any = "unbounded", "unbounded"
			if ok1 {
				v1 = b1
			}
			if ok2 {
				v2 = b2
			}
			return &Error{Msg: fmt.Sprintf("byte length of the %q property's type has changed from %v to %v", name, v1, v2)}
		}
		vs1 := t1.Values()
		vs2 := t2.Values()
		if vs1 != nil && vs2 == nil {
			return &Error{Msg: fmt.Sprintf("%q property was previously limited to specific values but is now unrestricted", name)}
		}
		if vs1 == nil && vs2 != nil {
			return &Error{Msg: fmt.Sprintf("%q property was previously unrestricted but is now limited to specific values", name)}
		}
		for _, v1 := range vs1 {
			if !slices.Contains(vs2, v1) {
				return &Error{Msg: fmt.Sprintf("%q property allowed value %q but it is no longer allowed", name, v1)}
			}
		}
		if len(vs1) < len(vs2) {
			for _, v2 := range vs2 {
				if !slices.Contains(vs1, v2) {
					return &Error{Msg: fmt.Sprintf("%q property previously disallowed value %q but it now allows it", name, v2)}
				}
			}
		}
		re1 := t1.Pattern()
		re2 := t2.Pattern()
		var v1, v2 = "none", "none"
		if re1 != nil {
			v1 = `"` + re1.String() + `"`
		}
		if re2 != nil {
			v2 = `"` + re2.String() + `"`
		}
		return &Error{Msg: fmt.Sprintf("regular expression of the %q property's type has changed from %s to %s", name, v1, v2)}
	case types.IntKind:
		if t1.IsUnsigned() != t2.IsUnsigned() {
			if t1.IsUnsigned() {
				return &Error{Msg: fmt.Sprintf("%q property's type has changed from unsigned %s to %s", name, t1, t2)}
			}
			return &Error{Msg: fmt.Sprintf("%q property's type has changed from %s to unsigned %s", name, t1, t2)}
		}
		if t1.BitSize() != t2.BitSize() {
			return &Error{Msg: fmt.Sprintf("%q property's type has changed from %s to %s", name, t1, t2)}
		}
		if t1.IsUnsigned() {
			min1, max1 := t1.UnsignedRange()
			min2, max2 := t2.UnsignedRange()
			return &Error{Msg: fmt.Sprintf("range of the %q property's type has changed from [%d,%d] to [%d,%d]", name, min1, max1, min2, max2)}
		}
		min1, max1 := t1.IntRange()
		min2, max2 := t2.IntRange()
		return &Error{Msg: fmt.Sprintf("range of the %q property's type has changed from [%d,%d] to [%d,%d]", name, min1, max1, min2, max2)}
	case types.FloatKind:
		if t1.BitSize() != t2.BitSize() {
			return &Error{Msg: fmt.Sprintf("%q property's type has changed from %s to %s", name, t1, t2)}
		}
		if t1.IsReal() && !t2.IsReal() {
			return &Error{Msg: fmt.Sprintf("%q property's type has changed from real to non-real", name)}
		}
		if !t1.IsReal() && t2.IsReal() {
			return &Error{Msg: fmt.Sprintf("%q property's type has changed from non-real to real", name)}
		}
		min1, max1 := t1.FloatRange()
		min2, max2 := t2.FloatRange()
		return &Error{Msg: fmt.Sprintf("range of the %q property's type has changed from [%g,%g] to [%g,%g]", name, min1, max1, min2, max2)}
	case types.DecimalKind:
		p1 := t1.Precision()
		p2 := t2.Precision()
		if p1 != p2 {
			return &Error{Msg: fmt.Sprintf("precision of the %q property's type has changed from %d to %d", name, p1, p2)}
		}
		s1 := t1.Scale()
		s2 := t2.Scale()
		if s1 != s2 {
			return &Error{Msg: fmt.Sprintf("scale of the %q property's type has changed from %d to %d", name, s1, s2)}
		}
		min1, max1 := t1.DecimalRange()
		min2, max2 := t2.DecimalRange()
		return &Error{Msg: fmt.Sprintf("range of %q property's type has changed from [%s,%s] to [%s,%s]", name, min1, max1, min2, max2)}
	case types.ArrayKind:
		err := checkTypeAlignment(name+"[]", t1.Elem(), t2.Elem(), exportMode)
		if err != nil {
			return err
		}
		n1 := t1.MinElements()
		n2 := t2.MinElements()
		if n1 != n2 {
			return &Error{Msg: fmt.Sprintf("minimum number of %q property elements has been changed from %d to %d", name, n1, n2)}
		}
		n1 = t1.MaxElements()
		n2 = t2.MaxElements()
		if n1 != n2 {
			return &Error{Msg: fmt.Sprintf("minimum number of %q property elements has been changed from %d to %d", name, n1, n2)}
		}
		if t1.Unique() {
			return &Error{Msg: fmt.Sprintf("%q property elements were initially required to be unique, but it is no longer required", name)}
		}
		return &Error{Msg: fmt.Sprintf("%q property elements were not required to be unique, but now it is required", name)}
	case types.MapKind:
		return checkTypeAlignment(name+"[]", t1.Elem(), t2.Elem(), exportMode)
	default:
		panic("unexpected kind")
	}
}
