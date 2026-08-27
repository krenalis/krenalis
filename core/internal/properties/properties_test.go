// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package properties

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

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
