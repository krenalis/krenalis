// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package dummy

import "testing"

// TestParseOperationDelay verifies operation delay validation.
func TestParseOperationDelay(t *testing.T) {
	tests := []struct {
		name    string
		delay   string
		wantErr string
	}{
		{
			name:  "single duration",
			delay: "500ms",
		},
		{
			name:  "minute duration",
			delay: "1m",
		},
		{
			name:  "range",
			delay: "2s-5m",
		},
		{
			name:    "range with invalid minute suffix",
			delay:   "2s-2min",
			wantErr: "maximum operation delay must be a valid duration",
		},
		{
			name:  "range with zero minimum",
			delay: "0s-2s",
		},
		{
			name:  "equal range",
			delay: "2s-2s",
		},
		{
			name:  "zero duration",
			delay: "0s",
		},
		{
			name:  "zero range",
			delay: "0s-0s",
		},
		{
			name:  "maximum duration",
			delay: "24h",
		},
		{
			name:  "range maximum duration",
			delay: "1s-24h",
		},
		{
			name:    "space before separator",
			delay:   "2s -5m",
			wantErr: "minimum operation delay must be a valid duration",
		},
		{
			name:    "space after separator",
			delay:   "2s- 5m",
			wantErr: "maximum operation delay must be a valid duration",
		},
		{
			name:    "too many separators",
			delay:   "2s-5s-7s",
			wantErr: "maximum operation delay must be a valid duration",
		},
		{
			name:    "missing minimum",
			delay:   "-5s",
			wantErr: "minimum operation delay must be a valid duration",
		},
		{
			name:    "missing maximum",
			delay:   "5s-",
			wantErr: "maximum operation delay must be a valid duration",
		},
		{
			name:    "inverted range",
			delay:   "5m-2s",
			wantErr: "minimum operational delay cannot be greater than maximum delay",
		},
		{
			name:    "negative duration",
			delay:   "-1s",
			wantErr: "minimum operation delay must be a valid duration",
		},
		{
			name:    "negative maximum",
			delay:   "1s--2s",
			wantErr: "maximum operation delay cannot be negative",
		},
		{
			name:    "too large duration",
			delay:   "25h",
			wantErr: "maximum operation delay must be less than or equal to 24h0m0s",
		},
		{
			name:    "range maximum too large",
			delay:   "1s-25h",
			wantErr: "maximum operation delay must be less than or equal to 24h0m0s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseOperationDelay(tt.delay)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
