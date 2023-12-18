//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"errors"
	"testing"
)

func Test_ReplacePlaceHolders(t *testing.T) {

	tests := []struct {
		s        string
		args     map[string]string
		expected string
		err      error
	}{
		{s: "", expected: ""},
		{s: "a text without placeholders", expected: "a text without placeholders"},
		{s: "${}", args: map[string]string{"": "missing"}, expected: "missing"},
		{s: "${   }", args: map[string]string{"": "missing"}, expected: "missing"},
		{s: "${name}", args: map[string]string{"name": "value"}, expected: "value"},
		{s: "${a}${b}${c}", args: map[string]string{"a": "1", "b": "2", "c": "3"}, expected: "123"},
		{s: "select * from ${limit}", args: map[string]string{"limit": "100"}, expected: "select * from 100"},
		{s: "select * from ${ limit }", args: map[string]string{"limit": "100"}, expected: "select * from 100"},
		{s: "users_${date}.csv", args: map[string]string{"date": "2023-11-09"}, expected: "users_2023-11-09.csv"},
		{s: "users_${ date }.csv", args: map[string]string{"date": "2023-11-09"}, expected: "users_2023-11-09.csv"},
		{s: "users_${ date time }.csv", args: map[string]string{"date time": "2023-11-09T12:22:31"}, expected: "users_2023-11-09T12:22:31.csv"},
		{s: "users_$ {date}.csv", args: map[string]string{"date": "2023-11-09"}, expected: "users_$ {date}.csv"},
		{s: "users_${date.csv", args: map[string]string{"date": "2023-11-09"}, err: errors.New("a placeholder is not closed")},
		{s: "users_${ ${ date.csv", args: map[string]string{"date": "2023-11-09"}, err: errors.New("a placeholder is not closed")},
		{s: "users_${ ${date.csv} $}", args: map[string]string{"date": "2023-11-09"}, err: errors.New("a placeholder is not closed")},
		{s: "users_date}.csv", args: map[string]string{"date": "2023-11-09"}, expected: "users_date}.csv"},
		{s: "users_{date}.csv", args: map[string]string{"date": "2023-11-09"}, expected: "users_{date}.csv"},
		{s: "users_${date}T${time}.csv", args: map[string]string{"date": "2023-11-09", "time": "12:22:31"}, expected: "users_2023-11-09T12:22:31.csv"},
		{s: "users_${yesterday}.csv", args: map[string]string{"today": "2023-11-09"}, err: errors.New("placeholder \"yesterday\" does not exist")},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			replacer := func(name string) (string, bool) {
				v, ok := test.args[name]
				return v, ok
			}
			got, err := replacePlaceholders(test.s, replacer)
			if err != nil {
				if test.err == nil {
					t.Fatalf("exepcted no errors, got error %s", err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("expected error %q, got error %q", test.err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("exepcted error %q, got no errors", test.err)
			}
			if got != test.expected {
				t.Fatalf("exepcted %q, got %s", test.expected, got)
			}
		})
	}

}

func Test_validateTimestampFormat(t *testing.T) {
	tests := []struct {
		format string
		err    string
	}{
		// Valid.
		{format: "'%Y'"},
		{format: "Excel"},
		{format: "ISO8601"},

		// Invalid.
		{format: "%Y", err: `invalid timestamp format "%Y"`},
		{format: "'%Y", err: `invalid timestamp format "'%Y"`},
		{format: "%Y'", err: `invalid timestamp format "%Y'"`},
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
