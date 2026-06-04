// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"
)

// Test_quoteStringForDynamicSQL verifies quoting for nested SQL strings.
func Test_quoteStringForDynamicSQL(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "empty",
			want: "''''",
		},
		{
			name: "base58 identifier",
			s:    "6NpT4zB8QaR2",
			want: "''6NpT4zB8QaR2''",
		},
		{
			name: "single quote",
			s:    "paul's car",
			want: "''paul\\''s car''",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got strings.Builder
			quoteStringForDynamicSQL(&got, test.s)
			if got.String() != test.want {
				t.Fatalf("expected %q, got %q", test.want, got.String())
			}
		})
	}
}
