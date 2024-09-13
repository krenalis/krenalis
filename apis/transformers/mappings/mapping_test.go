//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
)

func Test_InOutProperties(t *testing.T) {

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
		expressions   map[string]string
		inProperties  []string
		outProperties []string
	}{
		{
			expressions: map[string]string{
				"foo": "'a'",
				"boo": "5",
			},
			inProperties:  []string{},
			outProperties: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "'a'",
				"boo": "b.k.y",
			},
			inProperties:  []string{"b.y"},
			outProperties: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "b.k.x a b.k.x",
				"boo": "a b.k.x a",
			},
			inProperties:  []string{"a", "b.x"},
			outProperties: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "a '-' d.p.s? ' ' b['k'].z '*' c.z c.x",
			},
			inProperties:  []string{"a", "b.z", "c.x", "c.z", "d"},
			outProperties: []string{"foo"},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			mapping, err := New(test.expressions, inSchema, outSchema, nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got := mapping.InProperties()
			if got == nil || !slices.Equal(test.inProperties, got) {
				t.Fatalf("expected input properties %#v, got %#v", test.inProperties, got)
			}
			got = mapping.OutProperties()
			if got == nil || !slices.Equal(test.outProperties, got) {
				t.Fatalf("expected output properties %#v, got %#v", test.outProperties, got)
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
			name:        `An empty string assigned to a JSON property -> the empty string as JSON`,
			expressions: map[string]string{"C": "''"},
			expected:    map[string]any{"C": ""},
		},
		{
			name:        `A non-empty string assigned to a JSON property -> the string as JSON`,
			expressions: map[string]string{"C": "'boo'"},
			expected:    map[string]any{"C": "boo"},
		},
		{
			name:        `null assigned to a non-nullable JSON property -> the string as JSON`,
			expressions: map[string]string{"C": "null"},
			expected:    map[string]any{}, // REVIEW
		},
		{
			name:        `A property without a value assigned to a non-nullable JSON property -> no properties`,
			expressions: map[string]string{"C": "a"},
			expected:    map[string]any{},
		},
		{
			name:        `A property with a nil value assigned to a non-nullable JSON property -> nil as JSON`,
			expressions: map[string]string{"C": "c.z"},
			properties:  map[string]any{"c.z": nil},
			expected:    map[string]any{}, // REVIEW
		},
		{
			name:        `A property with an empty string assigned to a non-nullable JSON property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			properties:  map[string]any{"a": ""},
			expected:    map[string]any{"C": ""},
		},

		{
			name:        `An empty string assigned to a nullable JSON property -> the empty string as JSON`,
			expressions: map[string]string{"D": "''"},
			expected:    map[string]any{"D": ""},
		},
		{
			name:        `A non-empty string assigned to a nullable JSON property -> the string as JSON`,
			expressions: map[string]string{"D": "'boo'"},
			expected:    map[string]any{"D": "boo"},
		},
		{
			name:        `null assigned to a nullable JSON property -> nil`,
			expressions: map[string]string{"D": "null"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property without a value assigned to a non-nullable JSON property -> nil`,
			expressions: map[string]string{"D": "a"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with a nil value assigned to a non-nullable JSON property -> nil`,
			expressions: map[string]string{"D": "c.z"},
			properties:  map[string]any{"c.z": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable JSON property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			properties:  map[string]any{"a": ""},
			expected:    map[string]any{"C": ""},
		},

		{
			name:        `A JSON property without a value assigned to a non-nullable JSON property -> no properties`,
			expressions: map[string]string{"C": "d"},
			expected:    map[string]any{},
		},
		{
			name:        `A JSON property with a nil value assigned to a non-nullable JSON property -> no properties`,
			expressions: map[string]string{"C": "e"},
			properties:  map[string]any{"e": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A JSON property with a JSON null value assigned to a non-nullable JSON property -> no properties`,
			expressions: map[string]string{"C": "d"},
			properties:  map[string]any{"d": json.RawMessage(`null`)},
			expected:    map[string]any{"C": json.RawMessage(`null`)},
		},
		{
			name:        `A JSON property without a value assigned to a nullable JSON property -> nil`,
			expressions: map[string]string{"D": "e"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A JSON property with a nil value assigned to a nullable JSON property -> nil`,
			expressions: map[string]string{"D": "e"},
			properties:  map[string]any{"e": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A JSON property with a JSON null value assigned to a nullable JSON property -> no properties`,
			expressions: map[string]string{"D": "e"},
			properties:  map[string]any{"e": json.RawMessage(`null`)},
			expected:    map[string]any{"D": json.RawMessage(`null`)},
		},

		{
			name:        `A JSON property with a JSON null value assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "e"},
			properties:  map[string]any{"e": json.RawMessage(`null`)},
			expected:    map[string]any{},
		},
		{
			name:        `A JSON property with a JSON null value assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "e"},
			properties:  map[string]any{"e": json.RawMessage(`null`)},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping, err := New(test.expressions, inSchema, outSchema, test.layouts)
			if err != nil {
				t.Fatalf("unexpected error calling New: %q (%T)", err, err)
			}
			got, err := mapping.Transform(test.properties, test.purpose)
			if err != nil {
				if test.err == nil {
					t.Fatalf("unexpected error: %q (%T)", err, err)
				}
				if !cmp.Equal(test.err, err) {
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
