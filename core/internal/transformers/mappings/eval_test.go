// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestErrorHelpers checks formatting of conversion error messages.
func TestErrorHelpers(t *testing.T) {
	bErr := errBooleanConversion("and", "x", "foo", types.String())
	if bErr.Error() != "«x» (type string) does not represent a boolean when passed to the «and» function" {
		t.Fatalf("unexpected boolean error: %v", bErr)
	}
	jb := json.Value("true")
	bErr = errBooleanConversion("and", "x", jb, types.JSON())
	if bErr.Error() != "«x», of type JSON true, cannot be passed as boolean to the «and» function" {
		t.Fatalf("unexpected boolean json error: %v", bErr)
	}
	iErr := errInt32Conversion("fn", "x", 5.5, types.Float(64))
	if iErr.Error() != "«x», with a value of %!s(float64=5.5), cannot be passed as a 32-bit int to the «fn» function" {
		t.Fatalf("unexpected int error: %v", iErr)
	}
	ji := json.Value("\"foo\"")
	iErr = errInt32Conversion("fn", "x", ji, types.JSON())
	if iErr.Error() != "«x», of type JSON string, cannot be passed as an int to the «fn» function" {
		t.Fatalf("unexpected int json error: %v", iErr)
	}
	tErr := errStringConversion("up", "x", json.Value("[1]"))
	if tErr.Error() != "«x» (a JSON array) cannot be converted to a string value to be passed to the «up» function" {
		t.Fatalf("unexpected string error: %v", tErr)
	}
}

// Test_appendAsString ensures values are appended correctly as strings.
func Test_appendAsString(t *testing.T) {
	t0 := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		v   any
		typ types.Type
		out string
		err error
	}{
		{nil, types.String(), "start", nil},
		{"foo", types.String(), "startfoo", nil},
		{true, types.Boolean(), "start", errInvalidConversion},
		{int(3), types.Int(32), "start3", nil},
		{uint(4), types.Int(16).Unsigned(), "start4", nil},
		{1.5, types.Float(64), "start1.5", nil},
		{decimal.MustParse("2.7"), types.Decimal(2, 1), "start2.7", nil},
		{t0, types.DateTime(), "start2023-01-02T03:04:05Z", nil},
		{t0, types.Date(), "start2023-01-02", nil},
		{t0, types.Time(), "start03:04:05", nil},
		{json.Value("\"bar\""), types.JSON(), "startbar", nil},
		{json.Value("123"), types.JSON(), "start123", nil},
		{json.Value("true"), types.JSON(), "starttrue", nil},
		{json.Value("null"), types.JSON(), "startnull", nil},
		{json.Value("[1,2]"), types.JSON(), "start", errInvalidConversion},
	}
	for _, tt := range tests {
		buf, err := appendAsString([]byte("start"), tt.v, tt.typ)
		if err != tt.err {
			t.Fatalf("%v: expected err %v, got %v", tt.v, tt.err, err)
		}
		if string(buf) != tt.out {
			t.Fatalf("%v: expected %q, got %q", tt.v, tt.out, string(buf))
		}
	}
}
func Test_digitCountInt(t *testing.T) {

	tests := []struct {
		n        int64
		expected int
	}{
		{0, 1},
		{10, 2},
		{7940200381, 10},
		{-1, 2},
		{-2817482, 8},
		{9223372036854775807, 19},
		{-9223372036854775807, 20},
		{-9223372036854775808, 20},
	}

	for _, test := range tests {
		got := digitCountInt(test.n)
		if test.expected != got {
			t.Fatalf("%d: expected %d, got %d", test.n, test.expected, got)
		}
	}

}

func Test_digitCountUint(t *testing.T) {

	tests := []struct {
		n        uint64
		expected int
	}{
		{0, 1},
		{10, 2},
		{63471038, 8},
		{18446744073709551615, 20},
	}

	for _, test := range tests {
		got := digitCountUint(test.n)
		if test.expected != got {
			t.Fatalf("%d: expected %d, got %d", test.n, test.expected, got)
		}
	}

}

func Test_eval(t *testing.T) {

	attributes := map[string]any{
		"a": 165,
		"b": map[string]any{
			"c": "foo",
			"e": 1024,
		},
		"d": nil,
	}
	n := decimal.MustInt(5)
	dt := types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)

	tests := []struct {
		expr          []part
		expectedValue any
		expectedType  types.Type
		err           error
	}{
		{[]part{{value: ``, typ: types.String()}}, "", types.String(), nil},
		{[]part{{value: `a`, typ: types.String()}}, "a", types.String(), nil},
		{[]part{{value: n, typ: dt}}, n, dt, nil},
		{[]part{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, 165, types.Int(32), nil},
		{[]part{{path: path{elements: []string{"b", "c"}, decorators: []decorators{0, 0}}, typ: types.String()}}, "foo", types.String(), nil},
		{[]part{{path: path{elements: []string{"b", "e"}, decorators: []decorators{0, 0}}, typ: types.Int(32)}}, 1024, types.Int(32), nil},
		{[]part{{value: `a`, path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, "a165", types.String(), nil},
		{[]part{{path: path{elements: []string{"coalesce"}, decorators: []decorators{0}}, args: [][]part{
			{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}, {value: " boo", typ: types.String()}},
			{{value: "foo", typ: types.String()}},
		}}}, "165 boo", types.String(), nil},
		{[]part{{path: path{elements: []string{"coalesce"}, decorators: []decorators{0}}, args: [][]part{
			{{path: path{elements: []string{"d"}, decorators: []decorators{0}}, typ: types.String()}},
			{{path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}, {value: " boo", typ: types.String()}},
		}}}, "165 boo", types.String(), nil},
		{[]part{{value: ``, typ: types.String()}, {path: path{elements: []string{"a"}, decorators: []decorators{0}}, typ: types.Int(32)}}, "165", types.String(), nil},
		{[]part{{path: path{elements: []string{"x"}, decorators: []decorators{0}}, typ: types.Boolean()}}, nil, types.Type{}, nil},
		{[]part{{path: path{elements: []string{"b", "x"}, decorators: []decorators{0, 0}}, typ: types.Boolean()}}, nil, types.Boolean(), nil},
	}

	for i, test := range tests {
		got, typ, err := eval(test.expr, "", attributes)
		if err != nil {
			if test.err == nil {
				t.Fatalf("%d. unexpected error: %s", i+1, err)
			}
			if err.Error() != test.err.Error() {
				t.Fatalf("%d. expected error %q, got error %q", i+1, test.err.Error(), err.Error())
			}
			continue
		}
		if test.err != nil {
			t.Fatalf("%d. expected error %q, got no error", i+1, test.err)
		}
		if !reflect.DeepEqual(got, test.expectedValue) {
			t.Fatalf("%d. unexpected value\nexpected %#v\ngot      %#v", i+1, test.expectedValue, got)
		}
		if !types.Equal(typ, test.expectedType) {
			if typ.Valid() {
				t.Fatalf("%d. expected type %s, got %s", i+1, test.expectedType, typ)
			}
			t.Fatalf("%d. expected type %s, got invalid type", i+1, test.expectedType)
		}
	}

}

func Test_substring(t *testing.T) {

	tests := []struct {
		s        string
		start    int
		length   int
		expected string
	}{
		{"", 1, 0, ""},
		{"", 5, 3, ""},
		{"a", 1, 1, "a"},
		{"a", 1, -1, "a"},
		{"a", 2, -1, ""},
		{"hello world", 1, 0, ""},
		{"hello world", 1, -1, "hello world"},
		{"hello world", 1, 5, "hello"},
		{"hello world", 7, 5, "world"},
		{"hello world", 2, 3, "ell"},
		{"hello world", 1, 20, "hello world"},
		{"hello world", 1, 1, "h"},
		{"hello world", 11, 1, "d"},
		{"hello world", 11, 20, "d"},
		{"hello world", 12, 5, ""},
		{"hello world", 3, -1, "llo world"},
		{"hello world", 3, 9, "llo world"},
		{"hello world", 12, -1, ""},
		{"hello world", 50, -1, ""},
		{"The café is ready for lunch at the résidence", 8, 1, "é"},
		{"The café is ready for lunch at the résidence", 8, 30, "é is ready for lunch at the ré"},
		{"The café is ready for lunch at the résidence", 8, 31, "é is ready for lunch at the rés"},
		{"日本の文化はとても興味深いです。", 1, 5, "日本の文化"},
		{"日本の文化はとても興味深いです。", 6, 7, "はとても興味深"},
		{"日本の文化はとても興味深いです。", 10, -1, "興味深いです。"},
	}

	for _, test := range tests {
		got := substring(test.s, test.start, test.length)
		if test.expected != got {
			t.Fatalf("expected %q, got %q", test.expected, got)
		}
	}

}

func Test_valueOf(t *testing.T) {

	attributes := map[string]any{
		"a": 5, // int
		"b": map[string]any{ // object
			"c": "foo", // string
			"d": map[string]any{ // map(array(int))
				"e":  []any{1},
				".e": []any{2},
				"e]": []any{3},
			},
		},
		"f": nil, // string
		"g": json.Value("12.53"),
		"h": json.Value(`{"i":true,"i?":5,"?i?":"boo","[i":"foo","i]":"zoo"}`),
		"l": json.Value(`{"name":"Bob","email":"bob@axample.com"}`),
	}

	tests := []struct {
		path     path
		expected any
		err      error
	}{
		{path{elements: []string{"a"}, decorators: []decorators{0}}, 5, nil},
		{path{elements: []string{"a"}, decorators: []decorators{indexing}}, 5, nil},
		{path{elements: []string{"b", "c"}, decorators: []decorators{0, 0}}, "foo", nil},
		{path{elements: []string{"b", "c"}, decorators: []decorators{0, indexing}}, "foo", nil},
		{path{elements: []string{"b", "x"}, decorators: []decorators{0, 0}}, nil, nil},
		{path{elements: []string{"b", "d", "e"}, decorators: []decorators{0, 0, 0}}, []any{1}, nil},
		{path{elements: []string{"b", "d", ".e"}, decorators: []decorators{0, 0, indexing}}, []any{2}, nil},
		{path{elements: []string{"b", "d", "e]"}, decorators: []decorators{0, 0, indexing}}, []any{3}, nil},
		{path{elements: []string{"b", "d", "x"}, decorators: []decorators{0, 0, 0}}, nil, nil},
		{path{elements: []string{"f"}, decorators: []decorators{0}}, nil, nil},
		{path{elements: []string{"g"}, decorators: []decorators{0}}, json.Value("12.53"), nil},
		{path{elements: []string{"g", "x"}, decorators: []decorators{0, 0}}, nil, TransformationError{msg: `invalid g.x: g is not JSON object, it is number`}},
		{path{elements: []string{"g", "x"}, decorators: []decorators{0, indexing}}, nil, TransformationError{msg: `invalid g["x"]: g is not JSON object, it is number`}},
		{path{elements: []string{"h", "i"}, decorators: []decorators{0, 0}}, json.Value("true"), nil},
		{path{elements: []string{"h", "i"}, decorators: []decorators{0, optional}}, json.Value("true"), nil},
		{path{elements: []string{"h", "i?"}, decorators: []decorators{0, indexing | optional}}, json.Value("5"), nil},
		{path{elements: []string{"h", "?i?"}, decorators: []decorators{0, indexing}}, json.Value(`"boo"`), nil},
		{path{elements: []string{"h", "?i?"}, decorators: []decorators{0, indexing | optional}}, json.Value(`"boo"`), nil},
		{path{elements: []string{"h", "[i"}, decorators: []decorators{0, indexing}}, json.Value(`"foo"`), nil},
		{path{elements: []string{"h", "[i"}, decorators: []decorators{0, indexing | optional}}, json.Value(`"foo"`), nil},
		{path{elements: []string{"h", "i]"}, decorators: []decorators{0, indexing}}, json.Value(`"zoo"`), nil},
		{path{elements: []string{"h", "i]"}, decorators: []decorators{0, indexing | optional}}, json.Value(`"zoo"`), nil},
		{path{elements: []string{"h", "i", "x"}, decorators: []decorators{0, 0, 0}}, nil, TransformationError{msg: `invalid h.i.x: h.i is not JSON object, it is true`}},
		{path{elements: []string{"h", "i", "x"}, decorators: []decorators{0, 0, optional}}, nil, nil},
		{path{elements: []string{"h", "x"}, decorators: []decorators{0, 0}}, nil, nil},
		{path{elements: []string{"h", "i", "x"}, decorators: []decorators{0, 0, indexing | optional}}, nil, nil},
		{path{elements: []string{"x"}, decorators: []decorators{0}}, nil, nil},
		{path{elements: []string{"l", "email"}, decorators: []decorators{0, 0}}, json.Value(`"bob@axample.com"`), nil},
		{path{elements: []string{"l", "name"}, decorators: []decorators{0, 0}}, json.Value(`"Bob"`), nil},
	}

	for _, test := range tests {
		got, err := valueOf(test.path, attributes)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("%s. expected error %v (type %T), got error %v (%T)", test.path, test.err, test.err, err, err)
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatalf("%s. unexpected value\nexpected %v (type %T)\ngot      %v (type %T)", test.path, test.expected, test.expected, got, got)
		}

	}

}
