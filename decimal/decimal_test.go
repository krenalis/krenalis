//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package decimal

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ericlagergren/decimal"
)

func Test_Decimal_Append(t *testing.T) {

	tests := []struct {
		x        string
		expected string
	}{
		{"0", "0: 0"},
		{"5", "1: 5"},
		{"-23.7910", "2: -23.791"},
		{"0.0001", "3: 0.0001"},
		{"2.6e4", "4: 2.6e4"},
	}

	for i, test := range tests {
		buf := []byte(strconv.Itoa(i) + ": ")
		got := MustParse(test.x).Append(buf)
		if test.expected != string(got) {
			t.Fatalf("%q.Append(buf): expected %q, got %q", test.x, test.expected, got)
		}
	}

}

func Test_Decimal_Cmp(t *testing.T) {

	tests := []struct {
		x, y     string
		expected int
	}{
		{"0", "0", 0},
		{"6093371.0155", "6093371.0155", 0},
		{"0", "1", -1},
		{"1", "0", 1},
		{"5.6", "4.7", 1},
		{"123", "125", -1},
		{"806", "299", 1},
		{"123.7509", "125.7509", -1},
		{"779", "779.9", -1},
		{"147.01", "147", 1},
		{"12.0372", "12.0371", 1},
		{"28045.674", "28045.6745", -1},
		{"1.001", "1.0001", 1},
		{"-1", "-1", 0},
		{"-304", "-305", 1},
		{"-912", "-634", -1},
		{"-1", "1", -1},
		{"1", "-1", 1},
		{"-830.26", "-831", 1},
		{"-118494.08231", "-118494.08230", -1},
	}

	for _, test := range tests {
		got := MustParse(test.x).Cmp(MustParse(test.y))
		if test.expected != got {
			t.Fatalf("Cwd(%q, %q): expected %d, got %d", test.x, test.y, test.expected, got)
		}
	}

}

func Test_Decimal_Int64(t *testing.T) {

	tests := []struct {
		x        string
		expected int64
		err      error
	}{
		{"0", 0, nil},
		{"813", 813, nil},
		{"-79406124", -79406124, nil},
		{"9223372036854775807", math.MaxInt64, nil},  // MaxInt64
		{"-9223372036854775808", math.MinInt64, nil}, // MinInt64
		{"9223372036854775808", 0, ErrOutOfRange},    // MaxInt64+1
		{"-9223372036854775809", 0, ErrOutOfRange},   // MinInt64-1
		{"8305916205729431730562153", 0, ErrOutOfRange},
		{"-8305916205729431730562153", 0, ErrOutOfRange},
		{"1.1", 0, ErrOutOfRange},
		{"-0.1", 0, ErrOutOfRange},
	}

	for _, test := range tests {
		got, err := MustParse(test.x).Int64()
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("%q.Int64(): expected error '%#v', got error '%#v'", test.x, test.err, err)
		}
		if test.expected != got {
			t.Fatalf("%q.Int64(): expected %d, got %d", test.x, test.expected, got)
		}
	}

}

func Test_Decimal_MarshalJSON(t *testing.T) {

	tests := []struct {
		x        string
		expected string
	}{
		{"0", "0"},
		{"5", "5"},
		{"1.738", "1.738"},
		{"2068825619", "2068825619"},
		{"-0.0001", "-0.0001"},
		{"0.5e999", "5E+998"},
		{"1.3e-999", "1.3E-999"},
	}

	for _, test := range tests {
		got, err := MustParse(test.x).MarshalJSON()
		if err != nil {
			t.Fatalf("%q.MarshalJSON(): expected %q, got error %v", test.x, test.expected, err)
		}
		if test.expected != string(got) {
			t.Fatalf("%q.MarshalJSON(): expected %q, got %q", test.x, test.expected, string(got))
		}
	}

}

func Test_Decimal_Neg(t *testing.T) {

	tests := []struct {
		x        string
		expected string
	}{
		{"0", "0"},
		{"5", "-5"},
		{"-3", "3"},
		{"0.007", "-0.007"},
		{"4309.0012159", "-4309.0012159"},
		{"-0.5e999", "0.5e999"},
		{"1.23e-5", "-1.23e-5"},
		{"9223372036854775807", "-9223372036854775807"},
		{"-9223372036854775808", "9223372036854775808"},
	}

	for _, test := range tests {
		got := MustParse(test.x).Neg()
		if !MustParse(test.expected).Equal(got) {
			t.Fatalf("%q.Neg(): expected %q, got %q", test.x, test.expected, got)
		}
	}

}

func Test_Decimal_Uint64(t *testing.T) {

	tests := []struct {
		x        string
		expected uint64
		err      error
	}{
		{"0", 0, nil},
		{"813", 813, nil},
		{"79406124", 79406124, nil},
		{"18446744073709551615", math.MaxUint64, nil}, // MaxUint64
		{"18446744073709551616", 0, ErrOutOfRange},    // MaxInt64+1
		{"8305916205729431730562153", 0, ErrOutOfRange},
		{"-1", 0, ErrOutOfRange},
		{"1.1", 0, ErrOutOfRange},
		{"0.1", 0, ErrOutOfRange},
	}

	for _, test := range tests {
		got, err := MustParse(test.x).Uint64()
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("%q.Int64(): expected error '%#v', got error '%#v'", test.x, test.err, err)
		}
		if test.expected != got {
			t.Fatalf("%q.Int64(): expected %d, got %d", test.x, test.expected, got)
		}
	}

}

func Test_Decimal_Value(t *testing.T) {

	tests := []struct {
		x        string
		expected driver.Value
	}{
		{"0", int64(0)},
		{"10558279", int64(10558279)},
		{"-321", int64(-321)},
		{"9223372036854775807", int64(math.MaxInt64)},    // MaxInt64
		{"-9223372036854775808", int64(math.MinInt64)},   // MinInt64
		{"9223372036854775808", "9223372036854775808"},   // MaxInt64+1
		{"-9223372036854775809", "-9223372036854775809"}, // MinInt64-1
		{"-1", int64(-1)},
		{"1.1", "1.1"},
		{"-0.1", "-0.1"},
		{"793.048106", "793.048106"},
		{"2e5", int64(200000)},
	}

	for _, test := range tests {
		got, err := MustParse(test.x).Value()
		if err != nil {
			t.Fatalf("%q.Value(): expected %#v, got error %v", test.x, test.expected, got)
		}
		if test.expected != got {
			t.Fatalf("%q.Value(): expected %#v (type %T), got %#v (type %T)", test.x, test.expected, test.expected, got, got)
		}
	}

}

func Test_Float64(t *testing.T) {

	tests := []struct {
		f         float64
		precision int
		scale     int
		expected  string
		err       error
	}{
		{0, 1, 0, "0", nil},
		{5, 1, 0, "5", nil},
		{23.78, 4, 2, "23.78", nil},
		{23.78, 2, 0, "24", nil},
		{23.49, 4, 0, "23", nil},
		{0.001, 3, 3, "0.001", nil},
		{14627436592.089, 14, 3, "14627436592.089", nil},
		{14627436592.089, 13, 2, "14627436592.09", nil},
		{14627436592.089, 12, 1, "14627436592.1", nil},
		{14627436592.089, 11, 0, "14627436592", nil},
		{-22.951, 7, 5, "-22.951", nil},
		{330.164, 5, 3, "", ErrOutOfRange},
		{1, 1, 2, "", ErrOutOfRange},
		{math.NaN(), 1, 2, "", ErrOutOfRange},
		{math.Inf(-1), 1, 2, "", ErrOutOfRange},
		{math.Inf(1), 1, 2, "", ErrOutOfRange},
	}

	for _, test := range tests {
		got, err := Float64(test.f, test.precision, test.scale)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("expected error '%#v', got error '%#v'", test.err, err)
		}
		if test.expected != "" && !MustParse(test.expected).Equal(got) {
			t.Fatalf("Float64(%f, %d, %d): expected %q, got %q", test.f, test.precision, test.scale, test.expected, got)
		}
	}

}

func Test_Int(t *testing.T) {

	tests := []struct {
		i         int
		precision int
		scale     int
		expected  string
		err       error
	}{
		{0, 1, 0, "0", nil},
		{1, 1, 0, "1", nil},
		{0, 2, 1, "0", nil},
		{-1, 1, 0, "-1", nil},
		{-1, 5, 4, "-1", nil},
		{93506, 5, 0, "93506", nil},
		{3774, 12, 3, "3774", nil},
		{5712890, 10, 4, "0", ErrOutOfRange},
		{-92758264, 12, 4, "-92758264", nil},
		{math.MaxInt64, 22, 4, "0", ErrOutOfRange},
		{math.MaxInt64, 23, 4, "9223372036854775807", nil},
	}

	for _, test := range tests {
		got, err := Int(test.i, test.precision, test.scale)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("expected error '%#v', got error '%#v'", test.err, err)
		}
		if !MustParse(test.expected).Equal(got) {
			t.Fatalf("Int(%d, %d, %d): expected %q, got %q", test.i, test.precision, test.scale, test.expected, got)
		}
	}

}

func Test_New(t *testing.T) {

	tests := []struct {
		i        int64
		scale    int
		expected string
	}{
		{0, 0, "0"},
		{0, 5, "0"},
		{0, -5, "0"},
		{5, 0, "5"},
		{5, 1, "0.5"},
		{5, 7, "0.0000005"},
		{81, -1, "810"},
		{794, -5, "79400000"},
	}

	for _, test := range tests {
		got := New(test.i, test.scale)
		if !MustParse(test.expected).Equal(got) {
			t.Fatalf("New(%d, %d): expected %q, got %q", test.i, test.scale, test.expected, got)
		}
	}

}

func Test_Uint(t *testing.T) {

	tests := []struct {
		i         uint
		precision int
		scale     int
		expected  string
		err       error
	}{
		{0, 1, 0, "0", nil},
		{1, 1, 0, "1", nil},
		{0, 2, 1, "0", nil},
		{189, 4, 1, "189", nil},
		{189, 4, 2, "0", ErrOutOfRange},
		{4021945, 7, 0, "4021945", nil},
		{4021945, 20, 0, "4021945", nil},
		{math.MaxUint64, 24, 4, "18446744073709551615", nil},
		{math.MaxUint64, 30, 0, "18446744073709551615", nil},
		{math.MaxUint64, 21, 2, "0", ErrOutOfRange},
	}

	for _, test := range tests {
		got, err := Uint(test.i, test.precision, test.scale)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("expected error '%#v', got error '%#v'", test.err, err)
		}
		if !MustParse(test.expected).Equal(got) {
			t.Fatalf("Uint(%d, %d, %d): expected %q, got %q", test.i, test.precision, test.scale, test.expected, got)
		}
	}

}

func Test_Parse(t *testing.T) {

	tests := []struct {
		n         string
		precision int
		scale     int
		expected  string
		err       error
	}{
		{"0", 1, 0, "0", nil},
		{"0", 1, 1, "0", nil},
		{"-0", 1, 0, "0", nil},
		{"+0", 1, 0, "0", nil},
		{"0.0", 1, 0, "0", nil},
		{"0.00", 1, 0, "0", nil},
		{"0.00", 1, 1, "0", nil},
		{".0", 1, 0, "0", nil},
		{".0000", 1, 0, "0", nil},
		{"+.0", 1, 0, "0", nil},
		{"+.00", 1, 0, "0", nil},
		{"0e0", 1, 0, "0", nil},
		{"0e12", 1, 0, "0", nil},
		{"-0.1e0", 1, 1, "-0.1", nil},
		{"-.0", 1, 0, "0", nil},
		{"-.00", 1, 0, "0", nil},
		{".5", 1, 1, "0.5", nil},
		{".500", 1, 1, "0.5", nil},
		{".123", 3, 3, "0.123", nil},
		{"6", 5, 4, "6", nil},
		{"+1", 1, 0, "1", nil},
		{"-3", 1, 0, "-3", nil},
		{"1.0", 1, 0, "1", nil},
		{"1.000", 1, 0, "1", nil},
		{"0.1", 1, 1, "0.1", nil},
		{"-0.1", 1, 1, "-0.1", nil},
		{"23.670", 4, 2, "23.67", nil},
		{"-8492.033", 9, 5, "-8492.033", nil},
		{"33510672.20416625806", 19, 11, "33510672.20416625806", nil},
		{"0.17305728433", 12, 11, "0.17305728433", nil},
		{"-0.0000001", 7, 7, "-1e-7", nil},
		{"9210.7037", 0, 0, "9210.7037", nil},
		{"1.5600e2", 3, 0, "156", nil},
		{"1.56e-2", 4, 4, "0.0156", nil},
		{"0.1230e-03", 6, 6, "0.000123", nil},
		{"0.00123e-03", 8, 8, "0.00000123", nil},
		{"0.00123E3", 3, 2, "1.23", nil},
		{"0.00123e6", 6, 0, "1.23e+3", nil},
		{"-0.00123E10", 10, 0, "-1.23e+7", nil},
		{"123456789e100", 0, 0, "1.23456789e+108", nil},
		{"-9223372036854775808", 0, 0, "-9223372036854775808", nil},
		{"1e" + strconv.Itoa(decimal.MaxScale), 0, 0, "1e+999999999999999999", nil},
		{"1e" + strconv.Itoa(decimal.MinScale), 0, 0, "1e-999999999999999999", nil},
		{"9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", 100, 0, "9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", nil},
		{"-9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", 100, 0, "-9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", nil},
		{"9999999999.999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", 100, 90, "9999999999.999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", nil},
		{"999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999.9999999999", 100, 10, "999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999.9999999999", nil},
		{"-61668979943926223154440006", 27, 0, "-61668979943926223154440006", nil},

		{"", 20, 5, "", ErrSyntax},
		{" ", 20, 5, "", ErrSyntax},
		{"1.2 3", 20, 5, "", ErrSyntax},
		{"1_234", 20, 5, "", ErrSyntax},
		{"--2", 20, 5, "", ErrSyntax},
		{" 1", 20, 5, "", ErrSyntax},
		{"1 ", 20, 5, "", ErrSyntax},
		{"1.0 ", 20, 5, "", ErrSyntax},
		{"00.1", 20, 5, "", ErrSyntax},
		{"0.a", 20, 5, "", ErrSyntax},
		{"8..56", 20, 5, "", ErrSyntax},
		{"6.5.7", 20, 5, "", ErrSyntax},

		{"1", 1, 1, "", ErrOutOfRange},
		{"678", 3, 1, "", ErrOutOfRange},
		{"-8492.033", 8, 5, "", ErrOutOfRange},
		{"1.0000001", 7, 7, "", ErrOutOfRange},
		{"123.4", 4, 2, "", ErrOutOfRange},
		{"0.0000001", 7, 6, "", ErrOutOfRange},
		{"1e" + strconv.Itoa(decimal.MaxScale+1), 0, 0, "", ErrOutOfRange},
		{"1e" + strconv.Itoa(decimal.MinScale-1), 0, 0, "", ErrOutOfRange},
	}

	for _, test := range tests {
		got, err := Parse(test.n, test.precision, test.scale)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("Parse(%q): expected error '%#v', got error '%#v'", test.n, test.err, err)
		}
		if test.expected != "" && test.expected != str(got) {
			t.Fatalf("Parse(%q): expected %q, got %q", test.n, test.expected, str(got))
		}
	}

}

func Test_Range(t *testing.T) {

	tests := []struct {
		precision   int
		scale       int
		minExpected string
		maxExpected string
	}{
		{1, 0, "-9", "9"},
		{2, 0, "-99", "99"},
		{12, 0, "-999999999999", "999999999999"},
		{19, 0, "-9999999999999999999", "9999999999999999999"},
		{1, 1, "-0.9", "0.9"},
		{2, 2, "-0.99", "0.99"},
		{8, 8, "-0.99999999", "0.99999999"},
		{19, 19, "-0.9999999999999999999", "0.9999999999999999999"},
		{3, 2, "-9.99", "9.99"},
		{19, 7, "-999999999999.9999999", "999999999999.9999999"},
		{19, 18, "-9.999999999999999999", "9.999999999999999999"},
		{20, 0, "-99999999999999999999", "99999999999999999999"},
		{20, 20, "-0.99999999999999999999", "0.99999999999999999999"},
		{32, 0, "-99999999999999999999999999999999", "99999999999999999999999999999999"},
		{32, 32, "-0.99999999999999999999999999999999", "0.99999999999999999999999999999999"},
		{32, 8, "-999999999999999999999999.99999999", "999999999999999999999999.99999999"},
	}

	for _, test := range tests {
		minGot, maxGot := Range(test.precision, test.scale)
		if !MustParse(test.minExpected).Equal(minGot) {
			t.Fatalf("Range(%d, %d): expected minimum %q, got %q", test.precision, test.scale, test.minExpected, minGot)
		}
		if !MustParse(test.maxExpected).Equal(maxGot) {
			t.Fatalf("Range(%d, %d): expected maximum %q, got %q", test.precision, test.scale, test.maxExpected, maxGot)
		}
	}

}

func Test_Decimal_WriteTo(t *testing.T) {

	tests := []struct {
		n        any
		expected string
		err      error
	}{
		{"0", "0", nil},
		{"-8492.033", "-8492.033", nil},
		{"9210.7037", "9210.7037", nil},
		{"0.00123e-03", "0.00123e-03", nil},
		{"0.00123E3", "0.00123E3", nil},
		{"0.00123e6", "0.00123e6", nil},
		{"1e" + strconv.Itoa(decimal.MaxScale), "1e" + strconv.Itoa(decimal.MaxScale), nil},
		{"1e" + strconv.Itoa(decimal.MinScale), "1e" + strconv.Itoa(decimal.MinScale), nil},
		{"9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", "9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", nil},

		{0, "0", nil},
		{-62910, "-62910", nil},
		{8104722957631, "8104722957631", nil},

		{uint(0), "0", nil},
		{uint(249), "249", nil},
		{uint(93654729690714), "93654729690714", nil},
	}

	var got bytes.Buffer
	for _, test := range tests {
		got.Reset()
		var x Decimal
		switch n := test.n.(type) {
		case string:
			x = MustParse(n)
		case int:
			x = MustInt(n)
		case uint:
			x = MustUint(n)
		}
		n, err := x.WriteTo(&got)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("WriteTo %q: expected error '%#v', got error '%#v'", test.n, test.err, err)
		}
		if test.expected != got.String() {
			t.Fatalf("WriteTo %q: expected %q, got %q", test.n, test.expected, got.String())
		}
		if n != int64(got.Len()) {
			t.Fatalf("WriteTo %q: expected n == %d, got %d", test.n, n, got.Len())
		}
	}

}

func Test_alloc(t *testing.T) {

	var err error
	a := testing.AllocsPerRun(1, func() { _, err = Parse("1.23456e5", 0, 0) })
	if err != nil {
		t.Fatal(err)
	}
	if a != 0 {
		t.Fatalf("Parse: expected 0 allocations, got %.0f", a)
	}

	// TODO(marco): Try to do fewer allocations. It should be possible to make 3 instead of 5.
	a = testing.AllocsPerRun(1, func() { _, err = Parse("999999999999999999999999999999999999999999999999999999999999", 0, 0) })
	if err != nil {
		t.Fatal(err)
	}
	if a != 5 {
		t.Fatalf("Parse: expected 0 allocations, got %.0f", a)
	}

}

// str returns a string representation of x, intended for use in tests.
func str(x Decimal) string {
	s := fmt.Sprintf("%s", x)
	return strings.Replace(s, "E", "e", 1)
}
