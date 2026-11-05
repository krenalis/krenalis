// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"bytes"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestNormalizeJSON checks validation and normalization of open event data.
func TestNormalizeJSON(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "count", Type: types.Int(8), ReadOptional: true},
		{
			Name: "details", ReadOptional: true,
			Type: types.Object([]types.Property{
				{Name: "when", Type: types.DateTime()},
			}),
		},
		{Name: "raw", Type: types.JSON(), ReadOptional: true},
		{Name: "nullable", Type: types.String(), Nullable: true, ReadOptional: true},
	})
	large := strings.Repeat("9", 90)

	for _, test := range []struct {
		name    string
		value   string
		schema  types.Type
		want    string
		invalid bool
		path    string
	}{
		{name: "string", value: `"hello"`, schema: types.String(), want: `"hello"`},
		{name: "boolean", value: `true`, schema: types.Boolean(), want: `true`},
		{name: "integer", value: `123`, schema: types.Int(32), want: `123`},
		{name: "float", value: `1.250`, schema: types.Float(64).Real(), want: `1.25`},
		{name: "decimal precision", value: `12345678901234567890.1234`, schema: types.Decimal(30, 4), want: `12345678901234567890.1234`},
		{name: "datetime", value: `"2026-11-04T12:00:00.100000+01:00"`, schema: types.DateTime(), want: `"2026-11-04T11:00:00.100Z"`},
		{name: "date", value: `"2026-11-04"`, schema: types.Date(), want: `"2026-11-04"`},
		{name: "time", value: `"12:30:00.100000"`, schema: types.Time(), want: `"12:30:00.100"`},
		{name: "year", value: `2026`, schema: types.Year(), want: `2026`},
		{name: "uuid", value: `"550E8400-E29B-41D4-A716-446655440000"`, schema: types.UUID(), want: `"550e8400-e29b-41d4-a716-446655440000"`},
		{name: "ip", value: `"::ffff:192.0.2.1"`, schema: types.IP(), want: `"192.0.2.1"`},
		{name: "json null", value: `null`, schema: types.JSON(), want: `null`},
		{name: "opaque JSON", value: `{"b":2, "a":` + large + `}`, schema: types.JSON(), want: `{"b":2,"a":` + large + `}`},
		{name: "array", value: `["2026-11-04T12:00:00+01:00"]`, schema: types.Array(types.DateTime()), want: `["2026-11-04T11:00:00Z"]`},
		{name: "JSON array with null", value: `[null,"1",1]`, schema: types.Array(types.JSON()), want: `[null,"1",1]`},
		{name: "map", value: `{"z":"2026-11-04T12:00:00+01:00","a":"2026-11-04T10:00:00Z"}`, schema: types.Map(types.DateTime()), want: `{"a":"2026-11-04T10:00:00Z","z":"2026-11-04T11:00:00Z"}`},
		{name: "empty object", value: `{}`, schema: schema, want: `{}`},
		{name: "schema then extra order", value: `{"z":true,"nullable":"x","count":1,"a":"extra"}`, schema: schema, want: `{"count":1,"nullable":"x","a":"extra","z":true}`},
		{name: "nested extra properties", value: `{"details":{"extra":null,"when":"2026-11-04T12:00:00+01:00"}}`, schema: schema, want: `{"details":{"when":"2026-11-04T11:00:00Z","extra":null}}`},
		{name: "arbitrary extra keys", value: `{"a.b":1,"":false,"not-a-property":null}`, schema: schema, want: `{"":false,"a.b":1,"not-a-property":null}`},
		{name: "extra precision and kind", value: `{"number":` + large + `,"string":"100","null":null}`, schema: schema, want: `{"null":null,"number":` + large + `,"string":"100"}`},
		{name: "opaque declared field", value: `{"raw":{"outside.schema":[1,null,"1"]}}`, schema: schema, want: `{"raw":{"outside.schema":[1,null,"1"]}}`},
		{name: "nullable property", value: `{"nullable":null}`, schema: schema, want: `{"nullable":null}`},
		{name: "declared JSON null", value: `{"raw":null}`, schema: schema, want: `{"raw":null}`},
		{name: "number to string deferred", value: `1`, schema: types.String(), invalid: true},
		{name: "string to number deferred", value: `"1"`, schema: types.Int(32), invalid: true},
		{name: "string to boolean deferred", value: `"true"`, schema: types.Boolean(), invalid: true},
		{name: "fractional integer", value: `1.5`, schema: types.Int(32), invalid: true},
		{name: "integer range", value: `128`, schema: types.Int(8), invalid: true},
		{name: "decimal range", value: `1000`, schema: types.Decimal(5, 2), invalid: true},
		{name: "string length", value: `"long"`, schema: types.String().WithMaxLength(3), invalid: true},
		{name: "string bytes", value: `"éé"`, schema: types.String().WithMaxBytes(3), invalid: true},
		{name: "invalid date", value: `"2026-02-30"`, schema: types.Date(), invalid: true},
		{name: "invalid UUID", value: `"abc"`, schema: types.UUID(), invalid: true},
		{name: "invalid IP", value: `"999.0.0.1"`, schema: types.IP(), invalid: true},
		{name: "non-finite real", value: `"NaN"`, schema: types.Float(64).Real(), invalid: true},
		{name: "invalid array element", value: `[1]`, schema: types.Array(types.String()), invalid: true},
		{name: "null array element", value: `["x",null]`, schema: types.Array(types.String()), invalid: true},
		{name: "invalid map value", value: `{"x":1}`, schema: types.Map(types.String()), invalid: true},
		{name: "null map value", value: `{"x":null}`, schema: types.Map(types.String()), invalid: true},
		{name: "object null", value: `null`, schema: schema, invalid: true},
		{name: "scalar null", value: `null`, schema: types.Boolean(), invalid: true},
		{name: "array null", value: `null`, schema: types.Array(types.JSON()), invalid: true},
		{name: "object wrong kind", value: `[]`, schema: schema, invalid: true},
		{name: "non-nullable property", value: `{"count":null}`, schema: schema, invalid: true, path: "count"},
		{name: "nested property", value: `{"details":{"when":"invalid"}}`, schema: schema, invalid: true, path: "details.when"},
		{name: "required property", value: `{"details":{}}`, schema: schema, invalid: true, path: "details.when"},
		{name: "malformed object", value: `{"count":`, schema: schema, invalid: true},
		{name: "malformed extra", value: `{"extra":[}`, schema: schema, invalid: true},
		{name: "empty input", value: ``, schema: types.String(), invalid: true},
		{name: "malformed scalar", value: `tru`, schema: types.Boolean(), invalid: true},
		{name: "trailing value", value: `{} {}`, schema: schema, invalid: true},
		{name: "trailing scalar", value: `1 2`, schema: types.Int(32), invalid: true},
		{name: "trailing opaque value", value: `null true`, schema: types.JSON(), invalid: true},
		{name: "duplicate property", value: `{"count":1,"count":2}`, schema: schema, invalid: true},
		{name: "duplicate extra", value: `{"extra":1,"extra":2}`, schema: schema, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {

			input := json.Value(test.value)
			got, err := normalizeJSON("properties", input, test.schema)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected normalized JSON, got %v", err)
				}
				validation, ok := errors.AsType[ValidationError](err)
				if !ok {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				path := "properties"
				if test.path != "" {
					path += "." + test.path
				}
				if validation.PropertyPath() != path {
					t.Fatalf("expected path %q, got %q", path, validation.PropertyPath())
				}
				if got != nil {
					t.Fatalf("expected no partial result, got %s", got)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected a validation error, got nil")
			}
			if string(got) != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
			if string(input) != test.value {
				t.Fatalf("expected unchanged input %s, got %s", test.value, input)
			}

			clear(got)
			if !bytes.Equal(input, []byte(test.value)) {
				t.Fatalf("expected independently owned output, got modified input %q", input)
			}

		})
	}

}

// TestNormalizeJSONInvalidSchema keeps configuration errors distinct from data.
func TestNormalizeJSONInvalidSchema(t *testing.T) {
	for _, schema := range []types.Type{{}, types.Parameter("T")} {
		got, err := normalizeJSON("properties", json.Value(`{}`), schema)
		if err != nil {
			if _, ok := errors.AsType[ValidationError](err); ok {
				t.Fatalf("expected an internal configuration error, got %T", err)
			}
			if got != nil {
				t.Fatalf("expected no result, got %s", got)
			}
			continue
		}
		t.Fatal("expected an invalid schema error, got nil")
	}
}
