// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"bytes"
	"fmt"
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
)

func Test_Unmarshal(t *testing.T) {

	schema := types.Object([]types.Property{
		{
			Name: "Text",
			Type: types.Text().WithCharLen(10),
		},
		{
			Name: "Text_values",
			Type: types.Text().WithValues("a", "b", "c"),
		},
		{
			Name: "Text_regexp",
			Type: types.Text().WithRegexp(regexp.MustCompile(`o/o$`)),
		},
		{
			Name:     "Text_nil",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name: "Boolean",
			Type: types.Boolean(),
		},
		{
			Name: "Int8",
			Type: types.Int(8).WithIntRange(-20, 20),
		},
		{
			Name: "Int16",
			Type: types.Int(16),
		},
		{
			Name: "Int24",
			Type: types.Int(24),
		},
		{
			Name: "Int32",
			Type: types.Int(32),
		},
		{
			Name: "Int64",
			Type: types.Int(64),
		},
		{
			Name: "Uint8",
			Type: types.Uint(8),
		},
		{
			Name: "Uint16",
			Type: types.Uint(16),
		},
		{
			Name: "Uint24",
			Type: types.Uint(24),
		},
		{
			Name: "Uint32",
			Type: types.Uint(32),
		},
		{
			Name: "Uint64",
			Type: types.Uint(64),
		},
		{
			Name: "Float32",
			Type: types.Float(32),
		},
		{
			Name: "Float64",
			Type: types.Float(64),
		},
		{
			Name: "Float64_NaN",
			Type: types.Float(64),
		},
		{
			Name: "Float64_Infinity",
			Type: types.Float(64),
		},
		{
			Name: "Float64_NegInfinity",
			Type: types.Float(64),
		},
		{
			Name: "Decimal",
			Type: types.Decimal(10, 3),
		},
		{
			Name: "DateTime",
			Type: types.DateTime(),
		},
		{
			Name: "Date",
			Type: types.Date(),
		},
		{
			Name: "Time",
			Type: types.Time(),
		},
		{
			Name: "Year",
			Type: types.Year(),
		},
		{
			Name: "UUID",
			Type: types.UUID(),
		},
		{
			Name: "JSON",
			Type: types.JSON(),
		},
		{
			Name: "JSON_null",
			Type: types.JSON(),
		},
		{
			Name:     "JSON_nil",
			Type:     types.JSON(),
			Nullable: true,
		},
		{
			Name: "Inet",
			Type: types.Inet(),
		},
		{
			Name: "Array",
			Type: types.Array(types.Text()),
		},
		{
			Name: "Object",
			Type: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Boolean(),
				},
				{
					Name:           "b",
					Type:           types.Int(32),
					CreateRequired: true,
				},
				{
					Name:           "c",
					Type:           types.JSON(),
					UpdateRequired: true,
				},
				{
					Name:           "d",
					Type:           types.DateTime(),
					UpdateRequired: true,
					CreateRequired: true,
				},
			}),
		},
		{
			Name: "Map",
			Type: types.Map(types.Int(32)),
		},
	})

	records := []Record{
		{
			Attributes: map[string]any{
				"Text":                "some text",
				"Text_nil":            nil,
				"Boolean":             true,
				"Int8":                -12,
				"Int16":               8023,
				"Int24":               -2880217,
				"Int32":               1307298102,
				"Int64":               927041163082605,
				"Uint8":               uint(12),
				"Uint16":              uint(8023),
				"Uint24":              uint(2880217),
				"Uint32":              uint(1307298102),
				"Uint64":              uint(927041163082605),
				"Float32":             float64(float32(57.16038)),
				"Float64":             18372.36240184391,
				"Float64_NaN":         math.NaN(),
				"Float64_Infinity":    math.Inf(1),
				"Float64_NegInfinity": math.Inf(-1),
				"Decimal":             decimal.MustParse("1752.064"),
				"DateTime":            time.Date(2023, 10, 17, 9, 34, 25, 836540129, time.UTC),
				"Date":                time.Date(2023, 10, 17, 0, 0, 0, 0, time.UTC),
				"Time":                time.Date(1970, 01, 01, 9, 34, 25, 836540129, time.UTC),
				"Year":                2023,
				"UUID":                "550e8400-e29b-41d4-a716-446655440000",
				"JSON":                json.Value(`{"foo": 5,"boo": true}`),
				"JSON_null":           json.Value(`null`),
				"Inet":                "192.158.1.38",
				"Array":               []any{"foo", "boo"},
				"Object":              map[string]any{"a": false, "b": 9},
				"Map":                 map[string]any{"a": 1, "b": 2, "c": 3},
			},
		},
	}

	tests := []struct {
		language     state.Language
		schema       types.Type
		preserveJSON bool
		timeTruncate time.Duration
		data         string
		records      []Record
		expected     []Record
		err          error
	}{
		{
			language:     state.JavaScript,
			schema:       schema,
			preserveJSON: false,
			timeTruncate: time.Millisecond,
			data:         `{"records":[{"value":{"Text":"some text","Text_nil":null,"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17T09:34:25.836Z","Date":"2023-10-17T00:00:00.000Z","Time":"1970-01-01T09:34:25.836Z","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":{"foo": 5,"boo": true},"JSON_null":null,"Inet":"192.158.1.38","Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.JavaScript,
			schema:       schema,
			preserveJSON: true,
			timeTruncate: time.Millisecond,
			data:         `{"records":[{"value":{"Text":"some text","Text_nil":null,"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":"927041163082605","Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":"927041163082605","Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17T09:34:25.836Z","Date":"2023-10-17T00:00:00.000Z","Time":"1970-01-01T09:34:25.836Z","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":"{\"foo\": 5,\"boo\": true}","JSON_null":"null","Inet":"192.158.1.38","Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.Python,
			schema:       schema,
			preserveJSON: false,
			timeTruncate: time.Microsecond,
			data:         `{"records":[{"value":{"Text":"some text","Text_nil":null,"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":927041163082605,"Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":927041163082605,"Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17 09:34:25.83654","Date":"2023-10-17","Time":"09:34:25.83654","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":{"foo": 5,"boo": true},"JSON_null":null,"Inet":"192.158.1.38","Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.Python,
			schema:       schema,
			preserveJSON: true,
			timeTruncate: time.Microsecond,
			data:         `{"records":[{"value":{"Text":"some text","Text_nil":null,"Boolean":true,"Int8":-12,"Int16":8023,"Int24":-2880217,"Int32":1307298102,"Int64":927041163082605,"Uint8":12,"Uint16":8023,"Uint24":2880217,"Uint32":1307298102,"Uint64":927041163082605,"Float32":57.16038,"Float64":18372.36240184391,"Float64_NaN":"NaN","Float64_Infinity":"Infinity","Float64_NegInfinity":"-Infinity","Decimal":"1752.064","DateTime":"2023-10-17 09:34:25.83654","Date":"2023-10-17","Time":"09:34:25.83654","Year":2023,"UUID":"550e8400-e29b-41d4-a716-446655440000","JSON":"{\"foo\": 5,\"boo\": true}","JSON_null":"null","Inet":"192.158.1.38","Array":["foo","boo"],"Object":{"a":false,"b":9},"Map":{"a":1,"b":2,"c":3}}}]}`,
			records:      records,
		},
		{
			language:     state.JavaScript,
			schema:       schema,
			preserveJSON: true,
			data:         `{"records":[{"value":{"JSON_nil":null}}]}`,
			records:      []Record{{Attributes: map[string]any{"JSON_nil": nil}}},
		},
		{
			language:     state.Python,
			schema:       schema,
			preserveJSON: true,
			data:         `{"records":[{"value":{"JSON_nil":null}}]}`,
			records:      []Record{{Attributes: map[string]any{"JSON_nil": nil}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     ``,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{}`,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `[{"value":"boo"}]`,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[]}`,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[],}`,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[],"records":[]}`,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[5]}`,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":}}]}`,
			records:  make([]Record, 1),
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":true`,
			records:  make([]Record, 1),
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"error":true}`,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"error":"syntax error","records":[{"value":4}]}`,
			err:      errInvalidResponseFormat,
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"e":5}}}]}`,
			records:  []Record{{Err: newRecordValidationError("Object.e", `property «Object.e» is not part of output schema; rename or remove it in the transformation function`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"a":true}}}]}`,
			records:  []Record{{Purpose: Create, Err: newRecordValidationError("Object.b", `property «Object.b» is missing but it is required for creation`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"a":true}}}]}`,
			records:  []Record{{Purpose: Update, Err: newRecordValidationError("Object.c", `property «Object.c» is missing but it is required for update`)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{"b":false}}}]}`,
			records:  []Record{{Err: newRecordValidationError("Object.b", `property «Object.b» has a value that is not of type «number»`)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Int8":21}}]}`,
			records:  []Record{{Err: newRecordValidationError("Int8", `property «Int8» is greater than 20`)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Int8":-25}}]}`,
			records:  []Record{{Err: newRecordValidationError("Int8", `property «Int8» is less than -20`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":"a \" \\ b"}}]}`,
			records:  []Record{{Err: newRecordValidationError("Boolean", `property «Boolean» has a value that is not of type «bool»`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":null}}]}`,
			records:  []Record{{Err: newRecordValidationError("Boolean", `property «Boolean» cannot be «None», but it is set to «None»`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Date":"2023-02-30"}}]}`,
			records:  []Record{{Err: newRecordValidationError("Date", `property «Date» has a value that is not of type «datetime.date»`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text":"some long text"}}]}`,
			records:  []Record{{Err: newRecordValidationError("Text", `property «Text» exceeds the 10-char limit`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_values":"c"}}]}`,
			records:  []Record{{Attributes: map[string]any{"Text_values": "c"}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_values":"foo"}}]}`,
			records:  []Record{{Err: newRecordValidationError("Text_values", `property «Text_values» is not one of the allowed values`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_regexp":"fo/o"}}]}`,
			records:  []Record{{Attributes: map[string]any{"Text_regexp": "fo/o"}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Text_regexp":"faa"}}]}`,
			records:  []Record{{Err: newRecordValidationError("Text_regexp", `property «Text_regexp» does not match «/o\/o$/»`)}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{}},{"value":{}}]}`,
			records:  []Record{{Attributes: map[string]any{}}, {Attributes: map[string]any{}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":true}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Attributes: map[string]any{"Boolean": true}}, {Attributes: map[string]any{"Int32": 547}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"foo":"boo"}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Err: newRecordValidationError("foo", `property «foo» is not part of output schema; rename or remove it in the transformation function`)}, {Attributes: map[string]any{"Int32": 547}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":{}}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Purpose: Update, Err: newRecordValidationError("Object.c", `property «Object.c» is missing but it is required for update`)}, {Attributes: map[string]any{"Int32": 547}}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":3}},{"value":{"Int32":547}}]}`,
			records:  []Record{{Err: newRecordValidationError("Boolean", `property «Boolean» has a value that is not of type «boolean»`)}, {Attributes: map[string]any{"Int32": 547}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":3}},{"value":{"Object":{}}}]}`,
			records: []Record{
				{Err: newRecordValidationError("Boolean", `property «Boolean» has a value that is not of type «bool»`)},
				{Purpose: Create, Err: newRecordValidationError("Object.b", `property «Object.b» is missing but it is required for creation`)},
			},
		},
		{
			language: state.JavaScript,
			schema:   types.Type{},
			data:     `{"records":[{"value":{}},{"value":{"foo":5}}]}`,
			records:  []Record{{Attributes: map[string]any{}}, {Err: newRecordValidationError("foo", `property «foo» is not part of output schema; rename or remove it in the transformation function`)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"a.b.c":5}}]}`,
			records:  []Record{{Err: newRecordValidationError("a.b.c", `property «a.b.c» is not part of output schema; rename or remove it in the transformation function`)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Boolean":[true]}}]}`,
			records:  []Record{{Err: newRecordValidationError("Boolean", `property «Boolean» has a value that is not of type «boolean»`)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"error":"unexpected token ')'"}`,
			err:      FunctionExecError{"unexpected token ')'"},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"error":"an error occurred"}]}`,
			records:  []Record{{Err: RecordTransformationError{msg: "JavaScript: an error occurred"}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"error":"an error occurred"}]}`,
			records:  []Record{{Err: RecordTransformationError{msg: "Python: an error occurred"}}},
		},
		{
			language: state.Python,
			schema:   schema,
			data:     `{"records":[{"value":{"Object":true}}]}`,
			records:  []Record{{Err: newRecordValidationError("Object", `property «Object» has a value that is not of type «dict»`)}},
		},
		{
			language: state.JavaScript,
			schema:   schema,
			data:     `{"records":[{"value":{"Array":1}}]}`,
			records:  []Record{{Err: newRecordValidationError("Array", `property «Array» has a value that is not of type «array of string»`)}},
		},
	}

	for _, test := range tests {
		t.Run(test.language.String(), func(t *testing.T) {
			b := strings.NewReader(test.data)
			records := make([]Record, len(test.records))
			for i, record := range test.records {
				records[i].Purpose = record.Purpose
			}
			err := Unmarshal(b, records, test.schema, test.language, test.preserveJSON)
			if err != nil {
				if test.err == nil {
					t.Fatalf("Unmarshal: expected no error, got error %s", err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("Unmarshal: expected error %q, got error %q", test.err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("Unmarshal: expected error %q, got no error", test.err)
			}
			for i, want := range test.records {
				got := records[i]
				if got.Err != nil {
					if want.Err == nil {
						t.Fatalf("Unmarshal:\n\texpected no error\n\tgot error %q", got.Err.Error())
					}
					if !reflect.DeepEqual(want.Err, got.Err) {
						t.Fatalf("Unmarshal:\n\texpected error %q\n\tgot error %q", want.Err.Error(), got.Err.Error())
					}
					continue
				}
				if want.Err != nil {
					t.Fatalf("Unmarshal:\n\texpected error %q\n\tgot no error", want.Err)
				}
				if got.Attributes == nil {
					t.Fatalf("Unmarshal:\n\texpected attributes\n\tgot no attributes")
				}
				if err := equalValues(schema, test.timeTruncate, want.Attributes, got.Attributes); err != nil {
					t.Fatalf("Unmarshal:\n\texpected attributes %#v\n\tgot attributes      %#v\n\terror:   %s", want.Attributes, got.Attributes, err)
				}
			}
		})
	}

}

// Test_UnmarshalEdgeCases checks error scenarios and boundary cases for Unmarshal.
func Test_UnmarshalEdgeCases(t *testing.T) {
	simple := types.Object([]types.Property{{Name: "a", Type: types.Int(32)}})
	buf := strings.NewReader(`{"records":[{"value":{}}]}`)

	t.Run("nil reader", func(t *testing.T) {
		err := Unmarshal(nil, make([]Record, 1), simple, state.JavaScript, false)
		if err == nil || err.Error() != "core/transformers: r is nil" {
			t.Fatalf("expected r nil error, got %v", err)
		}
	})

	t.Run("schema not object", func(t *testing.T) {
		err := Unmarshal(buf, make([]Record, 1), types.Text(), state.JavaScript, false)
		if err == nil || err.Error() != "core/transformers: schema is not an object" {
			t.Fatalf("expected schema error, got %v", err)
		}
	})

	t.Run("invalid language", func(t *testing.T) {
		err := Unmarshal(buf, make([]Record, 1), simple, state.Language(7), false)
		if err == nil || err.Error() != "core/transformers: language is not valid" {
			t.Fatalf("expected language error, got %v", err)
		}
	})

	t.Run("more results than expected", func(t *testing.T) {
		data := strings.NewReader(`{"records":[{"value":{}},{"value":{}}]}`)
		err := Unmarshal(data, make([]Record, 1), simple, state.JavaScript, false)
		want := "core/transformers: expected 1 results got more"
		if err == nil || err.Error() != want {
			t.Fatalf("expected %q, got %v", want, err)
		}
	})

	t.Run("fewer results than expected", func(t *testing.T) {
		data := strings.NewReader(`{"records":[{"value":{}}]}`)
		err := Unmarshal(data, make([]Record, 2), simple, state.JavaScript, false)
		want := "core/transformers: expected 2 results got 1"
		if err == nil || err.Error() != want {
			t.Fatalf("expected %q, got %v", want, err)
		}
	})

	t.Run("array unique duplicated", func(t *testing.T) {
		sch := types.Object([]types.Property{{Name: "a", Type: types.Array(types.Text()).WithUnique()}})
		rec := []Record{{}}
		data := strings.NewReader(`{"records":[{"value":{"a":["x","x"]}}]}`)
		err := Unmarshal(data, rec, sch, state.JavaScript, false)
		if err != errInvalidResponseFormat {
			t.Fatalf("expected errInvalidResponseFormat, got %v", err)
		}
		if rec[0].Err == nil || rec[0].Err.Error() != "property «a» contains a duplicated value" {
			t.Fatalf("unexpected record error: %v", rec[0].Err)
		}
	})

	t.Run("array element bounds", func(t *testing.T) {
		sch := types.Object([]types.Property{{Name: "a", Type: types.Array(types.Int(32)).WithMinElements(2).WithMaxElements(3)}})
		rec := []Record{{}}
		less := strings.NewReader(`{"records":[{"value":{"a":[1]}}]}`)
		err := Unmarshal(less, rec, sch, state.JavaScript, false)
		if err != errInvalidResponseFormat {
			t.Fatalf("expected errInvalidResponseFormat for less elements, got %v", err)
		}
		if rec[0].Err == nil || rec[0].Err.Error() != "property «a» contains less than 2 elements" {
			t.Fatalf("unexpected error for less elements: %v", rec[0].Err)
		}
		rec[0].Err = nil
		rec[0].Attributes = nil
		more := strings.NewReader(`{"records":[{"value":{"a":[1,2,3,4]}}]}`)
		err = Unmarshal(more, rec, sch, state.JavaScript, false)
		if err != nil {
			t.Fatalf("unexpected error for more elements: %v", err)
		}
		if rec[0].Err != nil {
			t.Fatalf("did not expect record error for more elements: %v", rec[0].Err)
		}
		if got := rec[0].Attributes["a"]; got == nil || len(got.([]any)) != 4 {
			t.Fatalf("expected 4 elements, got %v", got)
		}
	})
}

// equalValues reports whether v1 and v2 are equal according to the type t.
// v1 is supposed to conform to type t, and v2 is checked for equality with v1.
// If t is a datetime, date, or time type, v1 is truncated to a multiple of
// timeTruncate.
func equalValues(t types.Type, timeTruncate time.Duration, v1, v2 any) error {
	if v1 == nil {
		if v2 != nil {
			return fmt.Errorf("expected nil, got %#v (%T)", v2, v2)
		}
		return nil
	} else if v2 == nil {
		return fmt.Errorf("expected %#v (%T), got nil", v1, v1)
	}
	switch t.Kind() {
	case types.FloatKind:
		f2, ok := v2.(float64)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		f1 := v1.(float64)
		switch {
		case math.IsNaN(f1):
			if !math.IsNaN(f2) {
				if t.BitSize() == 32 {
					return fmt.Errorf("expected value NaN, got %f", float32(f2))
				}
				return fmt.Errorf("expected value NaN, got %f", f2)
			}
		case math.IsInf(f1, 1):
			if !math.IsInf(f2, 1) {
				if t.BitSize() == 32 {
					return fmt.Errorf("expected value +Inf, got %f", float32(f2))
				}
				return fmt.Errorf("expected value +Inf, got %f", f2)
			}
		case math.IsInf(f1, -1):
			if !math.IsInf(f2, -1) {
				if t.BitSize() == 32 {
					return fmt.Errorf("expected value -Inf, got %f", float32(f2))
				}
				return fmt.Errorf("expected value -Inf, got %f", f2)
			}
		case t.BitSize() == 32:
			if float32(f1) != float32(f2) {
				return fmt.Errorf("expected value %f, got %f", float32(f1), float32(f2))
			}
		default:
			if f1 != f2 {
				return fmt.Errorf("expected value %f, got %f", f1, f2)
			}
		}
		return nil
	case types.DecimalKind:
		d2, ok := v2.(decimal.Decimal)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		d1 := v1.(decimal.Decimal)
		if d1.Cmp(d2) != 0 {
			return fmt.Errorf("expected value %s, got %s", v1, d2)
		}
		return nil
	case types.DateTimeKind, types.DateKind, types.TimeKind:
		t2, ok := v2.(time.Time)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		t1 := v1.(time.Time).Truncate(timeTruncate)
		if !t1.Equal(t2) {
			return fmt.Errorf("expected value %s, got %s", v1, t2)
		}
		return nil
	case types.JSONKind:
		j2, ok := v2.(json.Value)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		j1 := v1.(json.Value)
		if !bytes.Equal(j1, j2) {
			return fmt.Errorf("expected value %q (%T), got %q (%T)", string(j1), v1, string(j2), v2)
		}
		return nil
	case types.ArrayKind:
		a1 := v1.([]any)
		a2, ok := v2.([]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		for i, e1 := range a1 {
			err := equalValues(t.Elem(), timeTruncate, e1, a2[i])
			if err != nil {
				return err
			}
		}
		return nil
	case types.ObjectKind:
		o1 := v1.(map[string]any)
		o2, ok := v2.(map[string]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		unexpected := maps.Clone(o2)
		for _, p := range t.Properties().All() {
			s1, ok := o1[p.Name]
			if !ok {
				_, ok := o2[p.Name]
				if ok {
					return fmt.Errorf("not expected property %s, got property", p.Name)
				}
				continue
			}
			s2, ok := o2[p.Name]
			if !ok {
				return fmt.Errorf("expected property %s, got no property", p.Name)
			}
			err := equalValues(p.Type, timeTruncate, s1, s2)
			if err != nil {
				return err
			}
			delete(unexpected, p.Name)
		}
		if len(unexpected) > 0 {
			keys := slices.Sorted(maps.Keys(unexpected))
			return fmt.Errorf("unexpected property %q", keys[0])
		}
		return nil
	case types.MapKind:
		m1 := v1.(map[string]any)
		m2, ok := v2.(map[string]any)
		if !ok {
			return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
		}
		names := slices.Sorted(maps.Keys(m2))
		if len(m1) != len(m2) {
			for _, name := range names {
				if _, ok := m1[name]; !ok {
					return fmt.Errorf("unexpected property %q", name)
				}
			}
		}
		for _, name := range names {
			e1, ok := m1[name]
			if !ok {
				return fmt.Errorf("unexpected property %q", name)
			}
			e2 := m2[name]
			err := equalValues(t.Elem(), timeTruncate, e1, e2)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if v1 != v2 {
		return fmt.Errorf("expected value %#v (%T), got %#v (%T)", v1, v1, v2, v2)
	}
	return nil
}
