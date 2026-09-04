// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package properties

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// Test_Delete verifies deleting nested properties from maps, together with the
// objects their deletion leaves empty.
func Test_Delete(t *testing.T) {
	cases := []struct {
		name     string
		m        map[string]any
		path     []string
		expected map[string]any
	}{
		{
			name:     "top level property",
			m:        map[string]any{"a": 5, "b": 6},
			path:     []string{"a"},
			expected: map[string]any{"b": 6},
		},
		{
			name:     "nested property, with the object kept",
			m:        map[string]any{"consents": map[string]any{"marketing": true, "profiling": false}},
			path:     []string{"consents", "marketing"},
			expected: map[string]any{"consents": map[string]any{"profiling": false}},
		},
		{
			name:     "nested property, with the emptied objects deleted",
			m:        map[string]any{"a": 5, "traits": map[string]any{"privacy": map[string]any{"marketing": true}}},
			path:     []string{"traits", "privacy", "marketing"},
			expected: map[string]any{"a": 5},
		},
		{
			name:     "path that does not exist",
			m:        map[string]any{"consents": map[string]any{"marketing": true}},
			path:     []string{"consents", "profiling"},
			expected: map[string]any{"consents": map[string]any{"marketing": true}},
		},
		{
			name:     "path through a value that is not an object",
			m:        map[string]any{"a": 5},
			path:     []string{"a", "b"},
			expected: map[string]any{"a": 5},
		},
		{
			name:     "path through a JSON value",
			m:        map[string]any{"json": json.Value(`{"b":4}`)},
			path:     []string{"json", "b"},
			expected: map[string]any{"json": json.Value(`{"b":4}`)},
		},
	}

	for _, cas := range cases {
		t.Run(cas.name, func(t *testing.T) {
			Delete(cas.m, cas.path)
			if !reflect.DeepEqual(cas.m, cas.expected) {
				t.Fatalf("expected %v, got %v", cas.expected, cas.m)
			}
		})
	}
}

// Test_Read verifies retrieving nested properties from maps and JSON.
func Test_Read(t *testing.T) {
	jsonObj := json.Value(`{"b":{"c":4}}`)
	m := map[string]any{
		"a": 5,
		"nested": map[string]any{
			"b": map[string]any{"c": "foo"},
		},
		"json": jsonObj,
	}

	cases := []struct {
		path     []string
		expected any
		ok       bool
	}{
		{[]string{"a"}, 5, true},
		{[]string{"nested", "b", "c"}, "foo", true},
		{[]string{"nested", "x"}, nil, false},
		{[]string{"a", "b"}, nil, false},
		{[]string{"json", "b", "c"}, json.Value("4"), true},
		{[]string{"json", "x"}, nil, false},
	}

	for _, cas := range cases {
		got, ok := Read(m, cas.path)
		if ok != cas.ok || !reflect.DeepEqual(got, cas.expected) {
			t.Fatalf("%v: expected (%v,%v) got (%v,%v)", cas.path, cas.expected, cas.ok, got, ok)
		}
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic with empty path")
		}
	}()
	Read(m, []string{})
}
