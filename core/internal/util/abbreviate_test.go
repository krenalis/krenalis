// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package util

import (
	"testing"
)

// Test_abbreviate checks that Abbreviate correctly shortens strings of varying
// lengths and handles multi-byte characters.
func Test_abbreviate(t *testing.T) {
	tests := []struct {
		s        string
		n        int
		expected string
	}{
		{"", 3, ""},
		{"12", 3, "12"},
		{"123", 3, "123"},
		{"1234", 3, "[…]"},
		{"1234567890", 3, "[…]"},
		{"1234567890", 4, "1[…]"},
		{"1234567890", 6, "123[…]"},
		{"1234567890", 9, "123456[…]"},
		{"1234567890", 10, "1234567890"},
		{"1234567890", 11, "1234567890"},
		{"世", 5, "世"},
		{"世😊🌍", 3, "世😊🌍"},
		{"世😊🌍 ", 3, "[…]"},
		{"世😊 🌍€𝒜 🚀あ", 6, "世😊 […]"},
		{"世😊 🌍€𝒜 🚀あ", 7, "世😊 🌍[…]"},
		{"世😊 🌍€𝒜 🚀あ", 8, "世😊 🌍€[…]"},
		{"世😊 🌍€𝒜 🚀あ", 9, "世😊 🌍€𝒜 🚀あ"},
		{"世😊 🌍€𝒜 🚀あ", 12, "世😊 🌍€𝒜 🚀あ"},
		{"Lorem ipsum dolor sit amet.", 28, "Lorem ipsum dolor sit amet."},
		{"Lorem ipsum dolor sit amet.", 24, "Lorem ipsum dolor sit[…]"},
		{"Lorem ipsum dolor sit amet.", 8, "Lorem[…]"},
	}
	for _, test := range tests {
		got := Abbreviate(test.s, test.n)
		if got != test.expected {
			t.Errorf("Abbreviate(%q, %d): expected %q, got %q", test.s, test.n, test.expected, got)
		}
	}
}

// Test_abbreviate_panic verifies that Abbreviate panics when asked to
// abbreviate below the minimum length of three runes.
func Test_abbreviate_panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		} else if r != "cannot abbreviate to fewer than 3 rune" {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	Abbreviate("abc", 2)
}
