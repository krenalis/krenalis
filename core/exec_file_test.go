// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"

	"github.com/meergo/meergo/core/internal/connections"
)

func Test_newPathPlaceholderReplacer(t *testing.T) {
	now := time.Date(2035, 10, 30, 16, 33, 25, 0, time.UTC)
	tests := []struct {
		path     string
		expected string
		err      string
	}{

		// Valid.
		{path: "/files/users/${ today }.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/${ now }.csv", expected: "/files/users/2035-10-30-16-33-25.csv"},
		{path: "/files/users/${ unix }.csv", expected: "/files/users/2077374805.csv"},
		{path: "/files/users/${ UNIX }.csv", expected: "/files/users/2077374805.csv"},
		{path: "/files/users/${ Today }.csv", expected: "/files/users/2035-10-30.csv"},
		{path: "/files/users/${   Now }.csv", expected: "/files/users/2035-10-30-16-33-25.csv"},

		// Errors.
		{path: "${ today }} ${ yesterday }", err: "placeholder \"yesterday\" does not exist"},
		{path: "${ today }} ${ YESTERDAY }", err: "placeholder \"YESTERDAY\" does not exist"},
		{path: "${ invalid1 }} ${ invalid2 }", err: "placeholder \"invalid1\" does not exist"},
		{path: "/files/users/${ yesterday }.csv", err: "placeholder \"yesterday\" does not exist"},
	}
	replacer := newPathPlaceholderReplacer(now)
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			// This test here tests the "newPathPlaceholderReplacer" function,
			// and assumes that connections.ReplacePlaceholders is correct and
			// already tested elsewhere.
			got, gotErr := connections.ReplacePlaceholders(test.path, replacer)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if test.err != gotErrStr {
				t.Fatalf("expected error %q, got %q", test.err, gotErrStr)
			}
			if test.expected != got {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}
}
