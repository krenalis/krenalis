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

// TestFileRecordSemantics checks all three file record entry points.
func TestFileRecordSemantics(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)
	tests := []struct {
		name     string
		semantic types.Type
		value    string
		valid    bool
	}{
		{"current country", country, "IT", true},
		{"former country", country, "AN", true},
		{"unknown country", country, "ZZ", false},
		{"reserved country", country, "UK", false},
		{"lowercase country", country, "it", false},
		{"empty country", country, "", false},
		{"short country", country, "I", false},
		{"long country", country, "ITA", false},
		{"non-ASCII country", country, "é", false},
		{"phone at limit", types.String().AsPhone(), "+123456789012345", true},
		{"long phone", types.String().AsPhone(), "+1234567890123456", false},
		{"multibyte phone at limit", types.String().AsPhone(), "éééééééé", true},
		{"multibyte phone over limit", types.String().AsPhone(), "ééééééééé", false},
		{"empty phone", types.String().AsPhone(), "", true},
		{"phone without format restriction", types.String().AsPhone(), "a (b)", true},
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
						t.Fatal(err)
					}
					if err := got.Err; err != nil {
						if test.valid {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[InputValidationError](err); !ok {
							t.Fatalf("expected InputValidationError, got %T", err)
						}
						return
					}
					if !test.valid {
						t.Fatal("invalid record was accepted")
					}
					if got.Attributes["value"] != test.value {
						t.Fatalf("value changed: %#v", got.Attributes)
					}

				})

			}

		})

	}

}
