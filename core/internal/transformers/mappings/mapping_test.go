//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"bytes"
	"reflect"
	"slices"
	"testing"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
)

func Test_InOutPaths(t *testing.T) {

	inSchema := types.Object([]types.Property{
		{Name: "a", Type: types.Text()},
		{Name: "b", Type: types.Map(types.Object([]types.Property{
			{Name: "x", Type: types.Text()},
			{Name: "y", Type: types.Uint(32)},
			{Name: "z", Type: types.Text(), Nullable: true},
		}))},
		{Name: "c", Type: types.Object([]types.Property{
			{Name: "x", Type: types.Text()},
			{Name: "y", Type: types.Uint(32)},
			{Name: "z", Type: types.Text(), Nullable: true},
		})},
		{Name: "d", Type: types.JSON()},
	})

	outSchema := types.Object([]types.Property{
		{Name: "foo", Type: types.Text()},
		{Name: "boo", Type: types.Uint(32)},
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
		{Name: "a", Type: types.Text()},
		{Name: "b", Type: types.Map(types.Object([]types.Property{
			{Name: "x", Type: types.Text()},
			{Name: "y", Type: types.Uint(32)},
			{Name: "z", Type: types.Text(), Nullable: true},
		}))},
		{Name: "c", Type: types.Object([]types.Property{
			{Name: "x", Type: types.Text()},
			{Name: "y", Type: types.Uint(32)},
			{Name: "z", Type: types.Text(), Nullable: true},
		})},
		{Name: "d", Type: types.JSON()},
		{Name: "e", Type: types.JSON(), Nullable: true},
	})

	outSchema := types.Object([]types.Property{
		{Name: "A", Type: types.Text()},
		{Name: "B", Type: types.Text(), Nullable: true},
		{Name: "C", Type: types.JSON()},
		{Name: "D", Type: types.JSON(), Nullable: true},
		{Name: "E", Type: types.Int(32)},
		{Name: "F", Type: types.Text(), CreateRequired: true},
		{Name: "G", Type: types.Text(), UpdateRequired: true},
	})

	tests := []struct {
		name        string
		expressions map[string]string
		layouts     *state.TimeLayouts
		properties  map[string]any
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
			properties:  map[string]any{"c.z": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A property with a non-empty string -> the non-empty string`,
			expressions: map[string]string{"A": "a"},
			properties:  map[string]any{"a": "boo"},
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
			properties:  map[string]any{"c.z": nil},
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
			properties:  map[string]any{"c.z": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A property without a value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', a, 'h', 5)"},
			expected:    map[string]any{"C": json.Value(`{"h":5}`)},
		},
		{
			name:        `A property with a nil value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', c.z, 'h', 5)"},
			properties:  map[string]any{"c.z": nil},
			expected:    map[string]any{"C": json.Value(`{"h":5}`)},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			properties:  map[string]any{"a": ""},
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
			properties:  map[string]any{"c.z": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			properties:  map[string]any{"a": ""},
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
			properties:  map[string]any{"e": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A json property with a JSON null value assigned to a non-nullable json property -> no properties`,
			expressions: map[string]string{"C": "d"},
			properties:  map[string]any{"d": json.Value(`null`)},
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
			properties:  map[string]any{"e": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A json property with a JSON null value assigned to a nullable json property -> no properties`,
			expressions: map[string]string{"D": "e"},
			properties:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{"D": json.Value(`null`)},
		},

		{
			name:        `A json property with a JSON null value assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "e"},
			properties:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{},
		},
		{
			name:        `A json property with a JSON null value assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "e"},
			properties:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{"B": nil},
		},

		{
			name:        `"a ' ' c.x ': ' 5.45"`,
			expressions: map[string]string{"A": "a ' ' c.x ': ' 5.45"},
			properties:  map[string]any{"a": "foo", "c": map[string]any{"x": "boo"}},
			expected:    map[string]any{"A": "foo boo: 5.45"},
		},
		{
			name:        `"len(a)"`,
			expressions: map[string]string{"E": "len(a)"},
			properties:  map[string]any{"a": "foo"},
			expected:    map[string]any{"E": 3},
		},
		{
			name:        `Spurious properties`,
			expressions: map[string]string{"A": "a"},
			properties:  map[string]any{"a": "foo", "b": "boo", "c": 24},
			expected:    map[string]any{"A": "foo"},
		},

		{
			name:        `null assigned to a non nullable create required property -> error`,
			expressions: map[string]string{"F": "c.z"},
			properties:  map[string]any{"c.z": nil},
			purpose:     Create,
			err:         ValidationError{msg: `«c.z» is null but it is required for creation while mapping to «F»`},
		},
		{
			name:        `null assigned to a non nullable update required property -> error`,
			expressions: map[string]string{"G": "c.z"},
			properties:  map[string]any{"c.z": nil},
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
			got, err := mapping.Transform(test.properties, test.purpose)
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
			outType:  types.Array(types.Text()),
			value:    []any{1, 2, 3},
			expected: []any{"1", "2", "3"},
		},
		{
			inType:   types.Map(types.Boolean()),
			outType:  types.Map(types.Text()),
			value:    map[string]any{"a": true, "b": false},
			expected: map[string]any{"a": "true", "b": "false"},
		},
		{
			inType:   types.Object([]types.Property{{Name: "a", Type: types.Int(16)}, {Name: "b", Type: types.UUID()}}),
			outType:  types.Object([]types.Property{{Name: "a", Type: types.Text()}}),
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
			outType:  types.Map(types.Array(types.Text())),
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
