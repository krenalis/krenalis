// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connectors

import (
	"errors"
	"testing"
	"time"
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
			got, err := ReplacePlaceholders(test.s, replacer)
			if err != nil {
				if test.err == nil {
					t.Fatalf("expected no errors, got error %s", err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("expected error %q, got error %q", test.err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expected error %q, got no errors", test.err)
			}
			if got != test.expected {
				t.Fatalf("expected %q, got %s", test.expected, got)
			}
		})
	}

}

func Test_parseLastChangeTimeColumnWithFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		value       string
		expected    time.Time
		expectedErr string
	}{
		{
			name:     "ISO8601",
			format:   "ISO8601",
			value:    "2033-12-14T13:52Z",
			expected: time.Date(2033, 12, 14, 13, 52, 0, 0, time.UTC),
		},
		{
			name:     "ISO8601 with milliseconds",
			format:   "ISO8601",
			value:    "2033-12-14T13:52:45.678Z",
			expected: time.Date(2033, 12, 14, 13, 52, 45, 678000000, time.UTC),
		},
		{
			name:     "ISO8601 with nanoseconds",
			format:   "ISO8601",
			value:    "2033-12-14T13:52:45.123456789Z",
			expected: time.Date(2033, 12, 14, 13, 52, 45, 123456789, time.UTC),
		},
		{
			name:     "ISO8601 date only",
			format:   "ISO8601",
			value:    "2033-12-14",
			expected: time.Date(2033, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "ISO8601 - wrong value format",
			format:      "ISO8601",
			value:       "2033-12-14T13-",
			expectedErr: "last change time does not conform to the ISO8601 format",
		},
		{
			name:     "Excel",
			format:   "Excel",
			value:    "39448",
			expected: time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Excel - invalid timestamp (1)",
			format:      "Excel",
			value:       "2000-01-02",
			expectedErr: "last change time does not conform to the Excel format",
		},
		{
			name:        "Excel - invalid timestamp (2)",
			format:      "Excel",
			value:       "12.34.45",
			expectedErr: "last change time does not conform to the Excel format",
		},
		{
			name:     "Excel - only date",
			format:   "Excel",
			value:    "39448",
			expected: time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Excel - date and time",
			format:   "Excel",
			value:    "39052.5315393519",
			expected: time.Date(2006, 12, 1, 12, 45, 25, 0, time.UTC),
		},
		{
			name:     "strptime - only date (1)",
			format:   "%d %m %Y",
			value:    "14 12 2033",
			expected: time.Date(2033, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "strptime - only date (2)",
			format:   "%Y-%m-%d",
			value:    "2033-12-14",
			expected: time.Date(2033, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "strptime - date and time",
			format:   "%Y-%m-%d %H:%M:%S",
			value:    "2033-12-14 09:56:35",
			expected: time.Date(2033, 12, 14, 9, 56, 35, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseLastChangeTimeColumnWithFormat(test.format, test.value)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if test.expectedErr != gotErr {
				t.Fatalf("expected error %q, got %q", test.expectedErr, gotErr)
			}
			if test.expected != got {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}
}
