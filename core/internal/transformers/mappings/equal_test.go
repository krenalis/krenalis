// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestJSONEquality compares JSON with native and JSON values without losing
// numeric precision or structure.
func TestJSONEquality(t *testing.T) {

	object := types.Object([]types.Property{{Name: "x", Type: types.Array(types.Decimal(6, 2))}})
	largeText := strings.Repeat("x", 10000)
	tests := []struct {
		name      string
		left      string
		rightType types.Type
		right     any
		equal     bool
	}{
		{"adjacent integers", "9007199254740993", types.Float(64), 9007199254740992.0, false},
		{"exact integer", "9007199254740992", types.Float(64), 9007199254740992.0, true},
		{"signed integer", "-9223372036854775808", types.Int(64), int(math.MinInt64), true},
		{"unsigned integer", "18446744073709551615", types.Int(64).Unsigned(), uint(math.MaxUint64), true},
		{"decimal representation", "1.50e0", types.Decimal(6, 2), decimal.New(15, 1), true},
		{"exact fraction", "0.125", types.Float(64), 0.125, true},
		{"inexact fraction", "0.1", types.Float(64), 0.1, false},
		{"binary fraction", "0.1000000000000000055511151231257827021181583404541015625", types.Float(64), 0.1, true},
		{"NaN", "0", types.Float(64), math.NaN(), false},
		{"infinity", "1e9999", types.Float(64), math.Inf(1), false},
		{"negative infinity", "-1e9999", types.Float(64), math.Inf(-1), false},
		{"signed zero", "-0.000e99999999999999999999999", types.Float(64), 0.0, true},
		{"exponent spelling", " 15.0e-1 ", types.JSON(), json.Value("1.5"), true},
		{"different numbers", "1.4", types.JSON(), json.Value("1"), false},
		{"negative numbers", "-15e-1", types.JSON(), json.Value("-1.5"), true},
		{"opposite signs", "-15e-1", types.JSON(), json.Value("1.5"), false},
		{"large exponent", "1e99999999999999999999999", types.JSON(), json.Value("10e99999999999999999999998"), true},
		{
			"small exponent", "1e-99999999999999999999999", types.JSON(),
			json.Value("10e-100000000000000000000000"), true,
		},
		{
			"exponent difference", "1e99999999999999999999999", types.JSON(),
			json.Value("1e99999999999999999999998"), false,
		},
		{"native array", "[1.4]", types.Array(types.Int(32)), []any{1}, false},
		{"native nested object", `{"x":[1.5]}`, object, map[string]any{"x": []any{decimal.New(15, 1)}}, true},
		{"native map", `{"x":1.4}`, types.Map(types.Int(32)), map[string]any{"x": 1}, false},
		{"nested JSON", `{"x":[9007199254740993]}`, types.JSON(), json.Value(`{"x":[9007199254740992]}`), false},
		{"JSON array length", "[1,2]", types.JSON(), json.Value("[1]"), false},
		{"JSON array order", "[1,2]", types.JSON(), json.Value("[2,1]"), false},
		{"JSON object keys", `{"x":1}`, types.JSON(), json.Value(`{"y":1}`), false},
		{"JSON extra property", `{"x":1,"y":2}`, types.JSON(), json.Value(`{"x":1}`), false},
		{"JSON missing and null", `{}`, types.JSON(), json.Value(`{"x":null}`), false},
		{"JSON object order", `{"x":1,"y":2}`, types.JSON(), json.Value(`{ "y":2.0,"x":1e0 }`), true},
		{
			"JSON escaped keys", `{"a\u002eb":["\u0041",true,null]}`, types.JSON(),
			json.Value(`{"a.b":["A",true,null]}`), true,
		},
		{"JSON array and scalar", "[1]", types.JSON(), json.Value("1"), false},
		{"JSON object and scalar", `{"x":1}`, types.JSON(), json.Value("1"), false},
		{"JSON empty array and object", `[]`, types.JSON(), json.Value(`{}`), false},
		{"JSON booleans", "true", types.JSON(), json.Value("false"), false},
		{"JSON number and string", "1", types.JSON(), json.Value(`"1"`), false},
		{"JSON boolean and string", "true", types.JSON(), json.Value(`"true"`), false},
		{"JSON string and native number", `"1"`, types.Int(32), int(1), false},
		{"JSON null", "null", types.JSON(), json.Value("null"), true},
		{
			"decoder buffers", `{"pad":"` + largeText + `","x":[1.5]}`, types.Map(types.JSON()),
			map[string]any{"pad": json.Value(`"` + largeText + `"`), "x": json.Value(`[15e-1]`)}, true,
		},
	}
	for _, test := range tests {

		for _, fn := range []string{"eq", "ne"} {

			for _, args := range []string{"a, b", "b, a", "coalesce(a), if(true, b)"} {

				t.Run(test.name+"/"+fn+"("+args+")", func(t *testing.T) {

					schema := types.Object([]types.Property{
						{Name: "a", Type: types.JSON()}, {Name: "b", Type: test.rightType},
					})
					expr, _, err := Compile(fn+"("+args+")", schema, types.Boolean())
					if err != nil {
						t.Fatal(err)
					}

					got, typ, err := expr.Eval(map[string]any{"a": json.Value(test.left), "b": test.right})
					if err != nil {
						t.Fatal(err)
					}
					want := test.equal
					if fn == "ne" {
						want = !want
					}
					if got != want || !types.Equal(typ, types.Boolean()) {
						t.Fatalf("got %v (%s), want %v (boolean)", got, typ, want)
					}

				})

			}

		}

	}

}

// TestJSONNumberRepresentations checks decimal normalization against
// independent rational arithmetic.
func TestJSONNumberRepresentations(t *testing.T) {

	numbers := []string{
		"0", "-0.000e+19", "0.00001", "1e-5", "10e-6", "100e-7",
		"1.00001e-5", "0.1", "1.23000e+04", "12300", "12000.0e-2", "120",
		"0.00123000e-2", "0.0000123", "12.0e+0000000000000003", "12000",
		"-1234.500", "-123450e-2", "9007199254740992", "9007199254740993",
	}
	schema := types.Object([]types.Property{{Name: "a", Type: types.JSON()}, {Name: "b", Type: types.JSON()}})
	expr, _, err := Compile("eq(a, b)", schema, types.Boolean())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range numbers {

		for _, b := range numbers {

			var n0, n1 big.Rat
			if _, ok := n0.SetString(a); !ok {
				t.Fatalf("invalid test number %q", a)
			}
			if _, ok := n1.SetString(b); !ok {
				t.Fatalf("invalid test number %q", b)
			}

			got, _, err := expr.Eval(map[string]any{"a": json.Value(a), "b": json.Value(b)})
			if err != nil {
				t.Fatal(err)
			}
			if want := n0.Cmp(&n1) == 0; got != want {
				t.Fatalf("eq(%s, %s) = %v, want %v", a, b, got, want)
			}

		}

	}

}

// TestNumericEquality checks exact, symmetric eq/ne results for scalars and
// nested containers.
func TestNumericEquality(t *testing.T) {

	tests := []struct {
		name        string
		leftType    types.Type
		rightType   types.Type
		left, right any
		equal       bool
	}{
		{
			"decimal representation", types.Decimal(6, 2), types.Decimal(6, 2),
			decimal.New(15, 1), decimal.MustParse("1.5"), true,
		},
		{
			"decimal scale", types.Decimal(6, 3), types.Decimal(6, 2),
			decimal.MustParse("1.234"), decimal.MustParse("1.23"), false,
		},
		{
			"decimal and integer", types.Decimal(76, 0), types.Int(64),
			decimal.MustParse("9223372036854775807"), int(math.MaxInt64), true,
		},
		{
			"decimal and unsigned integer", types.Decimal(76, 0), types.Int(64).Unsigned(),
			decimal.MustParse("18446744073709551615"), uint(math.MaxUint64), true,
		},
		{"fraction and integer", types.Float(64), types.Int(32), 1.4, int(1), false},
		{"whole float and integer", types.Float(64), types.Int(32), 1.0, int(1), true},
		{"signed and unsigned", types.Int(32), types.Int(64).Unsigned(), int(42), uint(42), true},
		{"negative and unsigned", types.Int(32), types.Int(64).Unsigned(), int(-1), uint(math.MaxUint64), false},
		{"different float precision", types.Float(32), types.Float(64), float64(float32(0.1)), 0.1, false},
		{"beyond float precision", types.Int(64), types.Float(64), int(9007199254740993), 9007199254740992.0, false},
		{"maximum signed integer", types.Int(64), types.Float(64), int(math.MaxInt64), float64(math.MaxInt64), false},
		{"minimum signed integer", types.Int(64), types.Float(64), int(math.MinInt64), float64(math.MinInt64), true},
		{
			"maximum unsigned integer", types.Int(64).Unsigned(), types.Float(64),
			uint(math.MaxUint64), float64(math.MaxUint64), false,
		},
		{"exact fraction", types.Decimal(6, 2), types.Float(64), decimal.MustParse("1.5"), 1.5, true},
		{"inexact fraction", types.Decimal(6, 2), types.Float(64), decimal.MustParse("0.1"), 0.1, false},
		{"subnormal and zero", types.Float(64), types.Decimal(6, 2), math.SmallestNonzeroFloat64, decimal.Decimal{}, false},
		{"NaN", types.Float(64), types.Float(64), math.NaN(), math.NaN(), true},
		{
			"NaN representations", types.Float(64), types.Float(64),
			math.Float64frombits(0x7ff8000000000001), math.Float64frombits(0xfff0000000000001), true,
		},
		{"NaN precisions", types.Float(32), types.Float(64), math.NaN(), math.NaN(), true},
		{"NaN and infinity", types.Float(64), types.Float(64), math.NaN(), math.Inf(1), false},
		{"NaN and finite float", types.Float(64), types.Float(64), math.NaN(), 0.0, false},
		{"NaN and integer", types.Float(64), types.Int(32), math.NaN(), 0, false},
		{"NaN and decimal", types.Float(64), types.Decimal(6, 2), math.NaN(), decimal.Decimal{}, false},
		{"positive infinity", types.Float(64), types.Float(64), math.Inf(1), math.Inf(1), true},
		{"opposite infinities", types.Float(64), types.Float(64), math.Inf(-1), math.Inf(1), false},
		{"infinity and decimal", types.Float(64), types.Decimal(76, 0), math.Inf(1), decimal.MustParse("1e75"), false},
		{"signed zero", types.Float(64), types.Int(32), math.Copysign(0, -1), int(0), true},
	}
	for _, test := range tests {

		for _, container := range []string{"scalar", "array", "map", "object", "nested"} {

			leftType, rightType := test.leftType, test.rightType
			left, right := test.left, test.right
			switch container {
			case "array":
				leftType, rightType = types.Array(leftType), types.Array(rightType)
				left, right = []any{left}, []any{right}
			case "map":
				leftType, rightType = types.Map(leftType), types.Map(rightType)
				left, right = map[string]any{"x": left}, map[string]any{"x": right}
			case "object", "nested":
				leftType = types.Object([]types.Property{{Name: "x", Type: leftType}})
				rightType = types.Object([]types.Property{{Name: "x", Type: rightType}})
				left, right = map[string]any{"x": left}, map[string]any{"x": right}
				if container == "nested" {
					leftType, rightType = types.Array(types.Map(leftType)), types.Array(types.Map(rightType))
					left, right = []any{map[string]any{"key": left}}, []any{map[string]any{"key": right}}
				}
			}

			for _, fn := range []string{"eq", "ne"} {

				for _, args := range []string{"a, b", "b, a", "coalesce(a), if(true, b)"} {

					t.Run(test.name+"/"+container+"/"+fn+"("+args+")", func(t *testing.T) {

						schema := types.Object([]types.Property{
							{Name: "a", Type: leftType}, {Name: "b", Type: rightType},
						})
						expr, _, err := Compile(fn+"("+args+")", schema, types.Boolean())
						if err != nil {
							t.Fatal(err)
						}

						got, typ, err := expr.Eval(map[string]any{"a": left, "b": right})
						if err != nil {
							t.Fatal(err)
						}
						want := test.equal
						if fn == "ne" {
							want = !want
						}
						if got != want || !types.Equal(typ, types.Boolean()) {
							t.Fatalf("got %v (%s), want %v (boolean)", got, typ, want)
						}

					})

				}

			}

		}

	}

}

// TestNumericEqualityLiterals checks literal representations and the existing
// string and null comparisons.
func TestNumericEqualityLiterals(t *testing.T) {

	tests := []struct {
		source string
		want   any
	}{
		{"eq(1.5, 15e-1)", true},
		{"ne(1.5, 15e-1)", false},
		{"eq(.5, 0.5)", true},
		{"eq(1, '1')", true},
		{"eq('1', 1)", true},
		{"eq('a', 5)", false},
		{"eq(5, 'a')", false},
		{"ne('a', 5)", true},
		{"eq(null, 1)", nil},
		{"ne(1, null)", nil},
	}
	for _, test := range tests {

		t.Run(test.source, func(t *testing.T) {

			expr, _, err := Compile(test.source, types.Type{}, types.Boolean())
			if err != nil {
				t.Fatal(err)
			}
			got, typ, err := expr.Eval(nil)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want || !types.Equal(typ, types.Boolean()) {
				t.Fatalf("got %v (%s), want %v (boolean)", got, typ, test.want)
			}

		})

	}

}

// TestIntegerLiteralEquality checks string comparisons at the signed integer
// boundaries in both orders.
func TestIntegerLiteralEquality(t *testing.T) {

	numbers := []string{
		"0", "2147483647", "-2147483648", "2147483648", "-2147483649",
		"9223372036854775807", "-9223372036854775808",
	}

	for _, number := range numbers {

		quoted := fmt.Sprintf("%q", number)

		for _, fn := range []string{"eq", "ne"} {
			for _, args := range []string{quoted + ", " + number, number + ", " + quoted} {

				source := fn + "(" + args + ")"

				t.Run(source, func(t *testing.T) {

					expr, _, err := Compile(source, types.Type{}, types.Boolean())
					if err != nil {
						t.Fatal(err)
					}

					got, typ, err := expr.Eval(nil)
					if err != nil {
						t.Fatal(err)
					}

					want := fn == "eq"
					if got != want || !types.Equal(typ, types.Boolean()) {
						t.Fatalf("got %#v (%s), want %v (boolean)", got, typ, want)
					}

				})

			}
		}

	}

}

// TestScalarEquality checks symmetric coercions through properties, containers,
// and selectors.
func TestScalarEquality(t *testing.T) {

	tests := []struct {
		name                string
		leftType, rightType types.Type
		left, right         string
		equal               bool
	}{
		{"empty enum", types.String().WithValues("", "yes"), types.String(), `""`, `""`, true},
		{
			"different empty enums", types.String().WithValues("", "yes"), types.String().WithValues("", "no"),
			`""`, `""`, true,
		},
		{"leading zero", types.String(), types.Int(32), `"01"`, `1`, true},
		{"boolean spelling", types.String(), types.Boolean(), `"TRUE"`, `true`, true},
		{"boolean alias", types.String(), types.Boolean(), `"yes"`, `true`, true},
		{"false alias", types.String(), types.Boolean(), `"NO"`, `false`, true},
		{"invalid boolean", types.String(), types.Boolean(), `"invalid"`, `false`, false},
		{"float text", types.String(), types.Float(64), `"0.1"`, `0.1`, true},
		{"decimal text", types.String(), types.Decimal(6, 2), `"1.50"`, `1.5`, true},
		{"invalid integer", types.String(), types.Int(32), `"1.4"`, `1`, false},
		{"nonzero int8", types.Int(8), types.Boolean(), `2`, `true`, false},
		{"negative int8", types.Int(8), types.Boolean(), `-1`, `true`, false},
		{"one int8", types.Int(8), types.Boolean(), `1`, `true`, true},
		{"zero int8", types.Int(8), types.Boolean(), `0`, `false`, true},
		{"unsigned int8", types.Int(8).Unsigned(), types.Boolean(), `2`, `true`, false},
		{"one unsigned int8", types.Int(8).Unsigned(), types.Boolean(), `1`, `true`, true},
		{"year", types.Year(), types.Int(32), `2026`, `2026`, true},
		{"different year", types.Year(), types.Int(32), `2026`, `2027`, false},
		{
			"datetime and date", types.DateTime(), types.Date(),
			`"2026-09-08T12:34:56Z"`, `"2026-09-08"`, false,
		},
		{
			"midnight and date", types.DateTime(), types.Date(),
			`"2026-09-08T00:00:00Z"`, `"2026-09-08"`, true,
		},
		{"date spelling", types.String(), types.Date(), `"09/08/2026"`, `"2026-09-08"`, true},
		{"time spelling", types.String(), types.Time(), `"12:34:56.120"`, `"12:34:56.12"`, true},
		{"JSON number and native string", types.JSON(), types.String(), `1`, `"1"`, false},
		{"JSON boolean and native string", types.JSON(), types.String(), `true`, `"true"`, false},
		{"JSON string and native boolean", types.JSON(), types.Boolean(), `"true"`, `true`, false},
		{"JSON string and native number", types.JSON(), types.Int(32), `"1"`, `1`, false},
		{"JSON number and JSON string", types.JSON(), types.JSON(), `1`, `"1"`, false},
		{"JSON boolean and native int8", types.JSON(), types.Int(8), `true`, `1`, false},
		{"JSON string and native string", types.JSON(), types.String(), `"01"`, `"01"`, true},
		{"JSON date", types.JSON(), types.Date(), `"2026-09-08"`, `"2026-09-08"`, true},
		{"numeric control", types.Float(64), types.Int(32), `1.4`, `1`, false},
		{"whole float", types.Float(64), types.Int(32), `1.0`, `1`, true},
		{
			"UUID spelling", types.String(), types.UUID(),
			`"B902C3A3-FC39-44B4-8A3E-E15C4883932E"`, `"b902c3a3-fc39-44b4-8a3e-e15c4883932e"`, true,
		},
		{"IP spelling", types.String(), types.IP(), `"2001:0db8::1"`, `"2001:db8::1"`, true},
	}

	for _, test := range tests {

		for _, shape := range []string{"scalar", "array", "map", "object", "nested"} {

			st0, st1 := test.leftType, test.rightType
			input0, input1 := test.left, test.right
			switch shape {
			case "array":
				st0, st1 = types.Array(st0), types.Array(st1)
				input0, input1 = "["+input0+"]", "["+input1+"]"
			case "map":
				st0, st1 = types.Map(st0), types.Map(st1)
				input0, input1 = `{"x":`+input0+`}`, `{"x":`+input1+`}`
			case "object", "nested":
				st0 = types.Object([]types.Property{{Name: "x", Type: st0}})
				st1 = types.Object([]types.Property{{Name: "x", Type: st1}})
				input0, input1 = `{"x":`+input0+`}`, `{"x":`+input1+`}`
				if shape == "nested" {
					st0, st1 = types.Array(types.Map(st0)), types.Array(types.Map(st1))
					input0, input1 = `[{"item":`+input0+`}]`, `[{"item":`+input1+`}]`
				}
			}

			for _, selected := range []bool{false, true} {

				t.Run(fmt.Sprintf("%s/%s/selected=%t", test.name, shape, selected), func(t *testing.T) {

					inSchema := types.Object([]types.Property{{Name: "a", Type: st0}, {Name: "b", Type: st1}})
					input := `{"a":` + input0 + `,"b":` + input1 + `}`
					attributes, err := types.Decode[map[string]any](strings.NewReader(input), inSchema)
					if err != nil {
						t.Fatal(err)
					}
					a, b := "a", "b"
					if selected {
						a, b = "if(true, a)", "coalesce(b)"
					}
					expressions := map[string]string{
						"eqab": "eq(" + a + ", " + b + ")", "eqba": "eq(" + b + ", " + a + ")",
						"neab": "ne(" + a + ", " + b + ")", "neba": "ne(" + b + ", " + a + ")",
					}
					outSchema := types.Object([]types.Property{
						{Name: "eqab", Type: types.Boolean()}, {Name: "eqba", Type: types.Boolean()},
						{Name: "neab", Type: types.Boolean()}, {Name: "neba", Type: types.Boolean()},
					})
					mapping, err := New(expressions, inSchema, outSchema, false, nil)
					if err != nil {
						t.Fatal(err)
					}
					got, err := mapping.Transform(attributes, None)
					if err != nil {
						t.Fatal(err)
					}

					want := map[string]any{
						"eqab": test.equal, "eqba": test.equal, "neab": !test.equal, "neba": !test.equal,
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("got %v, want %v", got, want)
					}

				})

			}

		}

	}

}

// TestScalarEqualityLiterals applies the same coercion rules to literals and
// function results.
func TestScalarEqualityLiterals(t *testing.T) {

	tests := []struct {
		left, right string
		equal       bool
	}{
		{`'01'`, `1`, true},
		{`'TRUE'`, `true`, true},
		{`lower('YES')`, `true`, true},
		{`json_parse('1')`, `'1'`, false},
		{`json_parse('true')`, `'true'`, false},
	}

	for _, test := range tests {

		for _, fn := range []string{"eq", "ne"} {

			for _, args := range []string{test.left + ", " + test.right, test.right + ", " + test.left} {

				source := fn + "(" + args + ")"
				t.Run(source, func(t *testing.T) {

					expr, _, err := Compile(source, types.Type{}, types.Boolean())
					if err != nil {
						t.Fatal(err)
					}
					got, typ, err := expr.Eval(nil)
					if err != nil {
						t.Fatal(err)
					}

					want := test.equal
					if fn == "ne" {
						want = !want
					}
					if got != want || !types.Equal(typ, types.Boolean()) {
						t.Fatalf("got %v (%s), want %v (boolean)", got, typ, want)
					}

				})

			}

		}

	}

}

// TestTemporalEquality compares logical temporal values through properties,
// containers, and selectors.
func TestTemporalEquality(t *testing.T) {

	instant := time.Date(2026, 9, 8, 10, 0, 0, 125000000, time.UTC)
	date := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 10, 0, 0, 125000000, time.UTC)
	utc := time.FixedZone("UTC", 0)
	offset := time.FixedZone("UTC+02", 2*60*60)
	now := time.Now()
	tests := []struct {
		name        string
		typ         types.Type
		left, right time.Time
		equal       bool
	}{
		{"datetime location", types.DateTime(), instant, instant.In(utc), true},
		{"datetime offset", types.DateTime(), instant, instant.In(offset), true},
		{"datetime different instant", types.DateTime(), instant, instant.Add(time.Nanosecond), false},
		{"datetime monotonic", types.DateTime(), now, now.Round(0), true},
		{"date location", types.Date(), date, date.In(utc), true},
		{"date offset", types.Date(), date, time.Date(2026, 9, 8, 0, 0, 0, 0, offset), true},
		{"date different day", types.Date(), date, date.AddDate(0, 0, 1), false},
		{"time location", types.Time(), clock, clock.In(utc), true},
		{"time offset", types.Time(), clock, time.Date(1970, 1, 1, 10, 0, 0, 125000000, offset), true},
		{"time reference date", types.Time(), clock, instant, true},
		{"time different hour", types.Time(), clock, clock.Add(time.Hour), false},
		{"time different fraction", types.Time(), clock, clock.Add(time.Nanosecond), false},
	}

	for _, test := range tests {
		for _, shape := range []string{"scalar", "array", "map", "object"} {

			typ := test.typ
			var left, right any = test.left, test.right
			switch shape {
			case "array":
				typ = types.Array(typ)
				left, right = []any{left}, []any{right}
			case "map":
				typ = types.Map(typ)
				left, right = map[string]any{"x": left}, map[string]any{"x": right}
			case "object":
				typ = types.Object([]types.Property{{Name: "x", Type: typ}})
				left, right = map[string]any{"x": left}, map[string]any{"x": right}
			}

			for _, fn := range []string{"eq", "ne"} {
				for _, args := range []string{"a, b", "b, a", "if(true, a), coalesce(b)"} {
					t.Run(fmt.Sprintf("%s/%s/%s(%s)", test.name, shape, fn, args), func(t *testing.T) {

						schema := types.Object([]types.Property{{Name: "a", Type: typ}, {Name: "b", Type: typ}})
						expr, _, err := Compile(fn+"("+args+")", schema, types.Boolean())
						if err != nil {
							t.Fatal(err)
						}

						got, resultType, err := expr.Eval(map[string]any{"a": left, "b": right})
						if err != nil {
							t.Fatal(err)
						}
						want := test.equal
						if fn == "ne" {
							want = !want
						}
						if got != want || !types.Equal(resultType, types.Boolean()) {
							t.Fatalf("got %v (%s), want %v (boolean)", got, resultType, want)
						}

					})
				}
			}

		}
	}

}
