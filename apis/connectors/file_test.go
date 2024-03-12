//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"testing"
	"time"
)

func Test_parseTimestamp(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		value       string
		expected    time.Time
		expectedErr string
	}{
		{
			name:     "DateTime",
			format:   "DateTime",
			value:    "2033-12-14 13:52:00",
			expected: time.Date(2033, 12, 14, 13, 52, 0, 0, time.UTC),
		},
		{
			name:        "DateTime but an empty string is passed",
			format:      "DateTime",
			value:       "",
			expectedErr: `timestamp has not the format '2006-01-02 15:04:05'`,
		},
		{
			name:     "DateOnly",
			format:   "DateOnly",
			value:    "2033-12-14",
			expected: time.Date(2033, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "DateOnly but hour-minute-second is passed",
			format:      "DateOnly",
			value:       "2033-12-14 13:32:12",
			expectedErr: `timestamp has not the format '2006-01-02'`,
		},
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
			expectedErr: "timestamp format is not compatible with ISO 8601",
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
			expectedErr: "invalid timestamp for Excel",
		},
		{
			name:        "Excel - invalid timestamp (2)",
			format:      "Excel",
			value:       "12.34.45",
			expectedErr: "invalid timestamp for Excel",
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
			format:   "'%d %m %Y'",
			value:    "14 12 2033",
			expected: time.Date(2033, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "strptime - only date (2)",
			format:   "'%Y-%m-%d'",
			value:    "2033-12-14",
			expected: time.Date(2033, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "strptime - date and time",
			format:   "'%Y-%m-%d %H:%M:%S'",
			value:    "2033-12-14 09:56:35",
			expected: time.Date(2033, 12, 14, 9, 56, 35, 0, time.UTC),
		},
		{
			name: "strptime - format and value are enclosed in single quotes and that should be valid",
			// Note the "double" single quotes: a single quote surrounding the
			// format, and a single quote which is part of the format itself.
			format:   `''%Y-%m-%d''`,
			value:    "'2033-12-14'",
			expected: time.Date(2033, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Invalid format (1)",
			format:      "",
			value:       "2033-12-14T13:52Z",
			expectedErr: `invalid format ""`,
		},
		{
			name:        "Invalid format (2)",
			format:      "iso8601", // must be uppercase.
			value:       "2033-12-14T13:52Z",
			expectedErr: `invalid format "iso8601"`,
		},
		{
			name:        "Invalid format (2)",
			format:      `"%Y-%m-%d"`,
			value:       "2033-12-14",
			expectedErr: `invalid format "\"%Y-%m-%d\""`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseTimestamp(test.format, test.value)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if test.expectedErr != gotErr {
				t.Fatalf("expecting error %q, got %q", test.expectedErr, gotErr)
			}
			if test.expected != got {
				t.Fatalf("expecting %v, got %v", test.expected, got)
			}
		})
	}
}
