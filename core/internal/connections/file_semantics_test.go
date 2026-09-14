// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestFileRecordSemantics checks semantic validation and normalization through all three file record entry points.
func TestFileRecordSemantics(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)
	tests := []struct {
		name     string
		semantic types.Type
		value    string
		want     string
	}{
		{"current country", country, "IT", "IT"},
		{"former country", country, "AN", "AN"},
		{"unknown country", country, "ZZ", ""},
		{"reserved country", country, "UK", ""},
		{"lowercase country", country, "it", ""},
		{"empty country", country, "", ""},
		{"short country", country, "I", ""},
		{"long country", country, "ITA", ""},
		{"non-ASCII country", country, "é", ""},
		{"canonical phone", types.String().AsPhone(), "+390236618300", "+390236618300"},
		{"formatted phone", types.String().AsPhone(), "+39 02-36618 300", "+390236618300"},
		{"structurally possible phone", types.String().AsPhone(), "+12001230101", "+12001230101"},
		{"local-only phone", types.String().AsPhone(), "+12530000", ""},
		{"double plus phone", types.String().AsPhone(), "++390236618300", ""},
		{"long phone", types.String().AsPhone(), "+1234567890123456", ""},
		{"multibyte phone at limit", types.String().AsPhone(), "éééééééé", ""},
		{"multibyte phone over limit", types.String().AsPhone(), "ééééééééé", ""},
		{"empty phone", types.String().AsPhone(), "", ""},
		{"phone without format restriction", types.String().AsPhone(), "a (b)", ""},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			schema := types.Object([]types.Property{{Name: "value", Type: test.semantic}})
			for _, method := range []string{"Record", "RecordSlice", "RecordStrings"} {

				t.Run(method, func(t *testing.T) {

					var got Record
					rw := newRecordWriter("test", nil, time.Time{}, nil, time.Time{}, 2)
					rw.properties = schema.Properties().Slice()
					rw.numPropertiesPerRecord = 1
					rw.yield = func(record Record) bool {
						got = record
						return true
					}
					var err error
					switch method {
					case "Record":
						err = rw.Record(map[string]any{"value": test.value})
					case "RecordSlice":
						err = rw.RecordSlice([]any{test.value})
					case "RecordStrings":
						err = rw.RecordStrings([]string{test.value})
					}
					if err != nil {
						t.Fatalf("expected no error, got %v", err)
					}
					if err := got.Err; err != nil {
						if test.want != "" {
							t.Fatalf("expected no record error, got %v", err)
						}
						if _, ok := errors.AsType[InputValidationError](err); !ok {
							t.Fatalf("expected InputValidationError, got %T", err)
						}
						return
					}
					if test.want == "" {
						t.Fatal("expected a record error, got nil")
					}
					if got.Attributes["value"] != test.want {
						t.Fatalf("expected value %q, got %#v", test.want, got.Attributes["value"])
					}

				})

			}

		})

	}

}
