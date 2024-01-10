//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package errors

import (
	"testing"
)

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
