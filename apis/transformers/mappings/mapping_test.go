//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"reflect"
	"testing"
)

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
