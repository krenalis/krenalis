// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
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
	if iErr.Error() != "«x», with a value of 5.5, cannot be passed as a 32-bit int to the «fn» function" {
		t.Fatalf("expected a conversion error containing the value 5.5, got %v", iErr)
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

// TestSubstringArguments checks argument conversions and verifies that errors
// identify the argument that caused them.
func TestSubstringArguments(t *testing.T) {

	tests := []struct {
		name       string
		expression string
		typ        types.Type
		value      any
		attributes map[string]any
		expected   any
		message    string
	}{
		{name: "string start", expression: "substring('abc', value)", typ: types.String(), value: "2", expected: "bc"},
		{name: "string length", expression: "substring('abc', 1, value)", typ: types.String(), value: "2", expected: "ab"},
		{name: "fractional start", expression: "substring('abc', value)", typ: types.Float(64), value: 1.5, expected: "bc"},
		{name: "fractional length", expression: "substring('abc', 1, value)", typ: types.Float(64), value: 1.5, expected: "ab"},
		{name: "JSON numeric start", expression: "substring('abc', value)", typ: types.JSON(), value: json.Value(`2`), expected: "bc"},
		{name: "JSON numeric length", expression: "substring('abc', 1, value)", typ: types.JSON(), value: json.Value(`2`), expected: "ab"},
		{name: "unsigned start", expression: "substring('abc', value)", typ: types.Int(8).Unsigned(), value: uint(2), expected: "bc"},
		{name: "unsigned length", expression: "substring('abc', 1, value)", typ: types.Int(32).Unsigned(), value: uint(2), expected: "ab"},
		{name: "unsigned start and length", expression: "substring('aé🙂z', value, value)", typ: types.Int(16).Unsigned(), value: uint(2), expected: "é🙂"},
		{name: "zero unsigned start", expression: "substring('abc', value)", typ: types.Int(24).Unsigned(), value: uint(0), expected: "abc"},
		{name: "zero unsigned length", expression: "substring('abc', 1, value)", typ: types.Int(64).Unsigned(), value: uint(0), expected: ""},
		{name: "maximum start", expression: "substring('abc', value)", typ: types.Int(32).Unsigned(), value: uint(2147483647), expected: ""},
		{name: "maximum length", expression: "substring('abc', 1, value)", typ: types.Int(32).Unsigned(), value: uint(2147483647), expected: "abc"},
		{name: "minimum start", expression: "substring('abc', value)", typ: types.Int(32), value: -2147483648, expected: "abc"},
		{name: "null start", expression: "substring('abc', value)", typ: types.String(), value: nil, expected: nil},
		{name: "null length", expression: "substring('abc', 1, value)", typ: types.String(), value: nil, expected: nil},
		{
			name: "invalid start with two arguments", expression: "substring('abc', value)",
			typ: types.String(), value: "bad",
			message: "«value», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "invalid start with three arguments", expression: "substring('abc', value, 1)",
			typ: types.String(), value: "bad",
			message: "«value», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "invalid length", expression: "substring('abc', 1, value)",
			typ: types.String(), value: "bad",
			message: "«value», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "invalid start with dynamic length", expression: "substring('abc', start, length)",
			typ: types.String(), attributes: map[string]any{"start": "bad", "length": "1"},
			message: "«start», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "invalid length with dynamic start", expression: "substring('abc', start, length)",
			typ: types.String(), attributes: map[string]any{"start": "1", "length": "bad"},
			message: "«length», of type string, cannot be passed as int to the «substring» function",
		},
		{
			name: "unsigned start overflow", expression: "substring('abc', value)",
			typ: types.Int(32).Unsigned(), value: uint(2147483648),
			message: "«value», with a value of 2147483648, cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "unsigned length overflow", expression: "substring('abc', 1, value)",
			typ: types.Int(32).Unsigned(), value: uint(2147483648),
			message: "«value», with a value of 2147483648, cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "signed start underflow", expression: "substring('abc', value)",
			typ: types.Int(64), value: -2147483649,
			message: "«value», with a value of -2147483649, cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "signed length underflow", expression: "substring('abc', 1, value)",
			typ: types.Int(64), value: -2147483649,
			message: "«value», with a value of -2147483649, cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "float start overflow", expression: "substring('abc', value)",
			typ: types.Float(64), value: 2147483648.0,
			message: "«value», with a value of 2.147483648e+09, cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "decimal length overflow", expression: "substring('abc', 1, value)",
			typ: types.Decimal(10, 0), value: decimal.MustParse("2147483648"),
			message: "«value», with a value of 2147483648, cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "JSON start object", expression: "substring('abc', value)",
			typ: types.JSON(), value: json.Value(`{}`),
			message: "«value», of type JSON object, cannot be passed as an int to the «substring» function",
		},
		{
			name: "JSON length array", expression: "substring('abc', 1, value)",
			typ: types.JSON(), value: json.Value(`[]`),
			message: "«value», of type JSON array, cannot be passed as an int to the «substring» function",
		},
		{
			name: "JSON numeric start overflow", expression: "substring('abc', value)",
			typ: types.JSON(), value: json.Value(`2147483648`),
			message: "«value», with a value of 2147483648, cannot be passed as a 32-bit int to the «substring» function",
		},
		{
			name: "JSON source array", expression: "substring(value, 1)",
			typ: types.JSON(), value: json.Value(`[]`),
			message: "«value» (a JSON array) cannot be converted to a string value to be passed to the «substring» function",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			inSchema := types.Object([]types.Property{
				{Name: "value", Type: test.typ, Nullable: true},
				{Name: "start", Type: types.String(), Nullable: true},
				{Name: "length", Type: types.String(), Nullable: true},
			})
			outSchema := types.Object([]types.Property{{Name: "out", Type: types.String(), Nullable: true}})
			mapping, err := New(map[string]string{"out": test.expression}, inSchema, outSchema, "")
			if err != nil {
				t.Fatalf("expected mapping construction to succeed, got %v", err)
			}

			attributes := map[string]any{"value": test.value}
			maps.Copy(attributes, test.attributes)
			got, err := mapping.Transform(attributes, None)
			if err != nil {
				if test.message == "" {
					t.Fatalf("expected transformation to succeed, got %v", err)
				}
				if _, ok := errors.AsType[TransformationError](err); !ok {
					t.Errorf("expected TransformationError, got %T: %v", err, err)
				}
				expected := test.message + " while mapping to «out»"
				if err.Error() != expected {
					t.Errorf("expected error %q, got %q", expected, err.Error())
				}
				return
			}
			if test.message != "" {
				t.Fatalf("expected conversion error %q, got nil", test.message)
			}
			expected := map[string]any{"out": test.expected}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("expected output %#v, got %#v", expected, got)
			}

		})
	}

}

// TestSubstringEmptyString verifies that substring returns an empty string for
// empty input, with or without a length argument.
func TestSubstringEmptyString(t *testing.T) {

	expressions := []string{
		"substring('', 0)",
		"substring('', 1)",
		"substring('', -1)",
		"substring('', 2)",
		"substring('', 1, 1)",
		"substring('', 0, 1)",
		"substring('', -1, 1)",
		"substring('', 1, 0)",
		"substring('', 2, 1)",
	}
	expectedType := types.String()

	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			expr, _, err := Compile(expression, types.Type{}, expectedType)
			if err != nil {
				t.Fatalf("expected compilation to succeed, got %v", err)
			}
			got, typ, err := expr.Eval(nil)
			if err != nil {
				t.Fatalf("expected evaluation to succeed, got %v", err)
			}
			if got != "" {
				t.Errorf("expected an empty string, got %#v", got)
			}
			if !types.Equal(typ, expectedType) {
				t.Errorf("expected type %s, got %s", expectedType, typ)
			}
		})
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
		{true, types.Boolean(), "starttrue", nil},
		{false, types.Boolean(), "startfalse", nil},
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
		{"", 1, 1, ""},
		{"", 1, -1, ""},
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
		{"é🙂x", 2, 1, "🙂"},
		{"\uFFFDx", 1, 1, "\uFFFD"},
		{"é\uFFFD", 2, 10, "\uFFFD"},
		{"\xff", 1, 1, "\xff"},
		{"\xffab", 1, 1, "\xff"},
		{"\xffé", 1, 1, "\xff"},
		{"é\xff", 2, 1, "\xff"},
		{"\xff\uFFFD", 1, 1, "\xff"},
		{"\uFFFD\xff", 1, 1, "\uFFFD"},
		{"\x80", 1, 1, "\x80"},
		{"\xe2\x82", 1, 1, "\xe2"},
		{"\xe2\x82", 2, 1, "\x82"},
		{"\xe2\x82", 1, 10, "\xe2\x82"},
		{"\xc0\xaf", 1, 1, "\xc0"},
		{"\xed\xa0\x80", 1, 1, "\xed"},
		{"\xf4\x90\x80\x80", 1, 1, "\xf4"},
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
