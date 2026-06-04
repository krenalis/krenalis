// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package base58

import "testing"

// TestIsValid reports whether IsValid accepts valid Base58 strings.
func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{"empty", "", true},
		{"alphabet", alphabet, true},
		{"valid", "3mJr7AoUXx2Wqd", true},
		{"zero", "abc0def", false},
		{"uppercase O", "abcOdef", false},
		{"uppercase I", "abcIdef", false},
		{"lowercase l", "abcldef", false},
		{"space", "abc def", false},
		{"newline", "abc\ndef", false},
		{"punctuation", "abc-def", false},
		{"unicode", "abcédef", false},
		{"null byte", "abc\x00def", false},
		{"invalid first", "0abc", false},
		{"invalid last", "abc0", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := IsValid(test.s)
			if got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}
}

// TestIsValidGenerated reports whether Generate returns strings accepted by IsValid.
func TestIsValidGenerated(t *testing.T) {
	for _, n := range []int{0, 1, 32, 1024} {
		s := Generate(n)
		if got := IsValid(s); got != true {
			t.Fatalf("expected true for Generate(%d), got %v", n, got)
		}
	}
}

// TestGenerate reports whether Generate returns a valid Base58 string of the
// requested length.
func TestGenerate(t *testing.T) {
	id := Generate(12)
	if len(id) != 12 {
		t.Fatalf("expected length 12, got %d", len(id))
	}
	if !IsValid(id) {
		t.Fatalf("expected valid base58 value, got %q", id)
	}
}

// TestGenerateNegative reports whether Generate panics when n is negative.
func TestGenerateNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = Generate(-1)
}

// TestGenerateZero reports whether Generate returns an empty string when n is zero.
func TestGenerateZero(t *testing.T) {
	id := Generate(0)
	if id != "" {
		t.Fatalf("expected empty value, got %q", id)
	}
}
