// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package validation

import (
	"testing"
)

func TestParseBearer(t *testing.T) {

	tests := []struct {
		name   string
		header string
		token  string
		ok     bool
	}{
		{
			name:   "valid bearer token",
			header: "Bearer abc123",
			token:  "abc123",
			ok:     true,
		},
		{
			name:   "valid mixed-case scheme",
			header: "bEaReR token",
			token:  "token",
			ok:     true,
		},
		{
			name:   "valid tab separator",
			header: "Bearer\tabc",
			token:  "abc",
			ok:     true,
		},
		{
			name:   "valid multiple whitespace",
			header: "Bearer \t  abc",
			token:  "abc",
			ok:     true,
		},
		{
			name:   "valid token with spaces",
			header: "Bearer abc def",
			token:  "abc def",
			ok:     true,
		},
		{
			name:   "valid token preserves trailing space",
			header: "Bearer abc ",
			token:  "abc ",
			ok:     true,
		},
		{
			name:   "empty header",
			header: "",
			ok:     false,
		},
		{
			name:   "wrong scheme",
			header: "Basic abc",
			ok:     false,
		},
		{
			name:   "short scheme",
			header: "Bear abc",
			ok:     false,
		},
		{
			name:   "leading space before scheme",
			header: " Bearer abc",
			ok:     false,
		},
		{
			name:   "no separator",
			header: "Bearerabc",
			ok:     false,
		},
		{
			name:   "non-whitespace separator",
			header: "Bearer:abc",
			ok:     false,
		},
		{
			name:   "scheme only",
			header: "Bearer",
			ok:     false,
		},
		{
			name:   "scheme with space only",
			header: "Bearer ",
			ok:     false,
		},
		{
			name:   "scheme with tabs only",
			header: "Bearer\t\t",
			ok:     false,
		},
		{
			name:   "newline separator",
			header: "Bearer\nabc",
			ok:     false,
		},
		{
			name:   "prefix match with extra char",
			header: "BearerX abc",
			ok:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseBearer(test.header)
			if ok != test.ok {
				t.Fatalf("expected %t, got %t", test.ok, ok)
			}
			if ok && got != test.token {
				t.Fatalf("expected %q, got %q", test.token, got)
			}
		})
	}

}
