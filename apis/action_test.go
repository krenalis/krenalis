//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

import "testing"

func Test_validateTimestampFormat(t *testing.T) {
	tests := []struct {
		format string
		err    string
	}{
		// Valid.
		{format: "DateTime"},
		{format: "DateOnly"},
		{format: "'%Y'"},
		{format: "Excel"},
		{format: "ISO8601"},
		{format: "'%'"},

		// Invalid.
		{format: "'", err: `timestamp format "'" is not a valid timestamp format`},
		{format: "'''", err: `timestamp format "'''" is not a valid timestamp format`},
		{format: "", err: "timestamp format cannot be empty"},
		{format: "Date", err: `timestamp format "Date" is not a valid timestamp format`},
		{format: "excel", err: `timestamp format "excel" is not a valid timestamp format`},
		{format: "iso8601", err: `timestamp format "iso8601" is not a valid timestamp format`},
		{format: "%Y", err: `timestamp strptime format must be enclosed between "'" characters`},
		{format: "'%Y", err: `timestamp strptime format must be enclosed between "'" characters`},
		{format: "%Y'", err: `timestamp strptime format must be enclosed between "'" characters`},
		{format: "\xc3\x28", err: "timestamp format must be UTF-8 valid"},
		{format: "''", err: `timestamp format "''" is not a valid timestamp format`},
		{format: "'YYYY-MM-DD'", err: `timestamp format "'YYYY-MM-DD'" is not a valid timestamp format`},
		{format: "'%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y'", err: "timestamp format is longer than 64 runes"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			got := validateTimestampFormat(test.format)
			var gotStr string
			if got != nil {
				gotStr = got.Error()
			}
			if test.err != gotStr {
				t.Fatalf("expecting %q, got %q", test.err, gotStr)
			}
		})
	}
}
