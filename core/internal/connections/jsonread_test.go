// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestJSONNumberTransport preserves server text without inferring missing FLOAT
// digits or routing exact JSON numbers through binary64.
func TestJSONNumberTransport(t *testing.T) {
	for _, test := range []struct {
		name    string
		typ     types.Type
		input   any
		want    any
		invalid bool
	}{
		{"SQL null", types.JSON(), nil, nil, false},
		{"JSON null", types.JSON(), "null", json.Value("null"), false},
		{"missing property", types.JSON(), "{}", json.Value("{}"), false},
		{"null property", types.JSON(), `{"value":null}`, json.Value(`{"value":null}`), false},
		{"integer precision 38", types.JSON(), "12345678901234567890123456789012345678", json.Value("12345678901234567890123456789012345678"), false},
		{"decimal scale 37", types.JSON(), "0.0000000000000000000000000000000000001", json.Value("0.0000000000000000000000000000000000001"), false},
		{"server FLOAT text", types.JSON(), "1.000000000000000e+02", json.Value("1.000000000000000e+02"), false},
		{"negative zero", types.JSON(), "-0.000000000000000e+00", json.Value("-0.000000000000000e+00"), false},
		{"NaN text string", types.JSON(), `"NaN"`, json.Value(`"NaN"`), false},
		{"more precise server text", types.JSON(), "1.0000000000000001e+02", json.Value("1.0000000000000001e+02"), false},
		{"outside binary64 range", types.JSON(), "1.797693134862316e+308", json.Value("1.797693134862316e+308"), false},
		{"nested tokens and strings", types.JSON(), ` {"a.b":[1.000000000000000e+02,{"fixed":12345678901234567890123456789012345678,"text":"1E2"}]} `, json.Value(` {"a.b":[1.000000000000000e+02,{"fixed":12345678901234567890123456789012345678,"text":"1E2"}]} `), false},
		{"array repeated FLOAT text", types.Array(types.JSON()), `[1.000000000000000e+02,1.000000000000000e+02,12345678901234567890123456789012345678,null]`, []any{json.Value("1.000000000000000e+02"), json.Value("1.000000000000000e+02"), json.Value("12345678901234567890123456789012345678"), json.Value("null")}, false},
		{"map tokens", types.Map(types.JSON()), `{"a.b[0]":1.000000000000000e+02,"fixed":12345678901234567890123456789012345678,"null":null}`, map[string]any{"a.b[0]": json.Value("1.000000000000000e+02"), "fixed": json.Value("12345678901234567890123456789012345678"), "null": json.Value("null")}, false},
		{"NaN is not JSON", types.JSON(), "NaN", nil, true},
		{"infinity is not JSON", types.JSON(), "Infinity", nil, true},
		{"negative infinity is not JSON", types.JSON(), "-Infinity", nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalize("value", test.typ, test.input, true, nil)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected preserved JSON numbers, got %v", err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected invalid JSON error, got nil")
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected %#v, got %#v", test.want, got)
			}
		})
	}
}

// TestUpdatedAtJSONNumber keeps database cursors on temporal strings rather
// than numeric JSON values subject to server FLOAT formatting.
func TestUpdatedAtJSONNumber(t *testing.T) {
	for _, input := range []string{"100", "1.000000000000000e+02", "100.00000000000001"} {
		t.Run(input, func(t *testing.T) {
			_, err := parseUpdatedAtColumn("updated_at", types.JSON(), "ISO8601", json.Value(input), false, nil)
			if err != nil {
				return
			}
			t.Fatal("expected numeric JSON cursor rejection, got nil")
		})
	}
	got, err := parseUpdatedAtColumn("updated_at", types.JSON(), "ISO8601",
		json.Value(`"2024-11-01T12:30:00Z"`), false, nil)
	if err != nil {
		t.Fatalf("expected temporal JSON cursor, got %v", err)
	}
	want := time.Date(2024, 11, 1, 12, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
