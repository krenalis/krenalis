// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"testing"

	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

func Test_sameValue(t *testing.T) {

	object := types.Object([]types.Property{
		{Name: "foo", Type: types.String()},
		{Name: "boo", Type: types.Array(types.Boolean())},
	})

	tests := []struct {
		t        types.Type
		v, v2    any
		expected bool
	}{
		{t: types.String(), v: nil, v2: nil, expected: true},
		{t: types.String(), v: nil, v2: 5, expected: false},
		{t: types.Int(32), v: 4, v2: 4, expected: true},
		{t: types.Int(32), v: 4, v2: nil, expected: false},
		{t: types.Float(64), v: 12.9037, v2: 12.9037, expected: true},
		{t: types.JSON(), v: nil, v2: nil, expected: true},
		{t: types.JSON(), v: json.Value(`null`), v2: json.Value(`null`), expected: true},
		{t: types.JSON(), v: json.Value(`{"a":3,"b":[1,2]}`), v2: json.Value(`{"a":3,"b":[1,2]}`), expected: true},
		{t: types.JSON(), v: json.Value(`{"a":3,"b":[1,2]}`), v2: json.Value(`{"a":3,"c":true}`), expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{"a", "b"}, expected: true},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{"b", "a"}, expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{}, expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: nil, expected: false},
		{t: object, v: map[string]any{}, v2: nil, expected: false},
		{t: object, v: map[string]any{}, v2: map[string]any{}, expected: true},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: map[string]any{"foo": "a", "boo": []any{true, false, true}}, expected: true},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: map[string]any{"foo": "a", "boo": []any{true, true, true}}, expected: false},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: nil, expected: false},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"a": 5, "b": 0}, expected: true},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"b": 0, "a": 5}, expected: true},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"b": 0, "a": 3}, expected: false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := sameValue(test.t, test.v, test.v2)
			if test.expected != got {
				t.Fatalf("expected %t, got %t", test.expected, got)
			}
		})
	}

}
