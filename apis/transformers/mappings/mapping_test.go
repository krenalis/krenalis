//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"reflect"
	"slices"
	"testing"

	"github.com/meergo/meergo/types"
)

func TestInOutProperties(t *testing.T) {

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
		mapping       map[string]string
		inProperties  []string
		outProperties []string
	}{
		{
			mapping: map[string]string{
				"foo": "'a'",
				"boo": "5",
			},
			inProperties:  []string{},
			outProperties: []string{"boo", "foo"},
		},
		{
			mapping: map[string]string{
				"foo": "'a'",
				"boo": "b.k.y",
			},
			inProperties:  []string{"b.y"},
			outProperties: []string{"boo", "foo"},
		},
		{
			mapping: map[string]string{
				"foo": "b.k.x a b.k.x",
				"boo": "a b.k.x a",
			},
			inProperties:  []string{"a", "b.x"},
			outProperties: []string{"boo", "foo"},
		},
		{
			mapping: map[string]string{
				"foo": "a '-' d.p.s? ' ' b['k'].z '*' c.z c.x",
			},
			inProperties:  []string{"a", "b.z", "c.x", "c.z", "d"},
			outProperties: []string{"foo"},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			mapping, err := New(test.mapping, inSchema, outSchema, nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got := mapping.InProperties()
			if got == nil || !slices.Equal(test.inProperties, got) {
				t.Fatalf("expecting input properties %#v, got %#v", test.inProperties, got)
			}
			got = mapping.OutProperties()
			if got == nil || !slices.Equal(test.outProperties, got) {
				t.Fatalf("expecting output properties %#v, got %#v", test.outProperties, got)
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
				t.Fatalf("expecting %#v, got %#v", test.expected, test.value)
			}
		})
	}
}
