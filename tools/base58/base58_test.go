// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package base58

import (
	"bytes"
	"testing"
)

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

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		src  []byte
		want string
	}{
		{
			name: "empty",
			src:  []byte{},
			want: "",
		},
		{
			name: "nil",
			src:  nil,
			want: "",
		},
		{
			name: "single zero",
			src:  []byte{0x00},
			want: "1",
		},
		{
			name: "multiple zeroes",
			src:  []byte{0x00, 0x00, 0x00},
			want: "111",
		},
		{
			name: "single one",
			src:  []byte{0x01},
			want: "2",
		},
		{
			name: "single fifty seven",
			src:  []byte{57},
			want: "z",
		},
		{
			name: "single fifty eight",
			src:  []byte{58},
			want: "21",
		},
		{
			name: "leading zeroes",
			src:  []byte{0x00, 0x00, 0x00, 0x01},
			want: "1112",
		},
		{
			name: "hello world",
			src:  []byte("hello world"),
			want: "StV1DL6CwTryKyV",
		},
		{
			name: "hello world with leading zeroes",
			src:  append([]byte{0x00, 0x00}, []byte("hello world")...),
			want: "11StV1DL6CwTryKyV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Encode(tt.src)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []byte
	}{
		{
			name: "empty",
			s:    "",
			want: []byte{},
		},
		{
			name: "single zero",
			s:    "1",
			want: []byte{0x00},
		},
		{
			name: "multiple zeroes",
			s:    "111",
			want: []byte{0x00, 0x00, 0x00},
		},
		{
			name: "single one",
			s:    "2",
			want: []byte{0x01},
		},
		{
			name: "single fifty seven",
			s:    "z",
			want: []byte{57},
		},
		{
			name: "single fifty eight",
			s:    "21",
			want: []byte{58},
		},
		{
			name: "leading zeroes",
			s:    "1112",
			want: []byte{0x00, 0x00, 0x00, 0x01},
		},
		{
			name: "hello world",
			s:    "StV1DL6CwTryKyV",
			want: []byte("hello world"),
		},
		{
			name: "hello world with leading zeroes",
			s:    "11StV1DL6CwTryKyV",
			want: append([]byte{0x00, 0x00}, []byte("hello world")...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.s)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestDecodeInvalid(t *testing.T) {
	tests := []string{
		"0",
		"O",
		"I",
		"l",
		"abc0",
		"abcO",
		"abcI",
		"abcl",
		" ",
		"\n",
		"é",
	}

	for _, s := range tests {
		_, err := Decode(s)
		if err == nil {
			t.Fatalf("expected error for %q, got nil", s)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	tests := [][]byte{
		nil,
		{},
		{0x00},
		{0x00, 0x00},
		{0x01},
		{0x39},
		{0x3a},
		{0xff},
		[]byte("hello world"),
		{0x00, 0x00, 0x01, 0x02, 0x03, 0xff},
	}

	for n := 0; n <= 64; n++ {
		src := make([]byte, n)
		for i := range src {
			src[i] = byte((i*31 + n*17) % 256)
		}
		if n >= 3 {
			src[0] = 0x00
			src[1] = 0x00
		}
		tests = append(tests, src)
	}

	for _, src := range tests {
		encoded := Encode(src)

		got, err := Decode(encoded)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("expected %v, got %v", src, got)
		}
	}
}

func TestEncodeOutputIsValid(t *testing.T) {
	for n := 0; n <= 64; n++ {
		src := make([]byte, n)
		for i := range src {
			src[i] = byte((i*17 + 23) % 256)
		}

		got := Encode(src)
		if !IsValid(got) {
			t.Fatalf("expected valid Base58 string, got %q", got)
		}
	}
}

func TestAlphabetCompatibility(t *testing.T) {
	if len(alphabet) != 58 {
		t.Fatalf("expected alphabet length 58, got %d", len(alphabet))
	}

	for i := 0; i < len(alphabet); i++ {
		s := string(alphabet[i])

		if !IsValid(s) {
			t.Fatalf("expected %q to be valid, got invalid", s)
		}

		got, err := Decode(s)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		want := []byte{byte(i)}
		if !bytes.Equal(got, want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestEncodeDoesNotModifySource(t *testing.T) {
	src := []byte{0x00, 0x01, 0x02, 0x03, 0xff}
	want := append([]byte(nil), src...)

	_ = Encode(src)

	if !bytes.Equal(src, want) {
		t.Fatalf("expected %v, got %v", want, src)
	}
}

func TestDecodeReturnsNewSlice(t *testing.T) {
	got, err := Decode("2")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	got[0] = 99

	again, err := Decode("2")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []byte{0x01}
	if !bytes.Equal(again, want) {
		t.Fatalf("expected %v, got %v", want, again)
	}
}

func TestDecodeEmptyReturnsNewSlice(t *testing.T) {
	got, err := Decode("")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected length 0, got %d", len(got))
	}
}
