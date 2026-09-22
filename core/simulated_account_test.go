// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

func TestValidateSimulatedAccountConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		userCount   json.Value
		duplicate   json.Value
		countries   json.Value
		wantErrText string
	}{
		{name: "empty defaults", userCount: json.Value("0")},
		{name: "empty", userCount: json.Value("0"), countries: json.Value("{}")},
		{name: "IT", userCount: json.Value("0"), countries: json.Value(`{"IT": 40}`)},
		{name: "one", userCount: json.Value("1"), countries: json.Value(`{"IT": 40}`)},
		{name: "fractional percent", userCount: json.Value("101"), duplicate: json.Value("10.01"), countries: json.Value(`{"IT": 40}`)},
		{name: "one million", userCount: json.Value("1000000"), duplicate: json.Value("50"), countries: json.Value(`{"IT": 100}`)},
		{name: "missing user count", countries: json.Value(`{"IT": 100}`), wantErrText: "userCount is required"},
		{name: "missing countries", userCount: json.Value("1"), countries: json.Value("{}"), wantErrText: "countries must contain IT"},
		{name: "negative user count", userCount: json.Value("-1"), countries: json.Value(`{"IT": 100}`), wantErrText: "userCount must be between"},
		{name: "fractional user count", userCount: json.Value("0.5"), countries: json.Value(`{"IT": 100}`), wantErrText: "userCount must be an integer"},
		{name: "too many users", userCount: json.Value("1000001"), countries: json.Value(`{"IT": 100}`), wantErrText: "userCount must be between"},
		{name: "duplicate percent for empty", userCount: json.Value("0"), duplicate: json.Value("1"), countries: json.Value("{}"), wantErrText: "duplicateRecordPercent must be 0"},
		{name: "duplicate percent above limit", userCount: json.Value("1"), duplicate: json.Value("50.01"), countries: json.Value(`{"IT": 100}`), wantErrText: "duplicateRecordPercent must be between"},
		{name: "duplicate percent negative", userCount: json.Value("1"), duplicate: json.Value("-0.01"), countries: json.Value(`{"IT": 100}`), wantErrText: "duplicateRecordPercent must be between"},
		{name: "duplicate percent excessive precision", userCount: json.Value("1"), duplicate: json.Value("10.001"), countries: json.Value(`{"IT": 100}`), wantErrText: "duplicateRecordPercent is not valid"},
		{name: "null duplicate percent", userCount: json.Value("0"), duplicate: json.Value("null"), countries: json.Value("{}"), wantErrText: "duplicateRecordPercent must be a number"},
		{name: "unsupported country", userCount: json.Value("1"), countries: json.Value(`{"FR": 40}`), wantErrText: `country "FR" is not supported`},
		{name: "fractional country weight", userCount: json.Value("0"), countries: json.Value(`{"IT": 1.5}`), wantErrText: "weight must be an integer"},
		{name: "invalid country weight", userCount: json.Value("0"), countries: json.Value(`{"IT": 101}`), wantErrText: "weight must be an integer"},
		{name: "null countries", userCount: json.Value("0"), countries: json.Value("null"), wantErrText: "countries must be an object"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, err := validateSimulatedAccountConfiguration(test.userCount, test.duplicate, test.countries)
			if test.wantErrText == "" {
				if err != nil {
					t.Fatalf("expected valid configuration, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got success", test.wantErrText)
			}
			if !strings.Contains(err.Error(), test.wantErrText) {
				t.Fatalf("expected error containing %q, got %q", test.wantErrText, err)
			}
		})
	}
}
