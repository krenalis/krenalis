// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package util

// Abbreviate abbreviates s to almost n runes with n >= 3. It returns s if its
// length is not greater than n runes; otherwise truncate s to n-3 runes and
// append "[…]". It panics if n < 3.
func Abbreviate(s string, n int) string {
	if n < 3 {
		panic("cannot abbreviate to fewer than 3 rune")
	}
	if len(s) <= n {
		return s
	}
	l := 0
	var cuts [3]int
	for i := range s {
		l++
		c := l % 3
		if l > n {
			return s[:cuts[c]] + "[…]"
		}
		cuts[c] = i
	}
	return s
}
