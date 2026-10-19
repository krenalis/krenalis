// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// TestNewOperationError verifies that operation error messages remain valid
// UTF-8 and are abbreviated without exceeding the supported size.
func TestNewOperationError(t *testing.T) {

	const suffix = "[...]"
	const invalidMessage = "warehouse operation failed with an invalid error message"
	maxBytes := warehouses.MaxOperationErrorBytes
	exactUTF8 := strings.Repeat("a", maxBytes-2) + "é"
	unicodePrefix := strings.Repeat("a", maxBytes-len(suffix)-1)
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{name: "short", message: "warehouse failure", want: "warehouse failure"},
		{name: "exact UTF-8 limit", message: exactUTF8, want: exactUTF8},
		{
			name:    "oversized ASCII",
			message: strings.Repeat("x", maxBytes+1),
			want:    strings.Repeat("x", maxBytes-len(suffix)) + suffix,
		},
		{
			name:    "UTF-8 boundary",
			message: unicodePrefix + "é" + strings.Repeat("b", 10),
			want:    unicodePrefix + suffix,
		},
		{name: "invalid UTF-8", message: string([]byte{'a', 0xff, 'b'}), want: invalidMessage},
		{
			name:    "oversized invalid UTF-8",
			message: strings.Repeat(string([]byte{0x80}), maxBytes+1),
			want:    invalidMessage,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := warehouses.NewOperationError(errors.New(test.message)).Error()
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("expected valid UTF-8, got %q", got)
			}
			if len(got) > maxBytes {
				t.Fatalf("expected at most %d bytes, got %d", maxBytes, len(got))
			}
		})
	}

}

// TestNewPersistedOperationError verifies that invalid or oversized persisted
// operation error messages are replaced with a fixed message rather than
// normalized.
func TestNewPersistedOperationError(t *testing.T) {

	const invalidMessage = "warehouse operation failed with an invalid or oversized error message"
	maxSizeMessage := strings.Repeat("x", warehouses.MaxOperationErrorBytes)
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{name: "valid", message: "warehouse failure", want: "warehouse failure"},
		{name: "maximum size", message: maxSizeMessage, want: maxSizeMessage},
		{
			name:    "oversized",
			message: strings.Repeat("x", warehouses.MaxOperationErrorBytes+1),
			want:    invalidMessage,
		},
		{name: "invalid UTF-8", message: string([]byte{'a', 0xff, 'b'}), want: invalidMessage},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := warehouses.NewPersistedOperationError(test.message).Error()
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}

}
