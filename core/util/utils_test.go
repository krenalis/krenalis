//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package util

import (
	"strings"
	"testing"
)

func TestValidateStringField(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maxLen  int
		wantErr bool
		errSub  string
	}{
		{
			name:    "Empty string",
			input:   "",
			maxLen:  5,
			wantErr: true,
			errSub:  "is empty",
		},
		{
			name:    "Invalid UTF-8",
			input:   string([]byte{0xff, 0xfe, 0xfd}),
			maxLen:  5,
			wantErr: true,
			errSub:  "invalid UTF-8",
		},
		{
			name:    "Contains NUL byte",
			input:   "foo\x00bar",
			maxLen:  10,
			wantErr: true,
			errSub:  "contains the NUL byte",
		},
		{
			name:    "Too many runes",
			input:   "abcdef",
			maxLen:  3,
			wantErr: true,
			errSub:  "longer than",
		},
		{
			name:    "Valid short string",
			input:   "abc世𠜎",
			maxLen:  10,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringField("field", tt.input, tt.maxLen)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("wrong error message: got %q, want it to contain %q", err.Error(), tt.errSub)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			}
		})
	}
}
