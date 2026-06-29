// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package dummy

import "testing"

// TestParseOperationDelay verifies operation delay validation and
// canonicalization.
func TestParseOperationDelay(t *testing.T) {
	tests := []struct {
		name    string
		delay   string
		want    string
		wantErr bool
	}{
		{
			name:  "single duration",
			delay: "500ms",
			want:  "500ms",
		},
		{
			name:  "canonicalizes single duration",
			delay: "1m",
			want:  "1m0s",
		},
		{
			name:  "range",
			delay: "2s-5m",
			want:  "2s-5m0s",
		},
		{
			name:  "range with zero minimum",
			delay: "0s-2s",
			want:  "0s-2s",
		},
		{
			name:  "equal range becomes single duration",
			delay: "2s-2s",
			want:  "2s",
		},
		{
			name:  "zero duration becomes empty",
			delay: "0s",
			want:  "",
		},
		{
			name:  "zero range becomes empty",
			delay: "0s-0s",
			want:  "",
		},
		{
			name:  "maximum duration",
			delay: "24h",
			want:  "24h0m0s",
		},
		{
			name:  "range maximum duration",
			delay: "1s-24h",
			want:  "1s-24h0m0s",
		},
		{
			name:    "space before separator",
			delay:   "2s -5m",
			wantErr: true,
		},
		{
			name:    "space after separator",
			delay:   "2s- 5m",
			wantErr: true,
		},
		{
			name:    "too many separators",
			delay:   "2s-5s-7s",
			wantErr: true,
		},
		{
			name:    "missing minimum",
			delay:   "-5s",
			wantErr: true,
		},
		{
			name:    "missing maximum",
			delay:   "5s-",
			wantErr: true,
		},
		{
			name:    "inverted range",
			delay:   "5m-2s",
			wantErr: true,
		},
		{
			name:    "negative duration",
			delay:   "-1s",
			wantErr: true,
		},
		{
			name:    "negative maximum",
			delay:   "1s--2s",
			wantErr: true,
		},
		{
			name:    "too large duration",
			delay:   "25h",
			wantErr: true,
		},
		{
			name:    "range maximum too large",
			delay:   "1s-25h",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOperationDelay(tt.delay)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("parseOperationDelay(%q) = %q, want %q", tt.delay, got, tt.want)
			}
		})
	}
}
