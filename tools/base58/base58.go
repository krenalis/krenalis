// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package base58

import (
	"crypto/rand"
)

// alphabet is a Base58 alphabet without 0, O, I, and l to avoid ambiguity.
const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

const maxValid = byte(58 * (256 / 58)) // 232

// IsValid reports whether s contains only Base58 characters.
// It returns true for the empty string.
func IsValid(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isBase58[s[i]] {
			return false
		}
	}
	return true
}

// Generate returns a string of n random Base58 characters.
//
// The string is generated using cryptographically secure randomness.
// Generate panics if n is negative.
func Generate(n int) string {

	if n < 0 {
		panic("n is negative")
	}

	id := make([]byte, n)

	var buf [32]byte
	random := buf[:min(n+n/8+1, 32)]

	for i := 0; i < n; {
		_, _ = rand.Read(random)
		for _, b := range random {
			if b >= maxValid {
				continue
			}
			id[i] = alphabet[b%58]
			i++
			if i == len(id) {
				break
			}
		}
	}

	return string(id)
}

var isBase58 = [256]bool{
	'1': true,
	'2': true,
	'3': true,
	'4': true,
	'5': true,
	'6': true,
	'7': true,
	'8': true,
	'9': true,
	'A': true,
	'B': true,
	'C': true,
	'D': true,
	'E': true,
	'F': true,
	'G': true,
	'H': true,
	'J': true,
	'K': true,
	'L': true,
	'M': true,
	'N': true,
	'P': true,
	'Q': true,
	'R': true,
	'S': true,
	'T': true,
	'U': true,
	'V': true,
	'W': true,
	'X': true,
	'Y': true,
	'Z': true,
	'a': true,
	'b': true,
	'c': true,
	'd': true,
	'e': true,
	'f': true,
	'g': true,
	'h': true,
	'i': true,
	'j': true,
	'k': true,
	'm': true,
	'n': true,
	'o': true,
	'p': true,
	'q': true,
	'r': true,
	's': true,
	't': true,
	'u': true,
	'v': true,
	'w': true,
	'x': true,
	'y': true,
	'z': true,
}
