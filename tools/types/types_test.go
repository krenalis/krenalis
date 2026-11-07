// Copyright 2026 Open2b. All rights reserved.
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

func Test_Generic(t *testing.T) {

	tests := []struct {
		name    string
		typ     Type
		generic bool
	}{
		{"parameter", Parameter("T"), true},
		{"string", String(), false},
		{"array of string", Array(String()), false},
		{"array of parameter", Array(Parameter("T")), true},
		{"map of string", Map(String()), false},
		{"map of parameter", Map(Parameter("T")), true},
		{"array of map of parameter", Array(Map(Parameter("T"))), true},
		{"map of array of parameter", Map(Array(Parameter("T"))), true},
		{"object with string property", Object([]Property{{Name: "a", Type: String()}}), false},
		{"object with parameter property", Object([]Property{{Name: "a", Type: Parameter("T")}}), true},
		{"object with array of parameter property", Object([]Property{{Name: "a", Type: Array(Parameter("T"))}}), true},
		{"object with map of parameter property", Object([]Property{{Name: "a", Type: Map(Parameter("T"))}}), true},
		{"array of non-generic object", Array(Object([]Property{{Name: "a", Type: String()}})), false},
		{"array of generic object", Array(Object([]Property{{Name: "a", Type: Parameter("T")}})), true},
		{"map of generic object", Map(Object([]Property{{Name: "a", Type: Parameter("T")}})), true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.typ.Generic(); got != test.generic {
				t.Fatalf("expected generic %t, got %t", test.generic, got)
			}
		})
	}

}

// Test_IsFloat checks ordinary and real float recognition with optional bit
// sizes, false results for other types and panics for invalid arguments.
func Test_IsFloat(t *testing.T) {

	for _, size := range []int{32, 64} {
		for _, real := range []bool{false, true} {
			typ := Float(size)
			if real {
				typ = typ.Real()
			}
			t.Run(fmt.Sprintf("size=%d/real=%t", size, real), func(t *testing.T) {
				if got := typ.IsFloat(); !got {
					t.Errorf("IsFloat(): expected true, got %t", got)
				}
				for _, requested := range []int{32, 64} {
					expected := requested == size
					if got := typ.IsFloat(requested); got != expected {
						t.Errorf("IsFloat(%d): expected %t, got %t", requested, expected, got)
					}
				}
			})
		}
	}

	for _, typ := range []Type{Int(32), Int(64), String(), {}} {
		t.Run(fmt.Sprintf("other/type=%s", typ), func(t *testing.T) {
			if got := typ.IsFloat(); got {
				t.Errorf("IsFloat(): expected false, got %t", got)
			}
			for _, size := range []int{32, 64} {
				if got := typ.IsFloat(size); got {
					t.Errorf("IsFloat(%d): expected false, got %t", size, got)
				}
			}
		})
	}

	for _, typ := range []Type{Float(64), String(), {}} {
		for _, sizes := range [][]int{{-1}, {0}, {8}, {16}, {24}, {63}, {128}, {32, 64}} {
			t.Run(fmt.Sprintf("panic/type=%s/sizes=%v", typ, sizes), func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Fatal("expected panic, got none")
					}
				}()
				typ.IsFloat(sizes...)
			})
		}
	}

}

// Test_IsInt checks signed and unsigned int recognition with optional bit
// sizes, false results for other types and panics for invalid arguments.
func Test_IsInt(t *testing.T) {

	for _, size := range []int{8, 16, 24, 32, 64} {
		for _, unsigned := range []bool{false, true} {
			typ := Int(size)
			if unsigned {
				typ = typ.Unsigned()
			}
			t.Run(fmt.Sprintf("size=%d/unsigned=%t", size, unsigned), func(t *testing.T) {
				if got := typ.IsInt(); !got {
					t.Errorf("IsInt(): expected true, got %t", got)
				}
				for _, requested := range []int{8, 16, 24, 32, 64} {
					expected := requested == size
					if got := typ.IsInt(requested); got != expected {
						t.Errorf("IsInt(%d): expected %t, got %t", requested, expected, got)
					}
				}
			})
		}
	}

	for _, typ := range []Type{Float(32), Float(64), String(), {}} {
		t.Run(fmt.Sprintf("other/type=%s", typ), func(t *testing.T) {
			if got := typ.IsInt(); got {
				t.Errorf("IsInt(): expected false, got %t", got)
			}
			for _, size := range []int{8, 16, 24, 32, 64} {
				if got := typ.IsInt(size); got {
					t.Errorf("IsInt(%d): expected false, got %t", size, got)
				}
			}
		})
	}

	for _, typ := range []Type{Int(64), String(), {}} {
		for _, sizes := range [][]int{{-1}, {0}, {9}, {63}, {128}, {32, 64}} {
			t.Run(fmt.Sprintf("panic/type=%s/sizes=%v", typ, sizes), func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Fatal("expected panic, got none")
					}
				}()
				typ.IsInt(sizes...)
			})
		}
	}

}

// Test_IsReal checks recognition of floats marked with Real and false results
// for ordinary floats, ints, strings and the zero Type.
func Test_IsReal(t *testing.T) {

	tests := []struct {
		name     string
		typ      Type
		expected bool
	}{
		{"float32", Float(32), false},
		{"float64", Float(64), false},
		{"real32", Float(32).Real(), true},
		{"real64", Float(64).Real(), true},
		{"int", Int(32), false},
		{"string", String(), false},
		{"invalid", Type{}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.typ.IsReal(); got != test.expected {
				t.Fatalf("IsReal(): expected %t, got %t", test.expected, got)
			}
		})
	}

}

// Test_IsUnsigned checks that IsUnsigned returns true for unsigned integers
// and false for signed integers and types other than integers.
func Test_IsUnsigned(t *testing.T) {

	tests := []struct {
		name     string
		typ      Type
		expected bool
	}{
		{"signed", Int(32), false},
		{"unsigned", Int(32).Unsigned(), true},
		{"float", Float(32), false},
		{"string", String(), false},
		{"invalid", Type{}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.typ.IsUnsigned(); got != test.expected {
				t.Fatalf("IsUnsigned(): expected %t, got %t", test.expected, got)
			}
		})
	}

}

func Test_Ranges(t *testing.T) {

	for _, s := range []int{8, 16, 24, 32, 64} {
		t.Run(fmt.Sprintf("int/%d", s), func(t *testing.T) {

			unsignedMax := uint64(math.MaxUint64) >> (64 - s)
			signedMax := int64(unsignedMax >> 1)
			signedMin := -signedMax - 1
			if min, max := Int(s).IntRange(); min != signedMin || max != signedMax {
				t.Errorf("expected signed range [%d,%d], got [%d,%d]", signedMin, signedMax, min, max)
			}
			if min, max := Int(s).Unsigned().UnsignedRange(); min != 0 || max != unsignedMax {
				t.Errorf("expected unsigned range [0,%d], got [%d,%d]", unsignedMax, min, max)
			}

			signedYear := Int(s).AsYear()
			signedYearMax := min(signedMax, MaxYear)
			if lo, hi := signedYear.IntRange(); lo != MinYear || hi != signedYearMax {
				t.Errorf("expected signed year range [%d,%d], got [%d,%d]", MinYear, signedYearMax, lo, hi)
			}
			unsignedYear := Int(s).Unsigned().AsYear()
			unsignedYearMax := min(unsignedMax, MaxYear)
			if lo, hi := unsignedYear.UnsignedRange(); lo != MinYear || hi != unsignedYearMax {
				t.Errorf("expected unsigned year range [%d,%d], got [%d,%d]", MinYear, unsignedYearMax, lo, hi)
			}
			if got := signedYear.Unsigned(); !Equal(unsignedYear, got) {
				t.Fatalf("expected unsigned year type %#v, got %#v", unsignedYear, got)
			}
			if got := signedYear.WithIntRange(2, 100).Unsigned(); !Equal(unsignedYear, got) {
				t.Fatalf("expected default unsigned year type %#v, got %#v", unsignedYear, got)
			}

			for _, year := range []struct {
				name string
				typ  Type
				max  int64
			}{
				{"signed", signedYear, signedYearMax},
				{"unsigned", unsignedYear, int64(unsignedYearMax)},
			} {
				lo, hi := int64(MinYear), year.max
				tests := []struct {
					name     string
					min, max int64
				}{
					{"defaults", lo, hi},
					{"minimum", 2, hi},
					{"maximum", lo, 100},
					{"both bounds", 2, 100},
					{"single value", 2, 2},
				}
				for _, test := range tests {
					t.Run(fmt.Sprintf("year/%s/%s", year.name, test.name), func(t *testing.T) {

						var yearFirst, rangeFirst, restored Type
						var gotMin, gotMax int64
						if year.typ.IsUnsigned() {
							min, max := uint64(test.min), uint64(test.max)
							yearFirst = year.typ.WithUnsignedRange(min, max)
							rangeFirst = Int(s).Unsigned().WithUnsignedRange(min, max).AsYear()
							restored = yearFirst.WithUnsignedRange(uint64(lo), uint64(hi))
							min, max = yearFirst.UnsignedRange()
							gotMin, gotMax = int64(min), int64(max)
						} else {
							yearFirst = year.typ.WithIntRange(test.min, test.max)
							rangeFirst = Int(s).WithIntRange(test.min, test.max).AsYear()
							restored = yearFirst.WithIntRange(lo, hi)
							gotMin, gotMax = yearFirst.IntRange()
						}

						if gotMin != test.min || gotMax != test.max {
							t.Fatalf("expected range [%d,%d], got [%d,%d]", test.min, test.max, gotMin, gotMax)
						}
						if !Equal(yearFirst, rangeFirst) {
							t.Fatalf("expected equal construction results, got %#v and %#v", yearFirst, rangeFirst)
						}
						if test.min == lo && test.max == hi && !Equal(year.typ, yearFirst) {
							t.Fatalf("expected default year type %#v, got %#v", year.typ, yearFirst)
						}
						if !Equal(year.typ, restored) {
							t.Fatalf("expected restored year type %#v, got %#v", year.typ, restored)
						}

					})
				}
			}

		})
	}

	for _, s := range []int{32, 64} {
		if min, max := Float(s).FloatRange(); min != math.Inf(-1) || max != math.Inf(1) {
			t.Errorf("Float(%d).FloatRange(): expected (-Inf, +Inf), got (%g, %g)", s, min, max)
		}
	}

	if min, max := Float(32).Real().FloatRange(); min != -math.MaxFloat32 || max != math.MaxFloat32 {
		t.Errorf("Float(32).FloatRange(): expected (%f, %f), got (%f, %f)", -math.MaxFloat32, math.MaxFloat32, min, max)
	}
	if min, max := Float(64).Real().FloatRange(); min != -math.MaxFloat64 || max != math.MaxFloat64 {
		t.Errorf("Float(64).FloatRange(): expected (%f, %f), got (%f, %f)", -math.MaxFloat64, math.MaxFloat64, min, max)
	}

	for _, size := range []int{32, 64} {
		limit := math.MaxFloat64
		tiny := -math.SmallestNonzeroFloat64
		largeMin := math.MaxFloat64 / 2
		if size == 32 {
			limit = math.MaxFloat32
			tiny = 0
			largeMin = limit
		}
		tests := []struct {
			name             string
			min, max         float64
			wantMin, wantMax float64
		}{
			{"finite", -1, 1, -1, 1},
			{"single value", 0, 0, 0, 0},
			{"negative zero minimum", math.Copysign(0, -1), 1, 0, 1},
			{"negative zero maximum", -1, math.Copysign(0, -1), -1, 0},
			{"tiny negative minimum", -math.SmallestNonzeroFloat64, 1, tiny, 1},
			{"tiny negative maximum", -1, -math.SmallestNonzeroFloat64, -1, tiny},
			{"minimum", 0, math.Inf(1), 0, limit},
			{"maximum", math.Inf(-1), 0, -limit, 0},
			{"unbounded", math.Inf(-1), math.Inf(1), -limit, limit},
			{"finite upper default", 0, limit, 0, limit},
			{"finite lower default", -limit, 0, -limit, 0},
			{"finite defaults", -limit, limit, -limit, limit},
			{"large finite minimum", -math.MaxFloat64, 0, -limit, 0},
			{"large finite maximum", 0, math.MaxFloat64, 0, limit},
			{"large finite bounds", -math.MaxFloat64, math.MaxFloat64, -limit, limit},
			{"large positive range", math.MaxFloat64 / 2, math.MaxFloat64, largeMin, limit},
			{"large negative range", -math.MaxFloat64, -math.MaxFloat64 / 2, -limit, -largeMin},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("float/%d/%s", size, test.name), func(t *testing.T) {
				bounded := Float(size).WithFloatRange(test.min, test.max)
				realFirst := Float(size).Real().WithFloatRange(test.min, test.max)
				rangeFirst := bounded.Real()
				if !Equal(realFirst, rangeFirst) {
					t.Fatalf("expected equal construction results, got %#v and %#v", realFirst, rangeFirst)
				}
				if min, max := realFirst.FloatRange(); min != test.wantMin || max != test.wantMax {
					t.Fatalf("expected range [%g, %g], got [%g, %g]", test.wantMin, test.wantMax, min, max)
				}
				wantMin, wantMax := test.wantMin, test.wantMax
				if math.IsInf(test.min, -1) {
					wantMin = test.min
				}
				if math.IsInf(test.max, 1) {
					wantMax = test.max
				}
				if min, max := bounded.FloatRange(); min != wantMin || max != wantMax {
					t.Fatalf("expected range [%g, %g], got [%g, %g]", wantMin, wantMax, min, max)
				}
				canonical := Float(size).Real().WithFloatRange(test.wantMin, test.wantMax)
				if !Equal(realFirst, canonical) {
					t.Fatalf("expected canonical type %#v, got %#v", canonical, realFirst)
				}
				for _, typ := range []Type{bounded, realFirst} {
					min, max := typ.FloatRange()
					if min == 0 && math.Signbit(min) || max == 0 && math.Signbit(max) {
						t.Fatalf("expected positive zero bounds, got [%g, %g]", min, max)
					}
				}
				if test.wantMin == -limit && test.wantMax == limit && !Equal(Float(size).Real(), realFirst) {
					t.Fatal("expected the default real type, got redundant bounds")
				}
				replaced := Float(size).Real().WithFloatRange(-0.5, 0.5).WithFloatRange(test.min, test.max)
				if !Equal(realFirst, replaced) {
					t.Fatal("expected replacement of previous real bounds, got a different type")
				}
				replaced = Float(size).WithFloatRange(-0.5, 0.5).WithFloatRange(test.min, test.max)
				if !Equal(bounded, replaced) {
					t.Fatal("expected replacement of previous float bounds, got a different type")
				}
			})
		}
	}

	tests := []struct {
		name     string
		typ      Type
		min, max float64
		want     string
	}{
		{"wrong kind", Int(32), 0, 1, "type is not a float type"},
		{"NaN minimum", Float(64), math.NaN(), 1, "min and max cannot be NaN"},
		{"NaN maximum", Float(64), 0, math.NaN(), "min and max cannot be NaN"},
		{"reversed", Float(64), 1, 0, "max cannot be less than min"},
		{"reversed before rounding", Float(32), 1.00000001, 1, "max cannot be less than min"},
		{"reversed positive saturation", Float(32), math.MaxFloat64, math.MaxFloat64 / 2, "max cannot be less than min"},
		{"reversed negative saturation", Float(32), -math.MaxFloat64 / 2, -math.MaxFloat64, "max cannot be less than min"},
		{"positive infinity minimum", Float(64), math.Inf(1), math.Inf(1), "min cannot be +Inf"},
		{"negative infinity maximum", Float(64).Real(), math.Inf(-1), math.Inf(-1), "max cannot be -Inf"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil {
					t.Fatal("expected a panic, got none")
				}
				if message := fmt.Sprint(got); message != test.want {
					t.Fatalf("expected panic %q, got %q", test.want, message)
				}
			}()
			test.typ.WithFloatRange(test.min, test.max)
		})
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

// Test_String checks the string representation of types, including the
// unsigned flag of integers.
func Test_String(t *testing.T) {

	tests := []struct {
		typ      Type
		expected string
	}{
		{String(), "string"},
		{Boolean(), "boolean"},
		{Int(8), "int(8)"},
		{Int(32).Unsigned(), "unsigned int(32)"},
		{Int(64).Unsigned().WithUnsignedRange(1, 10), "unsigned int(64)"},
		{Float(32), "float(32)"},
		{Decimal(10, 2), "decimal(10,2)"},
		{Array(Int(16).Unsigned()), "array(unsigned int(16))"},
		{Map(Int(24)), "map(int(24))"},
		{Parameter("T"), "T"},
		{Type{}, ""},
	}

	for _, test := range tests {
		if got := test.typ.String(); got != test.expected {
			t.Errorf("String(): expected %q, got %q", test.expected, got)
		}
	}

}

// Test_StringConstraintPanics checks that combining incompatible string
// constraints or passing a nil pattern causes a panic.
func Test_StringConstraintPanics(t *testing.T) {

	pattern := regexp.MustCompile(`a`)
	tests := []struct {
		name  string
		apply func()
	}{
		{"pattern then max bytes", func() { String().WithPattern(pattern).WithMaxBytes(1) }},
		{"pattern then max length", func() { String().WithPattern(pattern).WithMaxLength(1) }},
		{"max bytes then pattern", func() { String().WithMaxBytes(1).WithPattern(pattern) }},
		{"max length then pattern", func() { String().WithMaxLength(1).WithPattern(pattern) }},
		{"nil pattern", func() { String().WithPattern(nil) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic, got none")
				}
			}()

			test.apply()
		})
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

	// Test invalid display name encoding.
	_, err = ObjectOf([]Property{{Name: "firstName", Type: String(), DisplayName: string([]byte{0xff})}})
	if err != nil {
		if err.Error() != "invalid UTF-8 encoding" {
			t.Errorf("expected invalid UTF-8 encoding error, got %v", err)
		}
	} else {
		t.Error("expected invalid UTF-8 encoding error, got nil")
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
		t1.semantics == t2.semantics &&
		t1.semanticsOption == t2.semanticsOption &&
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
	// Semantics.
	if t1.semantics != t2.semantics {
		return fmt.Errorf("expected semantics %s, got %s", t1.semantics, t2.semantics)
	}
	// Option associated with the semantics.
	if t1.semanticsOption != t2.semanticsOption {
		return fmt.Errorf("expected semantics option %v, got %v", t1.semanticsOption, t2.semanticsOption)
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
	if p1.DisplayName != p2.DisplayName {
		return fmt.Errorf("expected property display name %q, got %q", p1.DisplayName, p2.DisplayName)
	}
	if p1.Description != p2.Description {
		return fmt.Errorf("expected property description %q, got %q", p1.Description, p2.Description)
	}
	return nil
}
