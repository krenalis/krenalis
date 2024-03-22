//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"testing"

	"chichi/types"
)

func Test_isMetaProperty(t *testing.T) {
	tests := []struct {
		p        types.Property
		expected bool
	}{
		{types.Property{}, false}, // invalid property, shouldn't happen.
		{types.Property{Name: "a", Type: types.Int(32)}, false},
		{types.Property{Name: "hello", Type: types.Int(32)}, false},
		{types.Property{Name: "Hello", Type: types.Int(32)}, true},
		{types.Property{Name: "HeyTest", Type: types.Int(32)}, true},
	}
	for _, test := range tests {
		got := isMetaProperty(test.p.Name)
		if test.expected != got {
			t.Errorf("%#v: expected %t, got %t", test.p, test.expected, got)
		}
	}
}

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

		// Invalid.
		{format: "%Y", err: `invalid timestamp format "%Y"`},
		{format: "'%Y", err: `invalid timestamp format "'%Y"`},
		{format: "%Y'", err: `invalid timestamp format "%Y'"`},
		{format: "Date", err: `invalid timestamp format "Date"`},
		{format: "excel", err: `invalid timestamp format "excel"`},
		{format: "iso8601", err: `invalid timestamp format "iso8601"`},
		{format: "\xc3\x28", err: "timestamp format must be UTF-8 valid"},
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
