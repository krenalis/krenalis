// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	stdjson "encoding/json"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestNumericTransportFixtures checks payload bounds and literal keys.
func TestNumericTransportFixtures(t *testing.T) {
	cases := NumericTransportCases()
	if len(cases) == 0 || len(cases) > 64 {
		t.Fatalf("expected bounded fixture population, got %d", len(cases))
	}
	for id, fixture := range cases {
		row := fixture.Row(id)
		for i, value := range row[1:] {
			if fixture.Number == "" {
				if value != nil {
					t.Fatalf("expected SQL NULL fixture, got %T", value)
				}
				continue
			}
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("expected marshalable numeric fixture, got %v", err)
			}
			err = CheckNumericTransportJSON(string(raw), string(raw), true)
			if err != nil {
				t.Fatalf("expected bounded valid JSON fixture, got %v", err)
			}
			var token stdjson.RawMessage
			if i == 1 {
				var elements []stdjson.RawMessage
				err := stdjson.Unmarshal(raw, &elements)
				if err != nil {
					t.Fatalf("expected array fixture, got %v", err)
				}
				if len(elements) != 3 {
					t.Fatalf("expected three array elements, got %d", len(elements))
				}
				token = elements[0]
			} else {
				var fields map[string]stdjson.RawMessage
				err := stdjson.Unmarshal(raw, &fields)
				if err != nil {
					t.Fatalf("expected object fixture, got %v", err)
				}
				if string(fields["text"]) != `"1E2"` || string(fields["null"]) != "null" {
					t.Fatal("expected preserved text and explicit null, got altered fixture")
				}
				token = fields["n"]
			}
			if string(token) != fixture.Number {
				t.Fatalf("expected source token %s in column %d, got %s", fixture.Number, i, token)
			}
		}
	}
}

// TestNumericTransportJSON distinguishes token preservation from exact numbers.
func TestNumericTransportJSON(t *testing.T) {
	for _, test := range []struct {
		name, got, want string
		tokens, valid   bool
	}{
		{"fixed spelling", "1E2", "100.00", false, true},
		{"fixed precision", "12345678901234567890123456789012345678", "12345678901234567890123456789012345679", false, false},
		{"fixed scale", "1E-37", "0.0000000000000000000000000000000000001", false, true},
		{"token differs numerically", "0.1", "0.10000000000000001", true, false},
		{"token differs only lexically", "0.1", "0.10", true, false},
		{"numeric equivalence", "0.1", "0.10", false, true},
		{"token identical", "1.000000000000000e+02", "1.000000000000000e+02", true, true},
		{"nested tokens", `{"a.b":[1E2,{"n":0.1}],"null":null}`, `{"null":null,"a.b":[1E2,{"n":0.1}]}`, true, true},
		{"nested numeric change", `{"n":[1E2]}`, `{"n":[100]}`, true, false},
		{"numeric string distinct", `"1E2"`, "100", false, false},
		{"missing distinct from null", `{}`, `{"n":null}`, true, false},
		{"invalid JSON", "NaN", "null", true, false},
		{"oversized", strings.Repeat(" ", 16*1024) + "0", "0", true, false},
		{"large exponent bounded", "1E999999999999999999", "1.0E999999999999999999", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := CheckNumericTransportJSON(test.got, test.want, test.tokens)
			if err != nil {
				if test.valid {
					t.Fatalf("expected preserved fixture value, got %v", err)
				}
				return
			}
			if !test.valid {
				t.Fatal("expected changed or invalid fixture rejection, got success")
			}
		})
	}
}

// TestNumericTransportJSONDiagnostics checks useful, bounded numeric differences.
func TestNumericTransportJSONDiagnostics(t *testing.T) {
	for _, test := range []struct {
		name, got, want, message string
	}{
		{"scalar", "1E2", "100", `expected token "100", got "1E2"`},
		{"nested", `{"n":[1E2,2]}`, `{"n":[100,2]}`, `expected token "100", got "1E2"`},
		{"precision", "12345678901234567890123456789012345678", "12345678901234567890123456789012345679", `expected token "12345678901234567890123456789012345679", got "12345678901234567890123456789012345678"`},
		{"oversized token", strings.Repeat("9", 401), "1", `expected token "1", got "` + strings.Repeat("9", 397) + `..."`},
		{"untrusted text", `{"n":"private data"}`, `{"n":1}`, "numeric JSON differs"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := CheckNumericTransportJSON(test.got, test.want, true)
			if err != nil {
				if !strings.Contains(err.Error(), test.message) || len(err.Error()) > 900 ||
					strings.Contains(err.Error(), "private data") {
					t.Fatalf("expected bounded numeric diagnostic containing %q, got %v", test.message, err)
				}
				return
			}
			t.Fatal("expected JSON difference diagnostic, got success")
		})
	}
}
