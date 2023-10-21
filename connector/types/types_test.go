//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"testing"

	"golang.org/x/exp/maps"
)

func TestLen(t *testing.T) {

	type Expected struct {
		OK  bool
		Len int
	}

	tests := []struct {
		Type     Type
		Expected Expected
	}{
		{Text(), Expected{false, 0}},
		{Text().WithByteLen(1).WithCharLen(1), Expected{true, 1}},
		{Text().WithByteLen(math.MaxInt32).WithCharLen(math.MaxInt32), Expected{true, math.MaxInt32}},
		{Text().WithByteLen(MaxTextLen).WithCharLen(MaxTextLen), Expected{true, MaxTextLen}},
	}

	for _, test := range tests {
		got, ok := test.Type.ByteLen()
		if ok == test.Expected.OK {
			if got != test.Expected.Len {
				t.Errorf("ByteLen(%d): expected %d, got %d", test.Expected.Len, test.Expected.Len, got)
			}
		} else {
			t.Errorf("ByteLen(%d): expected %t, got %t", test.Expected.Len, test.Expected.OK, ok)
		}
		got, ok = test.Type.CharLen()
		if ok == test.Expected.OK {
			if got != test.Expected.Len {
				t.Errorf("CharLen(%d): expected %d, got %d", test.Expected.Len, test.Expected.Len, got)
			}
		} else {
			t.Errorf("CharLen(%d): expected %t, got %t", test.Expected.Len, test.Expected.OK, ok)
		}
	}

}

func TestAsRole(t *testing.T) {
	cases := []struct {
		object   Type
		role     Role
		expected Type
	}{
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
			role: BothRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
			role: SourceRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
			role: DestinationRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: SourceRole,
				},
			}),
			role:     DestinationRole,
			expected: Type{},
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: DestinationRole,
				},
			}),
			role:     SourceRole,
			expected: Type{},
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: SourceRole,
				},
				{
					Name: "LastName",
					Type: Text(),
					Role: SourceRole,
				},
			}),
			role:     DestinationRole,
			expected: Type{},
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: BothRole,
				},
				{
					Name: "LastName",
					Type: Text(),
					Role: SourceRole,
				},
			}),
			role: DestinationRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: BothRole,
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "ID",
					Type: Text(),
					Role: SourceRole,
				},
				{
					Name: "FirstName",
					Type: Text(),
				},
				{
					Name: "LastName",
					Type: Text(),
				},
			}),
			role: DestinationRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
				{
					Name: "LastName",
					Type: Text(),
				},
			}),
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			got := cas.object.AsRole(cas.role)
			gotValid := got.Valid()
			expectedValid := cas.expected.Valid()
			if !expectedValid && !gotValid {
				// Ok.
				return
			}
			if expectedValid && !gotValid {
				t.Fatal("unexpected invalid schema")
			}
			if !expectedValid && gotValid {
				t.Fatalf("expecting an invalid schema, got %v", got)
			}
			if !cas.expected.EqualTo(got) {
				t.Fatalf("expected schema %v != got %v", cas.expected, got)
			}
		})
	}

}

func TestHasFlatProperties(t *testing.T) {

	tests := []struct {
		Type     Type
		Expected bool
	}{
		{Boolean(), false},
		{Object([]Property{{Name: "email", Type: Text()}}), false},
		{Object([]Property{
			{Name: "address", Type: Object([]Property{
				{Name: "street1", Type: Text()},
				{Name: "street2", Type: Text()},
			})},
		}), false},
		{Object([]Property{
			{Name: "address", Type: Object([]Property{
				{Name: "street1", Type: Text()},
				{Name: "street2", Type: Text()},
			}), Flat: true},
		}), true},
		{Array(Float()), false},
		{Array(Object([]Property{{Name: "email", Type: Text()}})), false},
		{Array(Object([]Property{
			{Name: "name", Type: Object([]Property{
				{Name: "first", Type: Text()},
			})},
		})), false},
		{Array(Object([]Property{
			{Name: "address", Type: Object([]Property{
				{Name: "street1", Type: Text()},
				{Name: "street2", Type: Text()},
			}), Flat: true},
		})), true},
		{Map(Int()), false},
		{Map(Object([]Property{{Name: "email", Type: Text()}})), false},
		{Map(Object([]Property{
			{Name: "email", Type: Text()},
			{Name: "address", Type: Object([]Property{
				{Name: "street1", Type: Text()},
				{Name: "street2", Type: Text()},
			}), Flat: true},
		})), true},
		{Map(Array(Object([]Property{
			{Name: "email", Type: Text()},
			{Name: "address", Type: Object([]Property{
				{Name: "street", Type: Object([]Property{
					{Name: "line1", Type: Text()},
					{Name: "line2", Type: Text()},
				}), Flat: true},
				{Name: "City", Type: Text()},
			}), Flat: true},
		}))), true},
		{Map(Array(Object([]Property{
			{Name: "email", Type: Text()},
			{Name: "address", Type: Object([]Property{
				{Name: "street", Type: Object([]Property{
					{Name: "line1", Type: Text()},
					{Name: "line2", Type: Text()},
				}), Flat: true},
				{Name: "City", Type: Text()},
			}), Flat: true},
		}))).Unflatten(), false},
	}

	for i, test := range tests {
		if got := test.Type.HasFlatProperties(); got != test.Expected {
			t.Errorf("test %d: expected %t, got %t", i, test.Expected, got)
		}
	}

}

func Test_IsValidPropertyPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"", false},
		{".", false},
		{"a", true},
		{"a.b", true},
		{"a.b.c", true},
		{"a..b", false},
		{"a.b.", false},
		{".a.b", false},
	}
	for _, test := range tests {
		if got := IsValidPropertyPath(test.path); got != test.expected {
			t.Errorf("test %q: expected %t, got %t", test.path, test.expected, got)
		}
	}
}

func Test_PropertyByPath(t *testing.T) {
	cases := []struct {
		name     string
		t        Type
		path     Path
		expected Property
		err      error
	}{
		{
			name: "path with single component - property (of type Text) exists",
			t: Object([]Property{
				{Name: "first_name", Type: Text()},
			}),
			path:     Path{"first_name"},
			expected: Property{Name: "first_name", Type: Text()},
			err:      nil,
		},
		{
			name: "path with single component - property does not exist",
			t: Object([]Property{
				{Name: "first_name", Type: Text()},
			}),
			path:     Path{"email"},
			expected: Property{},
			err:      errors.New("property path \"email\" does not exist"),
		},
		{
			name: "path with single component - property (of type Object) exists",
			t: Object([]Property{
				{Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			}),
			path: Path{"billing_address"},
			expected: Property{
				Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			err: nil,
		},
		{
			name: "path with two components - property (of type Text) exists",
			t: Object([]Property{
				{Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			}),
			path:     Path{"billing_address", "street"},
			expected: Property{Name: "street", Type: Text()},
			err:      nil,
		},
		{
			name: "path with two components - property does not exist",
			t: Object([]Property{
				{Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			}),
			path:     Path{"billing_address", "city"},
			expected: Property{},
			err:      errors.New("property path \"billing_address.city\" does not exist"),
		},
		{
			name: "path with three components - property (Text within an Object within an Object) exists",
			t: Object([]Property{
				{Name: "movie", Type: Object([]Property{
					{Name: "director", Type: Object([]Property{
						{Name: "name", Type: Text()},
						{Name: "last_name", Type: Text()},
					})},
				})},
			}),
			path:     Path{"movie", "director", "last_name"},
			expected: Property{Name: "last_name", Type: Text()},
			err:      nil,
		},
		{
			name: "path with three components - second component of path is not an Object",
			t: Object([]Property{
				{Name: "movie", Type: Object([]Property{
					{Name: "director", Type: Object([]Property{
						{Name: "name", Type: Text()},
						{Name: "last_name", Type: Text()},
					})},
					{Name: "writer", Type: Text()},
				})},
			}),
			path:     Path{"movie", "writer", "last_name"},
			expected: Property{},
			err:      errors.New("property path \"movie.writer.last_name\" does not exist"),
		},
	}
	for _, cas := range cases {
		t.Run(cas.name, func(t *testing.T) {
			got, err := cas.t.PropertyByPath(cas.path)
			if err != nil {
				if cas.err == nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if err.Error() != cas.err.Error() {
					t.Fatalf("expected error %q, got error %q", cas.err.Error(), err.Error())
				}
				return
			}
			if cas.err != nil {
				t.Fatalf("expected error %q, got no error", cas.err)
			}
			if err := sameProperty(cas.expected, got); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_ParsePropertyPath(t *testing.T) {

	tests := []struct {
		Path     string
		Expected []string
	}{
		{"", nil},
		{".", nil},
		{"foo", []string{"foo"}},
		{"foo.boo", []string{"foo", "boo"}},
		{".boo", nil},
		{"foo.", nil},
		{"à.boo", nil},
		{"a.b.c", []string{"a", "b", "c"}},
		{"a . b . c", nil},
	}

	for _, test := range tests {
		got, err := ParsePropertyPath(test.Path)
		if err != nil {
			if test.Expected != nil {
				t.Errorf("%q: expected value %#v, got error %q", test.Path, test.Expected, err)
			}
			if err != ErrPathInvalid {
				t.Errorf("%q: expected error ErrPathInvalid, got %q", test.Path, err)
			}
			continue
		}
		if test.Expected == nil {
			t.Errorf("%q: expected ErrPathInvalid error, got value %#v", test.Path, got)
		}
		if len(test.Expected) != len(got) {
			t.Errorf("%q: expected %#v, got %#v", test.Path, test.Expected, got)
		}
		for i, v := range test.Expected {
			if v != got[i] {
				t.Errorf("%q: expected %#v, got %#v", test.Path, test.Expected, got)
			}
		}
	}

}

// sameType reports whether t1 and t2 are the same type. It compares t2 against
// t1.
func sameType(t1, t2 Type) error {
	if t1.pt == t2.pt &&
		t1.unique == t2.unique &&
		t1.real == t2.real &&
		t1.flat == t2.flat &&
		t1.p == t2.p &&
		t1.vl == nil && t2.vl == nil {
		return nil
	}
	// Physical type.
	if t1.pt != t2.pt {
		if !t2.pt.Valid() {
			return fmt.Errorf("unknown physical type %d", t2.pt)
		}
		return fmt.Errorf("expected physical type %d, got %d", t1.pt, t2.pt)
	}
	// Minimum and maximum.
	switch t1.pt {
	case PtInt, PtInt8, PtInt16, PtInt24:
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", t1.p, t1.s, t2.p, t2.s)
		}
	case PtUInt, PtUInt8, PtUInt16, PtUInt24:
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", uint32(t1.p), uint32(t1.s), uint32(t2.p), uint32(t2.s))
		}
	case PtInt64, PtUInt64, PtFloat, PtFloat32:
		if t1.vl != t2.vl {
			if t1.vl == nil {
				return fmt.Errorf("expected no range, got %v", t2.vl)
			} else if t2.vl == nil {
				return fmt.Errorf("expected range %v, got no range", t1.vl)
			} else {
				return fmt.Errorf("expected range %v, got %v", t1.vl, t2.vl)
			}
		}
	case PtDecimal:
		if vl1, ok := t1.vl.(decimalRange); ok {
			vl2, ok := t2.vl.(decimalRange)
			if !ok || !vl1.min.Equal(vl2.min) || !vl1.max.Equal(vl2.max) {
				return fmt.Errorf("expected range %v, got range %v", t1.vl, t2.vl)
			}
		} else if t2.vl != nil {
			return fmt.Errorf("expected no-range, got range %v", t2.vl)
		}
	}
	// Real.
	if t1.real != t2.real {
		return fmt.Errorf("expected real %t, got %t", t1.real, t2.real)
	}
	// Precision, byte length or items minimum length.
	if t1.p != t2.p {
		switch t1.pt {
		case PtDecimal:
			return fmt.Errorf("expected precision %d, got %d", t1.p, t2.p)
		case PtText:
			return fmt.Errorf("expected byte length %d, got %d", t1.p, t2.p)
		case PtArray:
			return fmt.Errorf("expected items minimum length %d, got %d", t1.p, t2.p)
		}
		return fmt.Errorf("expected p == 0, got %d", t2.p)
	}
	// Scale, character length or items maximum length.
	if t1.s != t2.s {
		switch t1.pt {
		case PtDecimal:
			return fmt.Errorf("expected scale %d, got %d", t1.s, t2.s)
		case PtText:
			return fmt.Errorf("expected character length %d, got %d", t1.s, t2.s)
		case PtArray:
			return fmt.Errorf("expected items maximum length %d, got %d", t1.s, t2.s)
		}
		return fmt.Errorf("expected s == 0, got %d", t2.s)
	}
	// Regular expression or values.
	if t1.pt == PtText {
		switch vl1 := t1.vl.(type) {
		case nil:
			if t2.vl != nil {
				return fmt.Errorf("expected no regular expression or values, got a %T value", t2.vl)
			}
		case *regexp.Regexp:
			if t2.vl == nil {
				return errors.New("expected regular expression, got nil")
			}
			vl2, ok := t2.vl.(*regexp.Regexp)
			if !ok {
				return fmt.Errorf("expected regular expression, got a %T value", t2.vl)
			}
			if vl1.String() != vl2.String() {
				return fmt.Errorf("expected regular expression %s, got %s", vl1.String(), vl2.String())
			}
		case []string:
			if t2.vl == nil {
				return errors.New("expected values, got nil")
			}
			vl2, ok := t2.vl.([]string)
			if !ok {
				return fmt.Errorf("expected values, got %T value", t2.vl)
			}
			if vl2 == nil {
				return fmt.Errorf("unexpected []string(nil)")
			}
			if len(vl1) != len(vl2) {
				return fmt.Errorf("expected %d values values, got %d", len(vl1), len(vl2))
			}
			for i, v1 := range vl1 {
				if v2 := (vl2)[i]; v1 != v2 {
					return fmt.Errorf("expected value %q, got %q", v1, v2)
				}
			}
		}
	}
	// Unique items and item type.
	if t1.pt == PtArray {
		if t1.unique != t2.unique {
			if t1.unique {
				return errors.New("expected unique items, got non-unique")
			}
			return errors.New("expected non-unique items, got unique")
		}
		if t2.vl == nil {
			return errors.New("expected item type, got nil")
		}
		if err := sameType(t1.vl.(Type), t2.vl.(Type)); err != nil {
			return err
		}
	}
	// Properties.
	if t1.pt == PtObject {
		if t2.vl == nil {
			return errors.New("expected properties, got nil")
		}
		properties2, ok := t2.vl.([]Property)
		if !ok {
			return fmt.Errorf("expected properties, got a %T value", t2.vl)
		}
		if properties2 == nil {
			return fmt.Errorf("unexpected []Property(nil)")
		}
		properties1 := t1.vl.([]Property)
		if len(properties1) != len(properties2) {
			return fmt.Errorf("expected %d properties, got %d", len(properties1), len(properties2))
		}
		for i, p1 := range properties1 {
			p2 := properties2[i]
			err := sameProperty(p1, p2)
			if err != nil {
				return err
			}
		}
	}
	// Value type.
	if t1.pt == PtMap {
		if t2.vl == nil {
			return errors.New("expected value type, got nil")
		}
		if err := sameType(t1.vl.(Type), t2.vl.(Type)); err != nil {
			return err
		}
	}
	return nil
}

// sameProperty reports whether p1 and p2 are the same property. It compares p2
// against p1.
func sameProperty(p1, p2 Property) error {
	if p1.Name != p2.Name {
		return fmt.Errorf("expected property name %q, got %q", p1.Name, p2.Name)
	}
	if p1.Label != p2.Label {
		return fmt.Errorf("expected property label %q, got %q", p1.Label, p2.Label)
	}
	if p1.Description != p2.Description {
		return fmt.Errorf("expected property description %q, got %q", p1.Description, p2.Description)
	}
	switch ph1 := p1.Placeholder.(type) {
	case nil:
		if p2.Placeholder != nil {
			return fmt.Errorf("expected property placeholder nil, got a %T value", p2.Placeholder)
		}
	case string:
		ph2, ok := p2.Placeholder.(string)
		if !ok {
			return fmt.Errorf("expected property placeholder with string type, got %T", p2.Placeholder)
		}
		if ph1 != ph2 {
			return fmt.Errorf("expected property placeholder %q, got %q", p1.Placeholder, p2.Placeholder)
		}
	case map[string]string:
		ph2, ok := p2.Placeholder.(map[string]string)
		if !ok {
			return fmt.Errorf("expected property placeholder with map[string]string type, got %T", p2.Placeholder)
		}
		if !maps.Equal(ph1, ph2) {
			return fmt.Errorf("expected property placeholder %#v, got %#v", ph1, ph2)
		}
	default:
		return fmt.Errorf("expected property placeholder %q, got a %T value", p1.Placeholder, p2.Placeholder)
	}
	if p1.Role != p2.Role {
		return fmt.Errorf("expected property key 'role' with value %s, got %s", p1.Role, p2.Role)
	}
	if err := sameType(p1.Type, p2.Type); err != nil {
		return err
	}
	if p1.Required != p2.Required {
		return fmt.Errorf("expected property key 'required' with value %t, got %t", p1.Required, p2.Required)
	}
	if p1.Nullable != p2.Nullable {
		return fmt.Errorf("expected property key 'nullable' with value %t, got %t", p1.Nullable, p2.Nullable)
	}
	if p1.Flat != p2.Flat {
		return fmt.Errorf("expected property key 'flat' with value %t, got %t", p1.Flat, p2.Flat)
	}
	return nil
}
