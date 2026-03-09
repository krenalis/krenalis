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

func asciiLower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func asciiEqualFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if asciiLower(s[i]) != t[i] {
			return false
		}
	}
	return true
}
