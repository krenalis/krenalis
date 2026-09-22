// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import "testing"

func TestQuoteErrorTerm(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain",
			in:   "metadata",
			want: "«metadata»",
		},
		{
			name: "path",
			in:   "address.first_name",
			want: "«address.first_name»",
		},
		{
			name: "contains closing quote",
			in:   "foo»bar",
			want: "«foo≫bar»",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QuoteErrorTerm(tt.in); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
