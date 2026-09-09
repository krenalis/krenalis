// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/go-cmp/cmp"
)

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
			mapping, err := New(test.expressions, inSchema, outSchema, false, nil)
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
	})

	outSchema := types.Object([]types.Property{
		{Name: "A", Type: types.String()},
		{Name: "B", Type: types.String(), Nullable: true},
		{Name: "C", Type: types.JSON()},
		{Name: "D", Type: types.JSON(), Nullable: true},
		{Name: "E", Type: types.Int(32)},
		{Name: "F", Type: types.String(), CreateRequired: true},
		{Name: "G", Type: types.String(), UpdateRequired: true},
	})

	tests := []struct {
		name        string
		expressions map[string]string
		layouts     *state.TimeLayouts
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
			attributes:  map[string]any{"c": map[string]any{"z": nil}},
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
			attributes:  map[string]any{"c": map[string]any{"z": nil}},
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
			name:        `null assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "null"},
			expected:    map[string]any{"C": json.Value("null")},
		},
		{
			name:        `A property without a value assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "a"},
			expected:    map[string]any{"C": json.Value("null")},
		},
		{
			name:        `A property with a nil value assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "c.z"},
			attributes:  map[string]any{"c": map[string]any{"z": nil}},
			expected:    map[string]any{"C": json.Value("null")},
		},
		{
			name:        `A property without a value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', a, 'h', 5)"},
			expected:    map[string]any{"C": json.Value(`{"h":5}`)},
		},
		{
			name:        `A property with a nil value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', c.z, 'h', 5)"},
			attributes:  map[string]any{"c": map[string]any{"z": nil}},
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
			name:        `A property without a value assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "a"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with a nil value assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "c.z"},
			attributes:  map[string]any{"c": map[string]any{"z": nil}},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			attributes:  map[string]any{"a": ""},
			expected:    map[string]any{"C": json.Value(`""`)},
		},

		{
			name:        `A json property without a value assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "d"},
			expected:    map[string]any{"C": json.Value("null")},
		},
		{
			name:        `A json property with a nil value assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "e"},
			attributes:  map[string]any{"e": nil},
			expected:    map[string]any{"C": json.Value("null")},
		},
		{
			name:        `A json property with a JSON null value assigned to a non-nullable json property -> JSON null`,
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
			name:        `A json property with a JSON null value assigned to a nullable json property -> JSON null`,
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
			attributes:  map[string]any{"c": map[string]any{"z": nil}},
			purpose:     Create,
			err:         ValidationError{msg: `«c.z» is null but it is required for creation while mapping to «F»`},
		},
		{
			name:        `null assigned to a non nullable update required property -> error`,
			expressions: map[string]string{"G": "c.z"},
			attributes:  map[string]any{"c": map[string]any{"z": nil}},
			purpose:     Update,
			err:         ValidationError{msg: `«c.z» is null but it is required for update while mapping to «G»`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping, err := New(test.expressions, inSchema, outSchema, false, test.layouts)
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

// Test_inPlace checks output values with either allocation policy and preserves
// input when inPlace is false.
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
				mapping, err := New(expressions, inSchema, outSchema, inPlace, nil)
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

// TestMappingJSONNull checks null results assigned to optional, nullable,
// required, and nested JSON properties.
func TestMappingJSONNull(t *testing.T) {

	inSchema := types.Object([]types.Property{
		{Name: "value", Type: types.String(), Nullable: true, ReadOptional: true},
		{Name: "document", Type: types.JSON(), Nullable: true, ReadOptional: true},
		{Name: "parent", Nullable: true, ReadOptional: true, Type: types.Object([]types.Property{
			{Name: "value", Type: types.String(), Nullable: true, ReadOptional: true},
		})},
	})
	properties := []types.Property{
		{Name: "optional", Type: types.JSON()},
		{Name: "nullable", Type: types.JSON(), Nullable: true},
		{Name: "required", Type: types.JSON(), CreateRequired: true, UpdateRequired: true},
		{Name: "nullableRequired", Type: types.JSON(), Nullable: true, CreateRequired: true, UpdateRequired: true},
		{Name: "unmapped", Type: types.JSON()},
	}
	outSchema := types.Object(append(properties, types.Property{Name: "parent", Type: types.Object(properties)}))
	tests := []struct {
		name, source string
		attributes   map[string]any
		nullableWant any
	}{
		{"literal", "null", nil, nil},
		{"missing string", "value", nil, nil},
		{"nil string", "value", map[string]any{"value": nil}, nil},
		{"missing JSON", "document", nil, nil},
		{"nil JSON", "document", map[string]any{"document": nil}, nil},
		{"JSON null", "document", map[string]any{"document": json.Value("null")}, json.Value("null")},
		{"missing ancestor", "parent.value", nil, nil},
		{"nil ancestor", "parent.value", map[string]any{"parent": nil}, nil},
		{"nil descendant", "parent.value", map[string]any{"parent": map[string]any{"value": nil}}, nil},
		{"missing JSON key", "document.value", map[string]any{"document": json.Value("{}")}, nil},
		{"if", "if(true, value, document)", nil, nil},
		{"coalesce", "coalesce(value, document)", nil, nil},
		{"json_parse nil", "json_parse(null)", nil, nil},
		{"json_parse null", "json_parse('null')", nil, json.Value("null")},
	}

	for _, test := range tests {
		for _, purpose := range []Purpose{None, Create, Update} {
			for _, inPlace := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/purpose=%d/inPlace=%t", test.name, purpose, inPlace), func(t *testing.T) {

					expressions := map[string]string{}
					for _, prefix := range []string{"", "parent."} {
						for _, name := range []string{"optional", "nullable", "required", "nullableRequired"} {
							expressions[prefix+name] = test.source
						}
					}
					mapping, err := New(expressions, inSchema, outSchema, inPlace, nil)
					if err != nil {
						t.Fatal(err)
					}

					got, err := mapping.Transform(test.attributes, purpose)
					if err != nil {
						t.Fatal(err)
					}
					want := map[string]any{
						"optional": json.Value("null"), "nullable": test.nullableWant,
						"required": json.Value("null"), "nullableRequired": test.nullableWant,
						"parent": map[string]any{
							"optional": json.Value("null"), "nullable": test.nullableWant,
							"required": json.Value("null"), "nullableRequired": test.nullableWant,
						},
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("got %#v, want %#v", got, want)
					}

				})
			}
		}
	}

}

// TestNewErrorOrder checks that destination paths determine which compilation
// error is returned first.
func TestNewErrorOrder(t *testing.T) {

	outSchema := types.Object([]types.Property{
		{Name: "b", Type: types.String()}, {Name: "a", Type: types.String()},
	})
	tests := []struct {
		name        string
		expressions map[string]string
	}{
		{"compilation errors", map[string]string{"b": "upper()", "a": "lower()"}},
		{"lookup and compilation errors", map[string]string{"missing": "''", "a": "lower()"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for range 100 {
				_, err := New(test.expressions, types.Type{}, outSchema, false, nil)
				if err != nil {
					want := "'lower' function requires a single argument"
					if err.Error() != want {
						t.Fatalf("got %q, want %q", err, want)
					}
					continue
				}
				t.Fatal("expected a compilation error")
			}
		})
	}

}

// TestMappingRequiredAncestors checks that descendants are required only in
// present, non-null objects.
func TestMappingRequiredAncestors(t *testing.T) {

	tests := []struct {
		name           string
		expressions    map[string]string
		parentRequired bool
		xNullable      bool
		want           map[string]any
		wantError      bool
	}{
		{"absent optional parent", map[string]string{"other": "'ok'"}, false, false,
			map[string]any{"other": "ok"}, false},
		{"null optional parent", map[string]string{"parent": "null"}, false, false,
			map[string]any{"parent": nil}, false},
		{"absent required parent", map[string]string{"other": "'ok'"}, true, false,
			map[string]any{"other": "ok"}, true},
		{"null required nullable parent", map[string]string{"parent": "null"}, true, false,
			map[string]any{"parent": nil}, false},
		{"omitted null leaf", map[string]string{"parent.x": "null"}, false, false,
			map[string]any{}, false},
		{"stored null leaf", map[string]string{"parent.x": "null"}, false, true,
			map[string]any{"parent": map[string]any{"x": nil}}, true},
	}
	for _, test := range tests {

		for _, purpose := range []Purpose{None, Create, Update} {

			for _, inPlace := range []bool{false, true} {

				t.Run(fmt.Sprintf("%s/purpose=%d/inPlace=%t", test.name, purpose, inPlace), func(t *testing.T) {

					parent := types.Object([]types.Property{
						{Name: "x", Type: types.String(), Nullable: test.xNullable},
						{Name: "missing", Type: types.String(), CreateRequired: true, UpdateRequired: true},
					})
					schema := types.Object([]types.Property{
						{
							Name: "parent", Type: parent, Nullable: true,
							CreateRequired: test.parentRequired, UpdateRequired: test.parentRequired,
						},
						{Name: "other", Type: types.String()},
					})
					mapping, err := New(test.expressions, types.Type{}, schema, inPlace, nil)
					if err != nil {
						t.Fatal(err)
					}

					got, err := mapping.Transform(nil, purpose)
					wantError := test.wantError && purpose != None
					if err != nil {
						if !wantError {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("got %T (%v), want ValidationError", err, err)
						}
						return
					}
					if wantError {
						t.Fatalf("got %#v, want a missing property error", got)
					}
					if !reflect.DeepEqual(got, test.want) {
						t.Fatalf("got %#v, want %#v", got, test.want)
					}

				})

			}

		}

	}

}

// TestMappingRequiredPaths validates root and nested required properties after
// individual field assignments.
func TestMappingRequiredPaths(t *testing.T) {

	inSchema := types.Object([]types.Property{{Name: "value", Type: types.String()}})
	for _, requiredOn := range []Purpose{Create, Update} {

		for _, prefix := range []string{"", "parent.", "parent.child."} {

			for _, complete := range []bool{false, true} {

				for _, purpose := range []Purpose{None, Create, Update} {

					for _, inPlace := range []bool{false, true} {

						name := fmt.Sprintf("%s/requiredOn=%d/complete=%t/purpose=%d/inPlace=%t",
							prefix, requiredOn, complete, purpose, inPlace)
						t.Run(name, func(t *testing.T) {

							outSchema := types.Object([]types.Property{
								{Name: "x", Type: types.String()},
								{
									Name: "missing", Type: types.String(),
									CreateRequired: requiredOn == Create, UpdateRequired: requiredOn == Update,
								},
							})
							expressions := map[string]string{prefix + "x": "value"}
							want := map[string]any{"x": "hello"}
							if complete {
								expressions[prefix+"missing"] = "'present'"
								want["missing"] = "present"
							}
							if prefix == "parent.child." {
								outSchema = types.Object([]types.Property{{Name: "child", Type: outSchema}})
								want = map[string]any{"child": want}
							}
							if prefix != "" {
								outSchema = types.Object([]types.Property{{Name: "parent", Type: outSchema}})
								want = map[string]any{"parent": want}
							}

							mapping, err := New(expressions, inSchema, outSchema, inPlace, nil)
							if err != nil {
								t.Fatal(err)
							}
							got, err := mapping.Transform(map[string]any{"value": "hello"}, purpose)
							wantError := purpose == requiredOn && !complete
							if err != nil {
								if !wantError {
									t.Fatal(err)
								}
								if _, ok := errors.AsType[ValidationError](err); !ok {
									t.Fatalf("got %T (%v), want ValidationError", err, err)
								}
								reason := "creation"
								if purpose == Update {
									reason = "update"
								}
								message := fmt.Sprintf("«%smissing» is missing but it is required for %s",
									prefix, reason)
								if err.Error() != message {
									t.Fatalf("got %q, want %q", err, message)
								}
								return
							}
							if wantError {
								t.Fatalf("got %#v, want a missing property error", got)
							}
							if !reflect.DeepEqual(got, want) {
								t.Fatalf("got %#v, want %#v", got, want)
							}

						})

					}

				}

			}

		}

	}

}

// TestMappingRequiredTimeFormatting checks structural validation after temporal
// values have been formatted.
func TestMappingRequiredTimeFormatting(t *testing.T) {

	object := types.Object([]types.Property{
		{Name: "at", Type: types.DateTime(), CreateRequired: true, UpdateRequired: true},
	})
	parent := types.Object([]types.Property{
		{Name: "at", Type: types.DateTime(), CreateRequired: true, UpdateRequired: true},
		{Name: "dates", Type: types.Array(types.Date()), CreateRequired: true, UpdateRequired: true},
		{Name: "objects", Type: types.Map(object), CreateRequired: true, UpdateRequired: true},
	})
	outSchema := types.Object([]types.Property{{Name: "parent", Type: parent}})
	expressions := map[string]string{"parent.at": "at", "parent.dates": "dates", "parent.objects": "objects"}
	at := time.Date(2026, 9, 8, 12, 34, 56, 125000000, time.UTC)
	date := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	want := map[string]any{"parent": map[string]any{
		"at": at.Unix(), "dates": []any{"2026-09-08"},
		"objects": map[string]any{"key": map[string]any{"at": at.Unix()}},
	}}
	for _, purpose := range []Purpose{None, Create, Update} {

		for _, inPlace := range []bool{false, true} {

			t.Run(fmt.Sprintf("purpose=%d/inPlace=%t", purpose, inPlace), func(t *testing.T) {

				layouts := &state.TimeLayouts{DateTime: "unix", Date: time.DateOnly}
				mapping, err := New(expressions, parent, outSchema, inPlace, layouts)
				if err != nil {
					t.Fatal(err)
				}

				input := map[string]any{
					"at": at, "dates": []any{date}, "objects": map[string]any{"key": map[string]any{"at": at}},
				}
				got, err := mapping.Transform(input, purpose)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}

			})

		}

	}

}
