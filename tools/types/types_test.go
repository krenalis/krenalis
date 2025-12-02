// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"testing"
)

func Test_BitSize(t *testing.T) {

	for _, s := range []int{8, 16, 24, 32, 64} {
		if got := Int(s).BitSize(); s != got {
			t.Errorf("Int(%d).BitSize(): expected %d, got %d", s, s, got)
		}
		if got := Int(s).Unsigned().BitSize(); s != got {
			t.Errorf("Int(%d).Unsigned().BitSize(): expected %d, got %d", s, s, got)
		}
	}

	for _, s := range []int{32, 64} {
		if got := Float(s).BitSize(); s != got {
			t.Errorf("Float(%d).BitSize(): expected %d, got %d", s, s, got)
		}
	}

}

func Test_NormalizedUTF8(t *testing.T) {
	invalidUTF8 := string([]byte{0xff})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "EmptyString",
			input: "",
			want:  "",
		},
		{
			name:  "ASCII",
			input: "Simple ASCII text",
			want:  "Simple ASCII text",
		},
		{
			name:  "AlreadyNormalized",
			input: "Caf\u00e9",
			want:  "Caf\u00e9",
		},
		{
			name:  "NeedsCompositionSingleRune",
			input: "Cafe\u0301",
			want:  "Caf\u00e9",
		},
		{
			name:  "NeedsCompositionMultipleRunes",
			input: "A\u030Angstro\u0308m",
			want:  "\u00c5ngstr\u00f6m",
		},
		{
			name:    "InvalidUTF8",
			input:   invalidUTF8,
			wantErr: "invalid UTF-8 encoding",
		},
		{
			name:    "ContainsNUL",
			input:   "text\x00with-nul",
			wantErr: "contains NUL byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizedUTF8(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func Test_Parameter(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := Parameter("custom")
		if !p.Generic() || p.Kind() != InvalidKind {
			t.Fatalf("unexpected parameter type: %#v", p)
		}
	})
	t.Run("invalid name", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Parameter("1invalid")
	})
	t.Run("kind name", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Parameter("int")
	})
}

func Test_Ranges(t *testing.T) {

	for _, s := range []int{8, 16, 24, 32, 64} {
		if min, max := Int(s).IntRange(); min != -1<<(s-1) || max != 1<<(s-1)-1 {
			t.Errorf("Int(%d).IntRange(): expected (%d, %d), got (%d, %d)", s, -1<<(s-1), 1<<(s-1)-1, min, max)
		}
		if min, max := Int(s).Unsigned().UnsignedRange(); min != 0 || max != 1<<s-1 {
			t.Errorf("Int(%d).Unsigned().UnsignedRange(): expected (%d, %d), got (%d, %d)", s, 0, 1<<s-1, min, max)
		}
	}

	for _, s := range []int{32, 64} {
		if min, max := Float(s).FloatRange(); min != math.Inf(-1) || max != math.Inf(1) {
			t.Errorf("Float(32).FloatRange(): expected (-Inf, +Inf), got (%f, %f)", min, max)
		}
	}

	if min, max := Float(32).Real().FloatRange(); min != -math.MaxFloat32 || max != math.MaxFloat32 {
		t.Errorf("Float(32).FloatRange(): expected (%f, %f), got (%f, %f)", -math.MaxFloat32, math.MaxFloat32, min, max)
	}
	if min, max := Float(64).Real().FloatRange(); min != -math.MaxFloat64 || max != math.MaxFloat64 {
		t.Errorf("Float(64).FloatRange(): expected (%f, %f), got (%f, %f)", -math.MaxFloat64, math.MaxFloat64, min, max)
	}

}

func Test_Len(t *testing.T) {

	type Expected struct {
		OK  bool
		Len int
	}

	tests := []struct {
		Type     Type
		Expected Expected
	}{
		{String(), Expected{false, 0}},
		{String().WithMaxBytes(1).WithMaxLength(1), Expected{true, 1}},
		{String().WithMaxBytes(math.MaxInt32).WithMaxLength(math.MaxInt32), Expected{true, math.MaxInt32}},
		{String().WithMaxBytes(MaxStringLen).WithMaxLength(MaxStringLen), Expected{true, MaxStringLen}},
	}

	for _, test := range tests {
		got, ok := test.Type.MaxBytes()
		if ok == test.Expected.OK {
			if got != test.Expected.Len {
				t.Errorf("MaxBytes(%d): expected %d, got %d", test.Expected.Len, test.Expected.Len, got)
			}
		} else {
			t.Errorf("MaxBytes(%d): expected %t, got %t", test.Expected.Len, test.Expected.OK, ok)
		}
		got, ok = test.Type.MaxLength()
		if ok == test.Expected.OK {
			if got != test.Expected.Len {
				t.Errorf("MaxLength(%d): expected %d, got %d", test.Expected.Len, test.Expected.Len, got)
			}
		} else {
			t.Errorf("MaxLength(%d): expected %t, got %t", test.Expected.Len, test.Expected.OK, ok)
		}
	}

}

func Test_ObjectOf_Errors(t *testing.T) {

	// Test InvalidPropertyNameError.
	_, err := ObjectOf([]Property{
		{Name: "firstName", Type: String()},
		{Name: "last name", Type: String()},
		{Name: "phone", Type: String()},
	})
	if err == nil {
		t.Error("expected InvalidPropertyNameError error, got no error")
	}
	if err2, ok := err.(InvalidPropertyNameError); ok {
		if err2.Index != 1 {
			t.Errorf("expected index 1, got %d", err2.Index)
		}
		if err2.Name != "last name" {
			t.Errorf("expected name \"last name\" , got %q", err2.Name)
		}
	} else {
		t.Errorf("expected InvalidPropertyNameError error, got a %T error", err)
	}

	// Test RepeatedPropertyNameError.
	_, err = ObjectOf([]Property{
		{Name: "firstName", Type: String()},
		{Name: "lastName", Type: String()},
		{Name: "firstName", Type: String()},
	})
	if err == nil {
		t.Error("expected RepeatedPropertyNameError error, got no error")
	}
	if err2, ok := err.(RepeatedPropertyNameError); ok {
		if err2.Index1 != 0 {
			t.Errorf("expected index1 0, got %d", err2.Index1)
		}
		if err2.Index2 != 2 {
			t.Errorf("expected index2 2, got %d", err2.Index2)
		}
		if err2.Name != "firstName" {
			t.Errorf("expected name \"firstName\" , got %q", err2.Name)
		}
	} else {
		t.Errorf("expected RepeatedPropertyNameError error, got a %T error", err)
	}

}

func Test_Unique(t *testing.T) {
	t.Run("array unique", func(t *testing.T) {
		a := Array(String()).WithUnique()
		if !a.Unique() {
			t.Fatal("expected unique true")
		}
	})
	t.Run("non array unique", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		String().Unique()
	})
	t.Run("invalid element type", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Array(JSON()).WithUnique()
	})
}

func Test_WithUnsignedRange(t *testing.T) {
	t.Run("uint8 range", func(t *testing.T) {
		u := Int(8).Unsigned().WithUnsignedRange(10, 20)
		min, max := u.UnsignedRange()
		if min != 10 || max != 20 {
			t.Fatalf("expected [10,20], got [%d,%d]", min, max)
		}
	})
	t.Run("uint64 same range", func(t *testing.T) {
		u := Int(64).Unsigned()
		u2 := u.WithUnsignedRange(0, math.MaxUint64)
		if !Equal(u, u2) {
			t.Fatalf("expected unchanged type")
		}
	})
	t.Run("invalid min", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Int(8).Unsigned().WithUnsignedRange(300, 301)
	})
	t.Run("max less than min", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Int(8).Unsigned().WithUnsignedRange(5, 4)
	})
	t.Run("max greater than max", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Int(8).Unsigned().WithUnsignedRange(0, math.MaxUint16)
	})
	t.Run("wrong kind", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Int(8).WithUnsignedRange(0, 1)
	})
}

// sameType reports whether t1 and t2 are the same type. It compares t2 against
// t1.
func sameType(t1, t2 Type) error {
	if t1.kind == t2.kind &&
		t1.size == t2.size &&
		t1.generic == t2.generic &&
		t1.unsigned == t2.unsigned &&
		t1.unique == t2.unique &&
		t1.real == t2.real &&
		t1.p == t2.p &&
		t1.s == t2.s &&
		t1.vl == nil && t2.vl == nil {
		return nil
	}
	// Kind.
	if t1.kind != t2.kind {
		if !t2.kind.Valid() {
			return fmt.Errorf("unknown kind %d", t2.kind)
		}
		return fmt.Errorf("expected kind %d, got %d", t1.kind, t2.kind)
	}
	// Minimum and maximum.
	switch {
	case t1.kind == IntKind && t1.unsigned && t1.size < 4: // 8, 16, 24, and 32 bits
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", uint32(t1.p), uint32(t1.s), uint32(t2.p), uint32(t2.s))
		}
	case t1.kind == IntKind && !t1.unsigned && t1.size < 4: // 8, 16, 24, and 32 bits
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", t1.p, t1.s, t2.p, t2.s)
		}
	case t1.kind == IntKind || t1.kind == FloatKind:
		if t1.vl != t2.vl {
			if t1.vl == nil {
				return fmt.Errorf("expected no range, got %v", t2.vl)
			} else if t2.vl == nil {
				return fmt.Errorf("expected range %v, got no range", t1.vl)
			} else {
				return fmt.Errorf("expected range %v, got %v", t1.vl, t2.vl)
			}
		}
	case t1.kind == DecimalKind:
		if vl1, ok := t1.vl.(decimalRange); ok {
			vl2, ok := t2.vl.(decimalRange)
			if !ok || !vl1.min.Equal(vl2.min) || !vl1.max.Equal(vl2.max) {
				return fmt.Errorf("expected range %v, got range %v", t1.vl, t2.vl)
			}
		} else if t2.vl != nil {
			return fmt.Errorf("expected no-range, got range %v", t2.vl)
		}
	}
	// Generic.
	if t1.generic != t2.generic {
		return fmt.Errorf("expected generic %t, got %t", t1.generic, t2.generic)
	}
	// Real.
	if t1.real != t2.real {
		return fmt.Errorf("expected real %t, got %t", t1.real, t2.real)
	}
	// Precision, byte length or elements minimum length.
	if t1.p != t2.p {
		switch t1.kind {
		case StringKind:
			return fmt.Errorf("expected byte length %d, got %d", t1.p, t2.p)
		case DecimalKind:
			return fmt.Errorf("expected precision %d, got %d", t1.p, t2.p)
		case ArrayKind:
			return fmt.Errorf("expected elements minimum length %d, got %d", t1.p, t2.p)
		}
		return fmt.Errorf("expected p == 0, got %d", t2.p)
	}
	// Scale, character length or elements maximum length.
	if t1.s != t2.s {
		switch t1.kind {
		case StringKind:
			return fmt.Errorf("expected character length %d, got %d", t1.s, t2.s)
		case DecimalKind:
			return fmt.Errorf("expected scale %d, got %d", t1.s, t2.s)
		case ArrayKind:
			return fmt.Errorf("expected elements maximum length %d, got %d", t1.s, t2.s)
		}
		return fmt.Errorf("expected s == 0, got %d", t2.s)
	}
	// Regular expression or values.
	if t1.kind == StringKind {
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
	// Unique elements and element type.
	if t1.kind == ArrayKind {
		if t1.unique != t2.unique {
			if t1.unique {
				return errors.New("expected unique elements, got non-unique")
			}
			return errors.New("expected non-unique elements, got unique")
		}
		if t2.vl == nil {
			return errors.New("expected element type, got nil")
		}
		if err := sameType(t1.vl.(Type), t2.vl.(Type)); err != nil {
			return err
		}
	}
	// Properties.
	if t1.kind == ObjectKind {
		if t2.vl == nil {
			return errors.New("expected properties, got nil")
		}
		pn2, ok := t2.vl.(Properties)
		if !ok {
			return fmt.Errorf("expected properties, got a %T value", t2.vl)
		}
		if pn2.properties == nil {
			return fmt.Errorf("unexpected nil properties")
		}
		if pn2.names == nil {
			return fmt.Errorf("unexpected nil names")
		}
		pn1 := t1.vl.(Properties)
		if len(pn1.properties) != len(pn2.properties) {
			return fmt.Errorf("expected %d properties, got %d", len(pn1.properties), len(pn2.properties))
		}
		if len(pn1.names) != len(pn2.names) {
			return fmt.Errorf("expected %d names, got %d", len(pn1.names), len(pn2.names))
		}
		for i, p1 := range pn1.properties {
			p2 := pn2.properties[i]
			err := sameProperty(p1, p2)
			if err != nil {
				return err
			}
		}
		for n1, i1 := range pn1.names {
			i2, ok := pn2.names[n1]
			if !ok {
				return fmt.Errorf("expected name %q, got no name", n1)
			}
			if i1 != i2 {
				return fmt.Errorf("expected index %d, got %d", i1, i2)
			}
		}
	}
	// Value type.
	if t1.kind == MapKind {
		if t2.vl == nil {
			return errors.New("expected value type, got nil")
		}
		if err := sameType(t1.vl.(Type), t2.vl.(Type)); err != nil {
			return err
		}
	}
	// Parameter name.
	if t1.kind == InvalidKind {
		if t1.vl != t2.vl {
			if name, ok := t1.vl.(string); ok {
				return fmt.Errorf("expected parameter name %q, got %#v", name, t2.vl)
			}
			return fmt.Errorf("expected no parameter name, got %#v", t2.vl)
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
	if p1.Prefilled != p2.Prefilled {
		return fmt.Errorf("expected property prefilled %q, got %q", p1.Prefilled, p2.Prefilled)
	}
	if err := sameType(p1.Type, p2.Type); err != nil {
		return err
	}
	if p1.CreateRequired != p2.CreateRequired {
		return fmt.Errorf("expected property key 'create required' with value %t, got %t", p1.CreateRequired, p2.CreateRequired)
	}
	if p1.UpdateRequired != p2.UpdateRequired {
		return fmt.Errorf("expected property key 'update required' with value %t, got %t", p1.UpdateRequired, p2.UpdateRequired)
	}
	if p1.ReadOptional != p2.ReadOptional {
		return fmt.Errorf("expected property key 'read optional' with value %t, got %t", p1.ReadOptional, p2.ReadOptional)
	}
	if p1.Nullable != p2.Nullable {
		return fmt.Errorf("expected property key 'nullable' with value %t, got %t", p1.Nullable, p2.Nullable)
	}
	if p1.Description != p2.Description {
		return fmt.Errorf("expected property description %q, got %q", p1.Description, p2.Description)
	}
	return nil
}
