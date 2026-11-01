// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/go-cmp/cmp"
)

// TestMappingArray checks array conversions and validation.
func TestMappingArray(t *testing.T) {

	date := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	createRequiredObject := types.Object([]types.Property{{
		Name: "required", Type: types.String(), ReadOptional: true, CreateRequired: true,
	}})
	updateRequiredObject := types.Object([]types.Property{{
		Name: "required", Type: types.String(), ReadOptional: true, UpdateRequired: true,
	}})
	tests := []struct {
		name        string
		expression  string
		source      types.Type
		destination types.Type
		value       any
		expected    any
		err         string
		purpose     Purpose
	}{
		{
			name:        "literal date",
			expression:  "array('12/31/2025')",
			source:      types.String(),
			destination: types.Array(types.Date()),
			expected:    []any{date},
		},
		{
			name:        "dynamic date",
			expression:  "array(value)",
			source:      types.String(),
			destination: types.Array(types.Date()),
			value:       "12/31/2025",
			expected:    []any{date},
		},
		{
			name:        "short date",
			expression:  "array(value)",
			source:      types.String(),
			destination: types.Array(types.Date()),
			value:       "12/31/25",
			expected:    []any{date},
		},
		{
			name:        "int8 to boolean",
			expression:  "array(value, 'true', false)",
			source:      types.Int(8),
			destination: types.Array(types.Boolean()),
			value:       1,
			expected:    []any{true, true, false},
		},
		{
			name:        "boolean to int8",
			expression:  "array(value)",
			source:      types.Boolean(),
			destination: types.Array(types.Int(8)),
			value:       true,
			expected:    []any{1},
		},
		{
			name:        "int64 to float32 before range validation",
			expression:  "array(value)",
			source:      types.Int(64),
			destination: types.Array(types.Float(32).WithFloatRange(0, 16777216)),
			value:       16777217,
			expected:    []any{float64(16777216)},
		},
		{
			name:        "uint64 to float32",
			expression:  "array(value)",
			source:      types.Int(64).Unsigned(),
			destination: types.Array(types.Float(32)),
			value:       uint(16777217),
			expected:    []any{float64(16777216)},
		},
		{
			name:        "int64 to float32 without double rounding",
			expression:  "array(value)",
			source:      types.Int(64),
			destination: types.Array(types.Float(32)),
			value:       (1 << 62) + (1 << 38) + 1,
			expected:    []any{float64((1 << 62) + (1 << 39))},
		},
		{
			name:        "uint64 to float32 without double rounding",
			expression:  "array(value)",
			source:      types.Int(64).Unsigned(),
			destination: types.Array(types.Float(32)),
			value:       uint((1 << 62) + (1 << 38) + 1),
			expected:    []any{float64((1 << 62) + (1 << 39))},
		},
		{
			name:        "duplicate after float32 rounding",
			expression:  "array(value, 16777216)",
			source:      types.Int(64),
			destination: types.Array(types.Float(32)).WithUnique(),
			value:       16777217,
			err:         "«array(value, 16777216)» is not convertible to the «array(float(32))» type",
		},
		{
			name:        "decimal to float32 without double rounding",
			expression:  "array(value)",
			source:      types.Decimal(17, 16),
			destination: types.Array(types.Float(32)),
			value:       decimal.MustParse("1.0000000596046448"),
			expected:    []any{1.0000001192092896},
		},
		{
			name:        "unique decimals after float32 rounding",
			expression:  "array(value, 1)",
			source:      types.Decimal(17, 16),
			destination: types.Array(types.Float(32)).WithUnique(),
			value:       decimal.MustParse("1.0000000596046448"),
			expected:    []any{1.0000001192092896, float64(1)},
		},
		{
			name:        "decimal scale",
			expression:  "array(value)",
			source:      types.Decimal(5, 3),
			destination: types.Array(types.Decimal(3, 1)),
			value:       decimal.MustParse("1.234"),
			err:         "«value» is not convertible to the «decimal(3,1)» type",
		},
		{
			name:        "decimal integer digits",
			expression:  "array(value)",
			source:      types.Decimal(5, 2),
			destination: types.Array(types.Decimal(2, 1)),
			value:       decimal.MustParse("12.3"),
			err:         "«value» is not convertible to the «decimal(2,1)» type",
		},
		{
			name:        "decimal trailing zeros",
			expression:  "array(value)",
			source:      types.Decimal(5, 3),
			destination: types.Array(types.Decimal(3, 1)),
			value:       decimal.New(1200, 3),
			expected:    []any{decimal.MustParse("1.2")},
		},
		{
			name:        "decimal zero without integer digits",
			expression:  "array(value)",
			source:      types.Decimal(5, 3),
			destination: types.Array(types.Decimal(1, 1)),
			value:       decimal.Decimal{},
			expected:    []any{decimal.Decimal{}},
		},
		{
			name:        "nested array",
			expression:  "array(array(value))",
			source:      types.String(),
			destination: types.Array(types.Array(types.Date())),
			value:       "12/31/2025",
			expected:    []any{[]any{date}},
		},
		{
			name:        "array property",
			expression:  "array(value)",
			source:      types.Array(types.Int(8)),
			destination: types.Array(types.Array(types.Boolean())),
			value:       []any{1, 0},
			expected:    []any{[]any{true, false}},
		},
		{
			name:        "map property",
			expression:  "array(value)",
			source:      types.Map(types.Int(8)),
			destination: types.Array(types.Map(types.Boolean())),
			value:       map[string]any{"key": 1},
			expected:    []any{map[string]any{"key": true}},
		},
		{
			name:        "object null to string map",
			expression:  "array(value)",
			source:      types.Object([]types.Property{{Name: "key", Type: types.String(), Nullable: true}}),
			destination: types.Array(types.Map(types.String())),
			value:       map[string]any{"key": nil},
			err:         "«value» is not convertible to the «map(string)» type",
		},
		{
			name:        "object null to JSON map",
			expression:  "array(value)",
			source:      types.Object([]types.Property{{Name: "key", Type: types.String(), Nullable: true}}),
			destination: types.Array(types.Map(types.JSON())),
			value:       map[string]any{"key": nil},
			expected:    []any{map[string]any{"key": json.Value("null")}},
		},
		{
			name:        "object property",
			expression:  "array(value)",
			source:      types.Object([]types.Property{{Name: "key", Type: types.Int(8)}}),
			destination: types.Array(types.Object([]types.Property{{Name: "key", Type: types.Boolean()}})),
			value:       map[string]any{"key": 1},
			expected:    []any{map[string]any{"key": true}},
		},
		{
			name:        "coalesce uses value",
			expression:  "array(coalesce(value, '12/30/2025'))",
			source:      types.String(),
			destination: types.Array(types.Date()),
			value:       "12/31/2025",
			expected:    []any{date},
		},
		{
			name:        "coalesce uses fallback",
			expression:  "array(coalesce(value, '12/31/2025'))",
			source:      types.String(),
			destination: types.Array(types.Date()),
			value:       nil,
			expected:    []any{date},
		},
		{
			name:        "JSON result",
			expression:  "array(value, null)",
			source:      types.String(),
			destination: types.JSON(),
			value:       "foo",
			expected:    json.Value(`["foo",null]`),
		},
		{
			name:        "dynamic JSON null",
			expression:  "array(value)",
			source:      types.String(),
			destination: types.Array(types.JSON()),
			value:       nil,
			expected:    []any{json.Value(`null`)},
		},
		{
			name:        "literal NaN",
			expression:  "array('NaN')",
			source:      types.String(),
			destination: types.Array(types.Float(64)),
			expected:    []any{math.NaN()},
		},
		{
			name:        "NaN64",
			expression:  "array(value)",
			source:      types.Float(64),
			destination: types.Array(types.Float(64)),
			value:       math.NaN(),
			expected:    []any{math.NaN()},
		},
		{
			name:        "positive infinity",
			expression:  "array(value)",
			source:      types.Float(64),
			destination: types.Array(types.Float(64)),
			value:       math.Inf(1),
			expected:    []any{math.Inf(1)},
		},
		{
			name:        "nil element",
			expression:  "array(value)",
			source:      types.String(),
			destination: types.Array(types.String()),
			value:       nil,
			err:         "«value» is not convertible to the «string» type",
		},
		{
			name:        "array elements satisfy pattern",
			expression:  "value",
			source:      types.Array(types.String()),
			destination: types.Array(types.String().WithPattern(regexp.MustCompile("^ok$"))),
			value:       []any{"ok", "ok"},
			expected:    []any{"ok", "ok"},
		},
		{
			name:        "pattern",
			expression:  "array(value)",
			source:      types.String(),
			destination: types.Array(types.String().WithPattern(regexp.MustCompile("^ok$"))),
			value:       "invalid",
			err:         "«value» is not convertible to the «string» type",
		},
		{
			name:        "real NaN",
			expression:  "array(value)",
			source:      types.Float(64),
			destination: types.Array(types.Float(64).Real()),
			value:       math.NaN(),
			err:         "«value» is not convertible to the «float(64)» type",
		},
		{
			name:        "real string NaN",
			expression:  "array(value)",
			source:      types.String(),
			destination: types.Array(types.Float(64).Real()),
			value:       "NaN",
			err:         "«value» is not convertible to the «float(64)» type",
		},
		{
			name:        "real narrowing overflow",
			expression:  "array(value)",
			source:      types.Float(64).Real(),
			destination: types.Array(types.Float(32).Real()),
			value:       math.MaxFloat64,
			err:         "«value» is not convertible to the «float(32)» type",
		},
		{
			name:        "JSON NaN",
			expression:  "array(value)",
			source:      types.Float(64),
			destination: types.Array(types.JSON()),
			value:       math.NaN(),
			err:         "«value» is not convertible to the «json» type",
		},
		{
			name:        "nested invalid date",
			expression:  "array(array(value))",
			source:      types.String(),
			destination: types.Array(types.Array(types.Date())),
			value:       "invalid",
			err:         "«value» is not convertible to the «date» type",
		},
		{
			name:        "too few",
			expression:  "array(value)",
			source:      types.Int(32),
			destination: types.Array(types.Int(32)).WithMinElements(2),
			value:       1,
			err:         "«array(value)» is not convertible to the «array(int(32))» type",
		},
		{
			name:        "too many",
			expression:  "array(value, 2)",
			source:      types.Int(32),
			destination: types.Array(types.Int(32)).WithMaxElements(1),
			value:       1,
			err:         "«array(value, 2)» is not convertible to the «array(int(32))» type",
		},
		{
			name:        "duplicate after conversion",
			expression:  "array(value, '1')",
			source:      types.String(),
			destination: types.Array(types.Int(32)).WithUnique(),
			value:       "01",
			err:         "«array(value, '1')» is not convertible to the «array(int(32))» type",
		},
		{
			name:        "required/create/object",
			expression:  "array(value)",
			source:      createRequiredObject,
			destination: types.Array(createRequiredObject),
			value:       map[string]any{},
			err:         "«array(value)» is not convertible to the «array(object)» type",
			purpose:     Create,
		},
		{
			name:        "required/create/map",
			expression:  "array(value)",
			source:      types.Map(createRequiredObject),
			destination: types.Array(types.Map(createRequiredObject)),
			value:       map[string]any{"key": map[string]any{}},
			err:         "«array(value)» is not convertible to the «array(map(object))» type",
			purpose:     Create,
		},
		{
			name:        "required/update/map",
			expression:  "array(value)",
			source:      types.Map(updateRequiredObject),
			destination: types.Array(types.Map(updateRequiredObject)),
			value:       map[string]any{"key": map[string]any{}},
			err:         "«array(value)» is not convertible to the «array(map(object))» type",
			purpose:     Update,
		},
		{
			name:        "valid array constraints",
			expression:  "array(value, '2')",
			source:      types.String(),
			destination: types.Array(types.Int(32)).WithMinElements(2).WithMaxElements(2).WithUnique(),
			value:       "01",
			expected:    []any{1, 2},
		},
		{
			name:        "required/create/present",
			expression:  "array(value)",
			source:      types.Map(createRequiredObject),
			destination: types.Array(types.Map(createRequiredObject)),
			value:       map[string]any{"key": map[string]any{"required": "ok"}},
			expected:    []any{map[string]any{"key": map[string]any{"required": "ok"}}},
			purpose:     Create,
		},
		{
			name:        "required/update/present",
			expression:  "array(value)",
			source:      types.Map(updateRequiredObject),
			destination: types.Array(types.Map(updateRequiredObject)),
			value:       map[string]any{"key": map[string]any{"required": "ok"}},
			expected:    []any{map[string]any{"key": map[string]any{"required": "ok"}}},
			purpose:     Update,
		},
		{
			name:        "create required during update",
			expression:  "array(value)",
			source:      types.Map(createRequiredObject),
			destination: types.Array(types.Map(createRequiredObject)),
			value:       map[string]any{"key": map[string]any{}},
			expected:    []any{map[string]any{"key": map[string]any{}}},
			purpose:     Update,
		},
		{
			name:        "update required during create",
			expression:  "array(value)",
			source:      types.Map(updateRequiredObject),
			destination: types.Array(types.Map(updateRequiredObject)),
			value:       map[string]any{"key": map[string]any{}},
			expected:    []any{map[string]any{"key": map[string]any{}}},
			purpose:     Create,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := types.Object([]types.Property{{Name: "value", Type: test.source, Nullable: true}})
			output := types.Object([]types.Property{{Name: "result", Type: test.destination}})
			mapping, err := New(map[string]string{"result": test.expression}, input, output, false)
			if err != nil {
				t.Fatalf("expected mapping creation to succeed, got %v", err)
			}
			got, err := mapping.Transform(map[string]any{"value": test.value}, test.purpose)
			if err != nil {
				if test.err == "" {
					t.Fatalf("expected transformation to succeed, got %v", err)
				}
				if _, ok := errors.AsType[ValidationError](err); !ok {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				want := test.err + " while mapping to «result»"
				if err.Error() != want {
					t.Fatalf("expected error %q, got %q", want, err.Error())
				}
				return
			}
			if test.err != "" {
				t.Fatalf("expected error %q, got nil", test.err)
			}
			expected := map[string]any{"result": test.expected}
			diff := cmp.Diff(expected, got, cmp.Comparer(func(a, b float64) bool {
				return a == b || math.IsNaN(a) && math.IsNaN(b)
			}))
			if diff != "" {
				t.Fatalf("expected no difference (-expected +got), got:\n%s", diff)
			}
		})
	}

}

// TestMappingNestedConversionErrors checks that constraint violations and
// conversion errors in nested values produce a ValidationError with a generic
// message for the containing type.
func TestMappingNestedConversionErrors(t *testing.T) {

	tests := []struct {
		name        string
		source      types.Type
		destination types.Type
		value       any
	}{
		{
			name:        "array element violates pattern",
			source:      types.Array(types.String()),
			destination: types.Array(types.String().WithPattern(regexp.MustCompile("^ok$"))),
			value:       []any{"ok", "invalid"},
		},
		{
			name:        "map value exceeds max bytes",
			source:      types.Map(types.String()),
			destination: types.Map(types.String().WithMaxBytes(2)),
			value:       map[string]any{"key": "invalid"},
		},
		{
			name:   "object property exceeds max length",
			source: types.Object([]types.Property{{Name: "key", Type: types.String()}}),
			destination: types.Object([]types.Property{
				{Name: "key", Type: types.String().WithMaxLength(2)},
			}),
			value: map[string]any{"key": "invalid"},
		},
		{
			name:   "deeply nested integer below minimum",
			source: types.JSON(),
			destination: types.Array(types.Map(types.Object([]types.Property{
				{Name: "key", Type: types.Int(32).WithIntRange(0, 5)},
			}))),
			value: json.Value(`[{"outer":{"key":-1}}]`),
		},
		{
			name:        "JSON array element exceeds maximum",
			source:      types.JSON(),
			destination: types.Array(types.Int(32).WithIntRange(0, 5)),
			value:       json.Value(`[6]`),
		},
		{
			name:        "JSON map value exceeds int64 range",
			source:      types.JSON(),
			destination: types.Map(types.Int(32)),
			value:       json.Value(`{"key":9223372036854775808}`),
		},
		{
			name:        "JSON object property contains invalid date",
			source:      types.JSON(),
			destination: types.Object([]types.Property{{Name: "key", Type: types.Date()}}),
			value:       json.Value(`{"key":"invalid"}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := types.Object([]types.Property{{Name: "value", Type: test.source}})
			output := types.Object([]types.Property{{Name: "result", Type: test.destination}})
			mapping, err := New(map[string]string{"result": "value"}, input, output, false)
			if err != nil {
				t.Fatalf("expected mapping creation to succeed, got %v", err)
			}
			_, err = mapping.Transform(map[string]any{"value": test.value}, None)
			if err != nil {
				if _, ok := err.(ValidationError); !ok {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				want := fmt.Sprintf("«value» is not convertible to the «%s» type while mapping to «result»",
					test.destination)
				if err.Error() != want {
					t.Fatalf("expected error %q, got %q", want, err.Error())
				}
				return
			}
			t.Fatal("expected ValidationError, got nil")
		})
	}

}

// TestMappingSemanticsValidation verifies that mappings apply semantic rules
// to values, including those nested in containers and object properties.
func TestMappingSemanticsValidation(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)

	tests := []struct {
		name     string
		typ      types.Type
		value    string
		expected string
		valid    bool
	}{
		{"current country", country, "US", "US", true},
		{"alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "USA", "USA", true},
		{"unknown alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ZZZ", "", false},
		{"unknown country", country, "ZZ", "", false},
		{"empty country", country, "", "", false},
		{"canonical phone", types.String().AsPhone(), "+390236618300", "+390236618300", true},
		{"formatted phone", types.String().AsPhone(), "+39 02-36618 300", "+390236618300", true},
		{"empty phone", types.String().AsPhone(), "", "", false},
		{"invalid phone", types.String().AsPhone(), "a (b)", "", false},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			for _, shape := range []string{"string", "array", "map", "nested", "object"} {

				t.Run(shape, func(t *testing.T) {

					typ := test.typ
					var value any = test.value
					var expected any = test.expected
					data := strconv.Quote(test.value)
					switch shape {
					case "array":
						typ = types.Array(typ)
						value = []any{test.value}
						expected = []any{test.expected}
						data = "[" + data + "]"
					case "map":
						typ = types.Map(typ)
						value = map[string]any{"home": test.value}
						expected = map[string]any{"home": test.expected}
						data = "{\"home\":" + data + "}"
					case "nested":
						typ = types.Array(types.Map(types.Array(typ)))
						value = []any{map[string]any{"home": []any{test.value}}}
						expected = []any{map[string]any{"home": []any{test.expected}}}
						data = "[{\"home\":[" + data + "]}]"
					case "object":
						typ = types.Object([]types.Property{{Name: "inner", Type: typ}})
						value = map[string]any{"inner": test.value}
						expected = map[string]any{"inner": test.expected}
						data = "{\"inner\":" + data + "}"
					}
					outSchema := types.Object([]types.Property{{Name: "target", Type: typ, CreateRequired: true}})
					for _, mode := range []string{"convert", "equal types", "json", "constant"} {

						if mode == "constant" && shape != "string" {
							continue
						}
						if mode == "equal types" && (!test.valid || test.value != test.expected) {
							continue
						}

						t.Run(mode, func(t *testing.T) {

							inputType, inputValue := typ, value
							expr := "source"
							switch mode {
							case "convert":
								inputType = types.String()
								switch shape {
								case "array":
									inputType = types.Array(inputType)
								case "map":
									inputType = types.Map(inputType)
								case "nested":
									inputType = types.Array(types.Map(types.Array(inputType)))
								case "object":
									inputType = types.Object([]types.Property{{Name: "inner", Type: inputType}})
								}
							case "equal types":
								inputType = typ
							case "json":
								inputType, inputValue = types.JSON(), json.Value(data)
							case "constant":
								expr = strconv.Quote(test.value)
							}
							inSchema := types.Object([]types.Property{{Name: "source", Type: inputType}})
							mapping, err := New(map[string]string{"target": expr}, inSchema, outSchema, false)
							if err != nil {
								if test.valid {
									t.Fatalf("expected mapping creation to succeed, got %v", err)
								}
								if mode != "constant" {
									t.Fatalf("expected mapping creation to succeed for %s mode, got %v", mode, err)
								}
								want := " is not a valid phone number"
								if test.typ.Semantics() == types.CountrySemantics {
									want = " is not a 2-letter country code"
									if test.typ.CountryFormat() == types.ISO3166Alpha3 {
										want = " is not a 3-letter country code"
									}
								}
								want = strconv.Quote(test.value) + want
								if !strings.Contains(err.Error(), want) {
									t.Fatalf("expected error containing %q, got %v", want, err)
								}
								return
							}
							got, err := mapping.Transform(map[string]any{"source": inputValue}, Create)
							if err != nil {
								if test.valid {
									t.Fatalf("expected transformation to succeed, got %v", err)
								}
								if _, ok := errors.AsType[ValidationError](err); !ok {
									t.Fatalf("expected ValidationError, got %T: %v", err, err)
								}
								return
							}
							if !test.valid {
								t.Fatalf("expected invalid value to be rejected, got %#v", got["target"])
							}
							if !reflect.DeepEqual(got["target"], expected) {
								t.Fatalf("expected value %#v, got %#v", expected, got["target"])
							}

						})

					}

				})

			}

		})

	}

}

func Test_InOutPaths(t *testing.T) {

	inSchema := types.Object([]types.Property{
		{Name: "a", Type: types.String()},
		{Name: "b", Type: types.Map(types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		}))},
		{Name: "c", Type: types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		})},
		{Name: "d", Type: types.JSON()},
	})

	outSchema := types.Object([]types.Property{
		{Name: "foo", Type: types.String()},
		{Name: "boo", Type: types.Int(32).Unsigned()},
	})

	tests := []struct {
		expressions map[string]string
		inPaths     []string
		outPaths    []string
	}{
		{
			expressions: map[string]string{
				"foo": "'a'",
				"boo": "5",
			},
			inPaths:  []string{},
			outPaths: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "'a'",
				"boo": "b.k.y",
			},
			inPaths:  []string{"b.y"},
			outPaths: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "b.k.x a b.k.x",
				"boo": "a b.k.x a",
			},
			inPaths:  []string{"a", "b.x"},
			outPaths: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "a '-' d.p.s? ' ' b['k'].z '*' c.z c.x",
			},
			inPaths:  []string{"a", "b.z", "c.x", "c.z", "d"},
			outPaths: []string{"foo"},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			mapping, err := New(test.expressions, inSchema, outSchema, false)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got := mapping.InPaths()
			if got == nil || !slices.Equal(test.inPaths, got) {
				t.Fatalf("expected input properties %#v, got %#v", test.inPaths, got)
			}
			got = mapping.OutPaths()
			if got == nil || !slices.Equal(test.outPaths, got) {
				t.Fatalf("expected output properties %#v, got %#v", test.outPaths, got)
			}
		})
	}

}

func Test_Transform(t *testing.T) {

	inSchema := types.Object([]types.Property{
		{Name: "a", Type: types.String()},
		{Name: "b", Type: types.Map(types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		}))},
		{Name: "c", Type: types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		})},
		{Name: "d", Type: types.JSON()},
		{Name: "e", Type: types.JSON(), Nullable: true},
		{Name: "emptyObject", Type: types.Object([]types.Property{{Name: "foo", Type: types.String(), ReadOptional: true}})},
	})

	outSchema := types.Object([]types.Property{
		{Name: "A", Type: types.String()},
		{Name: "B", Type: types.String(), Nullable: true},
		{Name: "C", Type: types.JSON()},
		{Name: "D", Type: types.JSON(), Nullable: true},
		{Name: "E", Type: types.Int(32)},
		{Name: "F", Type: types.String(), CreateRequired: true},
		{Name: "G", Type: types.String(), UpdateRequired: true},
		{Name: "H", Type: types.Int(64).Unsigned()},
		{Name: "I", Type: types.Float(64)},
		{Name: "J", Type: types.Decimal(5, 2)},
	})

	tests := []struct {
		name        string
		expressions map[string]string
		attributes  map[string]any
		purpose     Purpose
		expected    map[string]any
		err         error
	}{

		{
			name:        `An empty string -> an empty string`,
			expressions: map[string]string{"A": "''"},
			expected:    map[string]any{"A": ""},
		},
		{
			name:        `null assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "null"},
			expected:    map[string]any{},
		},
		{
			name:        `A property without a value -> no properties`,
			expressions: map[string]string{"A": "a"},
			expected:    map[string]any{},
		},
		{
			name:        `A nil property assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A property with a non-empty string -> the non-empty string`,
			expressions: map[string]string{"A": "a"},
			attributes:  map[string]any{"a": "boo"},
			expected:    map[string]any{"A": "boo"},
		},
		{
			name:        `null assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "null"},
			expected:    map[string]any{"B": nil},
		},
		{
			name:        `A nil property assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{"B": nil},
		},

		{
			name:        `An empty string assigned to a json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "''"},
			expected:    map[string]any{"C": json.Value(`""`)},
		},
		{
			name:        `A non-empty string assigned to a json property -> the string as JSON`,
			expressions: map[string]string{"C": "'boo'"},
			expected:    map[string]any{"C": json.Value(`"boo"`)},
		},
		{
			name:        `null assigned to a non-nullable json property -> the string as JSON`,
			expressions: map[string]string{"C": "null"},
			expected:    map[string]any{}, // TODO(marco): review.
		},
		{
			name:        `A property without a value assigned to a non-nullable json property -> no properties`,
			expressions: map[string]string{"C": "a"},
			expected:    map[string]any{},
		},
		{
			name:        `A property with a nil value assigned to a non-nullable json property -> nil as JSON`,
			expressions: map[string]string{"C": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A property without a value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', a, 'h', 5)"},
			expected:    map[string]any{"C": json.Value(`{"h":5}`)},
		},
		{
			name:        `An empty object in array() -> an empty JSON object`,
			expressions: map[string]string{"C": "array(emptyObject)"},
			attributes:  map[string]any{"emptyObject": map[string]any{}},
			expected:    map[string]any{"C": json.Value(`[{}]`)},
		},
		{
			name:        `An empty object in map() -> an empty JSON object`,
			expressions: map[string]string{"C": "map('k', emptyObject)"},
			attributes:  map[string]any{"emptyObject": map[string]any{}},
			expected:    map[string]any{"C": json.Value(`{"k":{}}`)},
		},
		{
			name:        `A property with a nil value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', c.z, 'h', 5)"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{"C": json.Value(`{"h":5}`)},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			attributes:  map[string]any{"a": ""},
			expected:    map[string]any{"C": json.Value(`""`)},
		},

		{
			name:        `An empty string assigned to a nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"D": "''"},
			expected:    map[string]any{"D": json.Value(`""`)},
		},
		{
			name:        `A non-empty string assigned to a nullable json property -> the string as JSON`,
			expressions: map[string]string{"D": "'boo'"},
			expected:    map[string]any{"D": json.Value(`"boo"`)},
		},
		{
			name:        `null assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "null"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property without a value assigned to a non-nullable json property -> nil`,
			expressions: map[string]string{"D": "a"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with a nil value assigned to a non-nullable json property -> nil`,
			expressions: map[string]string{"D": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			attributes:  map[string]any{"a": ""},
			expected:    map[string]any{"C": json.Value(`""`)},
		},

		{
			name:        `A json property without a value assigned to a non-nullable json property -> no properties`,
			expressions: map[string]string{"C": "d"},
			expected:    map[string]any{},
		},
		{
			name:        `A json property with a nil value assigned to a non-nullable json property -> no properties`,
			expressions: map[string]string{"C": "e"},
			attributes:  map[string]any{"e": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A json property with a JSON null value assigned to a non-nullable json property -> no properties`,
			expressions: map[string]string{"C": "d"},
			attributes:  map[string]any{"d": json.Value(`null`)},
			expected:    map[string]any{"C": json.Value(`null`)},
		},
		{
			name:        `A json property without a value assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "e"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A json property with a nil value assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "e"},
			attributes:  map[string]any{"e": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A json property with a JSON null value assigned to a nullable json property -> no properties`,
			expressions: map[string]string{"D": "e"},
			attributes:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{"D": json.Value(`null`)},
		},

		{
			name:        `A json property with a JSON null value assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "e"},
			attributes:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{},
		},
		{
			name:        `A json property with a JSON null value assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "e"},
			attributes:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{"B": nil},
		},

		{
			name:        `"a ' ' c.x ': ' 5.45"`,
			expressions: map[string]string{"A": "a ' ' c.x ': ' 5.45"},
			attributes:  map[string]any{"a": "foo", "c": map[string]any{"x": "boo"}},
			expected:    map[string]any{"A": "foo boo: 5.45"},
		},
		{
			name:        `"len(a)"`,
			expressions: map[string]string{"E": "len(a)"},
			attributes:  map[string]any{"a": "foo"},
			expected:    map[string]any{"E": 3},
		},
		{
			name:        `Spurious properties`,
			expressions: map[string]string{"A": "a"},
			attributes:  map[string]any{"a": "foo", "b": "boo", "c": 24},
			expected:    map[string]any{"A": "foo"},
		},

		{
			name:        `null assigned to a non nullable create required property -> error`,
			expressions: map[string]string{"F": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			purpose:     Create,
			err:         ValidationError{msg: `«c.z» is null but it is required for creation while mapping to «F»`},
		},
		{
			name:        `null assigned to a non nullable update required property -> error`,
			expressions: map[string]string{"G": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			purpose:     Update,
			err:         ValidationError{msg: `«c.z» is null but it is required for update while mapping to «G»`},
		},
		{
			name:        `A string exceeding the int64 range assigned to an int property -> error`,
			expressions: map[string]string{"E": "a"},
			attributes:  map[string]any{"a": "9223372036854775808"},
			err:         ValidationError{msg: `number «a» is too small or too large for the «int(32)» type while mapping to «E»`},
		},
		{
			name:        `A JSON number exceeding the int64 range assigned to an int property -> error`,
			expressions: map[string]string{"E": "d"},
			attributes:  map[string]any{"d": json.Value(`9223372036854775808`)},
			err:         ValidationError{msg: `number «d» is too small or too large for the «int(32)» type while mapping to «E»`},
		},
		{
			name:        `A string exceeding the uint64 range assigned to an unsigned int property -> error`,
			expressions: map[string]string{"H": "a"},
			attributes:  map[string]any{"a": "18446744073709551616"},
			err:         ValidationError{msg: `number «a» is too small or too large for the «unsigned int(64)» type while mapping to «H»`},
		},
		{
			name:        `A string exceeding the float64 range assigned to a float property -> error`,
			expressions: map[string]string{"I": "a"},
			attributes:  map[string]any{"a": "1e400"},
			err:         ValidationError{msg: `number «a» is too small or too large for the «float(64)» type while mapping to «I»`},
		},
		{
			name:        `A JSON number exceeding the precision assigned to a decimal property -> error`,
			expressions: map[string]string{"J": "d"},
			attributes:  map[string]any{"d": json.Value(`1000`)},
			err:         ValidationError{msg: `number «d» is too small or too large for the «decimal(5,2)» type while mapping to «J»`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping, err := New(test.expressions, inSchema, outSchema, false)
			if err != nil {
				t.Fatalf("unexpected error calling New: %q (%T)", err, err)
			}
			got, err := mapping.Transform(test.attributes, test.purpose)
			if err != nil {
				if test.err == nil {
					t.Fatalf("unexpected error: %q (%T)", err, err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("expected error %q (%T), got %q (%T)", test.err, test.err, err, err)
				}
				return
			}
			if !cmp.Equal(test.expected, got) {
				t.Fatalf("unexpected result from Transform:\n\n%s\n", cmp.Diff(test.expected, got))
			}
		})
	}

}

func Test_inPlace(t *testing.T) {

	clone := func(v map[string]any, t types.Type) map[string]any {
		j, _ := types.Marshal(v, t)
		v, _ = types.Decode[map[string]any](bytes.NewReader(j), t)
		return v
	}

	tests := []struct {
		inType   types.Type
		outType  types.Type
		value    any
		expected any
	}{
		{
			inType:   types.Array(types.Int(32)),
			outType:  types.Array(types.String()),
			value:    []any{1, 2, 3},
			expected: []any{"1", "2", "3"},
		},
		{
			inType:   types.Map(types.Boolean()),
			outType:  types.Map(types.String()),
			value:    map[string]any{"a": true, "b": false},
			expected: map[string]any{"a": "true", "b": "false"},
		},
		{
			inType:   types.Object([]types.Property{{Name: "a", Type: types.Int(16)}, {Name: "b", Type: types.UUID()}}),
			outType:  types.Object([]types.Property{{Name: "a", Type: types.String()}}),
			value:    map[string]any{"a": 22, "b": "90620928-691e-4aab-9b5c-ce202cad156f"},
			expected: map[string]any{"a": "22"},
		},
		{
			inType:   types.Array(types.Map(types.Int(32))),
			outType:  types.Array(types.Map(types.Float(64))),
			value:    []any{map[string]any{"x": 12, "y": -68}, map[string]any{"a": 5, "b": 8032}},
			expected: []any{map[string]any{"x": 12.0, "y": -68.0}, map[string]any{"a": 5.0, "b": 8032.0}},
		},
		{
			inType:   types.Map(types.Array(types.Float(64))),
			outType:  types.Map(types.Array(types.String())),
			value:    map[string]any{"foo": []any{4.67, -1.02}},
			expected: map[string]any{"foo": []any{"4.67", "-1.02"}},
		},
	}
	expressions := map[string]string{"out": "in"}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			inSchema := types.Object([]types.Property{{Name: "in", Type: test.inType}})
			outSchema := types.Object([]types.Property{{Name: "out", Type: test.outType}})
			v := map[string]any{"in": test.value}
			z := clone(v, inSchema)
			inPlace := false
			for {
				mapping, err := New(expressions, inSchema, outSchema, inPlace)
				if err != nil {
					t.Fatalf("unexpected error calling New: %q (%T)", err, err)
				}
				got, err := mapping.Transform(z, None)
				if err != nil {
					t.Fatalf("unexpected error calling Transform: %q (%T)", err, err)
				}
				if !reflect.DeepEqual(map[string]any{"out": test.expected}, got) {
					t.Fatalf("expected %#v, got %#v", test.expected, got)
				}
				if inPlace {
					if !reflect.DeepEqual(z["in"], got["out"]) {
						t.Fatal("expected changed value, got unchanged")
					}
					return
				}
				if !reflect.DeepEqual(v, z) {
					t.Fatal("expected unchanged value, got changed")
				}
				inPlace = true
			}
		})
	}
}

func Test_sortMappingExpressions(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		wantErr bool
	}{
		{
			name:    "no conflict",
			paths:   []string{"a", "b.c", "b.d", "c.d.e"},
			wantErr: false,
		},
		{
			name:    "duplicate",
			paths:   []string{"a", "b", "b", "c"},
			wantErr: true,
		},
		{
			name:    "logical prefix",
			paths:   []string{"foo", "foo.bar"},
			wantErr: true,
		},
		{
			name:    "logical prefix reversed order",
			paths:   []string{"foo.bar", "foo"},
			wantErr: true,
		},
		{
			name:    "no conflict with similar prefix",
			paths:   []string{"value", "value_currency"},
			wantErr: false,
		},
		{
			name:    "similar but not prefix",
			paths:   []string{"foo.bar", "foo.barb"},
			wantErr: false,
		},
		{
			name:    "single element",
			paths:   []string{"only"},
			wantErr: false,
		},
		{
			name:    "multiple prefixes",
			paths:   []string{"a", "a.b", "a.b.c", "b", "b.c"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exprs []mappingExpr
			for _, p := range tt.paths {
				exprs = append(exprs, mappingExpr{path: p})
			}
			err := sortMappingExpressions(exprs)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("unexpected error %q", err)
				}
			} else if tt.wantErr {
				t.Error("expected error, got no error")
			}
		})
	}
}

func Test_storeValue(t *testing.T) {
	tests := []struct {
		value    map[string]any
		path     string
		v        any
		expected map[string]any
	}{
		{
			value:    map[string]any{},
			path:     "email",
			v:        "test@example.com",
			expected: map[string]any{"email": "test@example.com"},
		},
		{
			value:    map[string]any{},
			path:     "user.email",
			v:        "test@example.com",
			expected: map[string]any{"user": map[string]any{"email": "test@example.com"}},
		},
		{
			value:    map[string]any{"user": map[string]any{"name": "Mike"}},
			path:     "user.email",
			v:        "test@example.com",
			expected: map[string]any{"user": map[string]any{"name": "Mike", "email": "test@example.com"}},
		},
		{
			value:    map[string]any{"user": map[string]any{"address": map[string]any{"city": "Milan"}}},
			path:     "user.address.zip",
			v:        "20122",
			expected: map[string]any{"user": map[string]any{"address": map[string]any{"city": "Milan", "zip": "20122"}}},
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			storeValue(test.value, test.path, test.v)
			if !reflect.DeepEqual(test.value, test.expected) {
				t.Fatalf("expected %#v, got %#v", test.expected, test.value)
			}
		})
	}
}
