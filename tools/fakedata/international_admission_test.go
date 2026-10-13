// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

// TestInternationalClosedDocuments checks closed nested objects and raw token
// failures through the full socioeconomic entry point.
func TestInternationalClosedDocuments(t *testing.T) {

	encoded, err := internationalCanonical(internationalSocioeconomicFixture(t))
	if err != nil {
		t.Fatalf("expected artificial model JSON, got %v", err)
	}

	for _, test := range []struct{ name, old, replacement string }{
		{"unknown root", `"schema":`, `"extra":0,"schema":`},
		{"duplicate escaped", `"schema":`, `"\u0073chema":"sw-socioeconomic-data-v1","schema":`},
		{"unknown nested", `"minimum":21`, `"minimum":21,"other":0`},
		{"required missing", `"minimum":21`, `"other":21`},
		{"compiled forbidden", `"lowerBounds":`, `"compiled":[],"lowerBounds":`},
		{"extra age partition", `"agePartitions":{`, `"agePartitions":{"employeeIncome":{"lowerBounds":[21]},`},
		{"outcome absent", `"outcome":"industry",`, ``},
		{"null source", `"sourceModel":{`, `"sourceModel":null,"removedSourceModel":{`},
		{"null definition", `"code":"below-primary"`, `"code":"below-primary","definition":null`},
		{"boolean", `"companyCount":65536`, `"companyCount":true`},
		{"quoted integer", `"companyCount":65536`, `"companyCount":"65536"`},
		{"exponent", `"companyCount":65536`, `"companyCount":65536e0`},
		{"decimal", `"companyCount":65536`, `"companyCount":65536.0`},
		{"negative zero", `"decimalPlaces":0`, `"decimalPlaces":-0`},
		{"uint32 overflow", `"workforceCap":4096`, `"workforceCap":4294967296`},
		{"unknown law property", `"values":[0,1]`, `"values":[0,1],"extra":0`},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := bytes.Replace(encoded, []byte(test.old), []byte(test.replacement), 1)
			if bytes.Equal(input, encoded) {
				t.Fatal("expected fixture substitution, got unchanged bytes")
			}
			_, err := DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(input))
			if err != nil {
				return
			}
			t.Fatal("expected invalid full document rejected, got success")
		})
	}

}

// TestInternationalOriginalsRequired prevents conversion fallback or acceptance
// of sources whose physical identity differs from the prescribed originals.
func TestInternationalOriginalsRequired(t *testing.T) {

	t.Run("missing CSV", func(t *testing.T) {
		_, err := ConvertItalianBaseData(t.Context(), fstest.MapFS{})
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("expected missing-file error, got %v", err)
			}
			return
		}
		t.Fatal("expected missing source rejected, got embedded fallback")
	})

	t.Run("modified CSV", func(t *testing.T) {
		data, err := os.ReadFile("data/it-2026a/age_weights.csv")
		if err != nil {
			t.Fatalf("expected original CSV, got %v", err)
		}
		source := fstest.MapFS{"age_weights.csv": {Data: append(data, '\n')}}
		_, err = ConvertItalianBaseData(t.Context(), source)
		if err != nil {
			if !strings.Contains(err.Error(), "checksum mismatch") {
				t.Fatalf("expected checksum rejection, got %v", err)
			}
			return
		}
		t.Fatal("expected modified source rejected, got success")
	})

	t.Run("modified model", func(t *testing.T) {
		_, err := ConvertItalianSocioeconomicData(t.Context(), strings.NewReader("{}"))
		if err != nil {
			if !strings.Contains(err.Error(), "checksum mismatch") {
				t.Fatalf("expected canonical checksum rejection, got %v", err)
			}
			return
		}
		t.Fatal("expected modified model rejected, got success")
	})

}
