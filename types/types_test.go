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
)

func Test_BitSize(t *testing.T) {

	for _, s := range []int{8, 16, 24, 32, 64} {
		if got := Int(s).BitSize(); s != got {
			t.Errorf("Int(%d).BitSize(): expected %d, got %d", s, s, got)
		}
		if got := Uint(s).BitSize(); s != got {
			t.Errorf("Uint(%d).BitSize(): expected %d, got %d", s, s, got)
		}
	}

	for _, s := range []int{32, 64} {
		if got := Float(s).BitSize(); s != got {
			t.Errorf("Float(%d).BitSize(): expected %d, got %d", s, s, got)
		}
	}

}

func Test_Ranges(t *testing.T) {

	for _, s := range []int{8, 16, 24, 32, 64} {
		if min, max := Int(s).IntRange(); min != -1<<(s-1) || max != 1<<(s-1)-1 {
			t.Errorf("Int(%d).IntRange(): expected (%d, %d), got (%d, %d)", s, -1<<(s-1), 1<<(s-1)-1, min, max)
		}
		if min, max := Uint(s).UintRange(); min != 0 || max != 1<<s-1 {
			t.Errorf("Int(%d).IntRange(): expected (%d, %d), got (%d, %d)", s, 0, 1<<s-1, min, max)
		}
	}

	for _, s := range []int{32, 64} {
		if min, max := Float(s).FloatRange(); min != math.Inf(-1) || max != math.Inf(1) {
			t.Errorf("Float(32).FloatRange(): expected (-Inf, +Inf), got (%f, %f)", min, max)
		}
	}

	if min, max := Float(32).AsReal().FloatRange(); min != -math.MaxFloat32 || max != math.MaxFloat32 {
		t.Errorf("Float(32).FloatRange(): expected (%f, %f), got (%f, %f)", -math.MaxFloat32, math.MaxFloat32, min, max)
	}
	if min, max := Float(64).AsReal().FloatRange(); min != -math.MaxFloat64 || max != math.MaxFloat64 {
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

func Test_ObjectOf_Errors(t *testing.T) {

	// Test InvalidPropertyNameError.
	_, err := ObjectOf([]Property{
		{Name: "firstName", Type: Text()},
		{Name: "last name", Type: Text()},
		{Name: "phone", Type: Text()},
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
		{Name: "firstName", Type: Text()},
		{Name: "lastName", Type: Text()},
		{Name: "firstName", Type: Text()},
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

func Test_Properties(t *testing.T) {
	properties := []Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Boolean()},
	}
	i := 0
	for k, p := range Object(properties).Properties() {
		if k != i {
			t.Fatalf("expected i=%d, got i=%d", i, k)
		}
		if err := sameProperty(p, properties[i]); err != nil {
			t.Fatal(err)
		}
		i++
	}
}

// sameType reports whether t1 and t2 are the same type. It compares t2 against
// t1.
func sameType(t1, t2 Type) error {
	if t1.kind == t2.kind &&
		t1.size == t2.size &&
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
	case t1.kind == IntKind && t1.size < 4: // 8, 16, 24, and 32 bits
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", t1.p, t1.s, t2.p, t2.s)
		}
	case t1.kind == UintKind && t1.size < 4: // 8, 16, 24, and 32 bits
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", uint32(t1.p), uint32(t1.s), uint32(t2.p), uint32(t2.s))
		}
	case t1.kind == IntKind || t1.kind == UintKind || t1.kind == FloatKind:
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
	// Real.
	if t1.real != t2.real {
		return fmt.Errorf("expected real %t, got %t", t1.real, t2.real)
	}
	// Precision, byte length or items minimum length.
	if t1.p != t2.p {
		switch t1.kind {
		case DecimalKind:
			return fmt.Errorf("expected precision %d, got %d", t1.p, t2.p)
		case TextKind:
			return fmt.Errorf("expected byte length %d, got %d", t1.p, t2.p)
		case ArrayKind:
			return fmt.Errorf("expected items minimum length %d, got %d", t1.p, t2.p)
		}
		return fmt.Errorf("expected p == 0, got %d", t2.p)
	}
	// Scale, character length or items maximum length.
	if t1.s != t2.s {
		switch t1.kind {
		case DecimalKind:
			return fmt.Errorf("expected scale %d, got %d", t1.s, t2.s)
		case TextKind:
			return fmt.Errorf("expected character length %d, got %d", t1.s, t2.s)
		case ArrayKind:
			return fmt.Errorf("expected items maximum length %d, got %d", t1.s, t2.s)
		}
		return fmt.Errorf("expected s == 0, got %d", t2.s)
	}
	// Regular expression or values.
	if t1.kind == TextKind {
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
	if t1.kind == ArrayKind {
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
	if t1.kind == ObjectKind {
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
	if t1.kind == MapKind {
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
	if p1.Placeholder != p2.Placeholder {
		return fmt.Errorf("expected property placeholder %q, got %q", p1.Placeholder, p2.Placeholder)
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
	if p1.Note != p2.Note {
		return fmt.Errorf("expected property note %q, got %q", p1.Note, p2.Note)
	}
	return nil
}
