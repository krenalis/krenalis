// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

// asciiWordSet stores lowercase ASCII words grouped by byte length.
//
// It is used for fast case-insensitive membership checks without allocations,
// Unicode handling, or string normalization.
type asciiWordSet struct {
	byLen [32][]string
}

// newASCIIWordSet returns a set containing words.
func newASCIIWordSet(words ...string) asciiWordSet {
	var set asciiWordSet
	for _, word := range words {
		if len(word) >= len(set.byLen) {
			panic("readonlysql: ASCII word too long")
		}
		set.byLen[len(word)] = append(set.byLen[len(word)], word)
	}
	return set
}

// Has reports whether word is in s, using ASCII case folding.
func (s asciiWordSet) Has(word string) bool {
	n := len(word)
	if n >= len(s.byLen) {
		return false
	}
	for _, candidate := range s.byLen[n] {
		if asciiEqualFold(word, candidate) {
			return true
		}
	}
	return false
}

// asciiLower returns the lowercase ASCII form of b.
func asciiLower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// asciiEqualFold reports whether s and t are equal under ASCII case folding.
func asciiEqualFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if asciiLower(s[i]) != asciiLower(t[i]) {
			return false
		}
	}
	return true
}

// asciiLowerString returns the lowercase ASCII form of s.
func asciiLowerString(s string) string {
	for i := 0; i < len(s); i++ {
		if 'A' <= s[i] && s[i] <= 'Z' {
			b := make([]byte, len(s))
			copy(b, s[:i])
			b[i] = asciiLower(s[i])
			for j := i + 1; j < len(s); j++ {
				b[j] = asciiLower(s[j])
			}
			return string(b)
		}
	}
	return s
}
