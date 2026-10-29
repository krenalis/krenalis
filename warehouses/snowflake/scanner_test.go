// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestScanStringArray preserves empty String elements and their positions.
func TestScanStringArray(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []any
	}{
		{"empty array", []any{}},
		{"nonempty elements", []any{"first", "second"}},
		{"empty element", []any{""}},
		{"empty first element", []any{"", "x"}},
		{"empty last element", []any{"x", ""}},
		{"only empty elements", []any{"", ""}},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.values)
			if err != nil {
				t.Fatalf("expected JSON String array, got %v", err)
			}
			s := &scanner{}
			got, err := s.normalize("value", types.Array(types.String()), string(data))
			if err != nil {
				t.Fatalf("expected String array, got %v", err)
			}
			if !reflect.DeepEqual(got, test.values) {
				t.Fatalf("expected String array %#v, got %#v", test.values, got)
			}
		})
	}
}
