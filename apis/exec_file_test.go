//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"testing"
	"time"
)

func Test_replacePlaceholders(t *testing.T) {
	now := time.Date(2035, 10, 30, 16, 33, 25, 0, time.UTC)
	tests := []struct {
		path     string
		expected string
		err      string
	}{

		// Valid.
		{path: "", expected: ""},
		{path: "{", expected: "{"},
		{path: "}", expected: "}"},
		{path: "{{", expected: "{{"},
		{path: "}}", expected: "}}"},
		{path: "{{unix}}", expected: "2077374805"},
		{path: "{{ unix }}", expected: "2077374805"},
		{path: "{{\tunix\t}}", expected: "2077374805"},
		{path: "{{\nunix\n}}", expected: "2077374805"},
		{path: "{{ today } }", expected: "{{ today } }"},
		{path: "{ { today }}", expected: "{ { today }}"},
		{path: "{ { today } }", expected: "{ { today } }"},
		{path: "/path/to/file.csv", expected: "/path/to/file.csv"},
		{path: "{{    \t  unix \t     }}", expected: "2077374805"},
		{path: "/files/users/{{today}}.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/{{ now }}.csv", expected: "/files/users/2035-10-30-16-33-25.csv"},
		{path: "/files/users/{{ unix }}.csv", expected: "/files/users/2077374805.csv"},
		{path: "/files/users/{{ UNIX }}.csv", expected: "/files/users/2077374805.csv"},
		{path: "/files/users/{{ today }}.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/{{\ttoday }}.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/{{ Today }}.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/{{   Now }}.csv", expected: "/files/users/2035-10-30-16-33-25.csv"},
		{path: "/files/users/{ { today }}.csv", expected: "/files/users/{ { today }}.csv"},
		{path: "{{  \t  \n \t\n unix \t\t\n  \t   }}", expected: "2077374805"},
		{path: "/files/users/{{      today    }}.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/{{today}}{{ today }}.csv", expected: "/files/users/2035-10-302035-10-30.csv"},

		// Errors.
		{path: "{{}}", err: "invalid placeholder: {{}}"},
		{path: "{{ }}", err: "invalid placeholder: {{ }}"},
		{path: "{{ _ }}", err: "invalid placeholder: {{ _ }}"},
		{path: "{{ \xa0\xa1 }}", err: "invalid placeholder: {{ \xa0\xa1 }}"},
		{path: "{{ {{ today }} }}", err: "invalid placeholder: {{ 2035-10-30 }}"},
		{path: "{{ today }} {{ yesterday }}", err: "invalid placeholder: {{ yesterday }}"},
		{path: "{{ today }} {{ YESTERDAY }}", err: "invalid placeholder: {{ YESTERDAY }}"},
		{path: "/files/users/{{ un ix }}.csv", err: "invalid placeholder: {{ un ix }}"},
		{path: "{{ invalid1 }} {{ invalid2 }}", err: "invalid placeholder: {{ invalid1 }}"},
		{path: "/files/users/{{ yesterday }}.csv", err: "invalid placeholder: {{ yesterday }}"},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, gotErr := replacePathPlaceholders(test.path, now)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if test.err != gotErrStr {
				t.Fatalf("expecting error %q, got %q", test.err, gotErrStr)
			}
			if test.expected != got {
				t.Fatalf("expecting %q, got %q", test.expected, got)
			}
		})
	}
}
