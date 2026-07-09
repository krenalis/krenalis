// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package base58

import (
	"crypto/rand"
	"errors"
)

// alphabet is a Base58 alphabet without 0, O, I, and l to avoid ambiguity.
const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

const maxValid = byte(58 * (256 / 58)) // maxValid is the largest accepted random byte without modulo bias.

// errInvalidBase58 reports an invalid Base58 character.
var errInvalidBase58 = errors.New("base58: invalid character")

// IsValid reports whether s contains only Base58 characters.
// It returns true for the empty string.
func IsValid(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isBase58(s[i]) {
			return false
		}
	}
	return true
}

// Decode returns the bytes encoded in s.
// It returns an error if s contains a non-Base58 character.
func Decode(s string) ([]byte, error) {

	if len(s) == 0 {
		return []byte{}, nil
	}

	zeros := 0
	for zeros < len(s) && s[zeros] == alphabet[0] {
		zeros++
	}

	decoded := make([]byte, 0, (len(s)-zeros)*733/1000+1)
	for i := zeros; i < len(s); i++ {
		c := s[i]
		if !isBase58(c) {
			return nil, errInvalidBase58
		}

		carry := int(decodeBase58[c])
		for j := 0; j < len(decoded); j++ {
			carry += int(decoded[j]) * 58
			decoded[j] = byte(carry)
			carry >>= 8
		}

		for carry > 0 {
			decoded = append(decoded, byte(carry))
			carry >>= 8
		}
	}

	out := make([]byte, zeros+len(decoded))
	for i := 0; i < len(decoded); i++ {
		out[zeros+i] = decoded[len(decoded)-1-i]
	}

	return out, nil
}

// Encode returns src encoded as a Base58 string.
func Encode(src []byte) string {

	if len(src) == 0 {
		return ""
	}

	zeros := 0
	for zeros < len(src) && src[zeros] == 0 {
		zeros++
	}

	digits := make([]byte, 0, (len(src)-zeros)*138/100+1)
	for _, b := range src[zeros:] {
		carry := int(b)

		for i := 0; i < len(digits); i++ {
			carry += int(digits[i]) << 8
			digits[i] = byte(carry % 58)
			carry /= 58
		}

		for carry > 0 {
			digits = append(digits, byte(carry%58))
			carry /= 58
		}
	}

	out := make([]byte, zeros+len(digits))
	for i := 0; i < zeros; i++ {
		out[i] = alphabet[0]
	}
	for i := 0; i < len(digits); i++ {
		out[zeros+i] = alphabet[digits[len(digits)-1-i]]
	}

	return string(out)
}

// Generate returns a string of n random Base58 characters.
//
// The string is generated using cryptographically secure randomness.
// Generate panics if n is negative.
func Generate(n int) string {

	if n < 0 {
		panic("base58: n is negative")
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

// isBase58 reports whether c is a Base58 character.
func isBase58(c byte) bool {
	// The zero value also represents alphabet[0], so it is handled explicitly.
	return decodeBase58[c] != 0 || c == alphabet[0]
}

// decodeBase58 maps Base58 characters to their numeric values.
var decodeBase58 = [256]byte{
	'1': 0,
	'2': 1,
	'3': 2,
	'4': 3,
	'5': 4,
	'6': 5,
	'7': 6,
	'8': 7,
	'9': 8,
	'A': 9,
	'B': 10,
	'C': 11,
	'D': 12,
	'E': 13,
	'F': 14,
	'G': 15,
	'H': 16,
	'J': 17,
	'K': 18,
	'L': 19,
	'M': 20,
	'N': 21,
	'P': 22,
	'Q': 23,
	'R': 24,
	'S': 25,
	'T': 26,
	'U': 27,
	'V': 28,
	'W': 29,
	'X': 30,
	'Y': 31,
	'Z': 32,
	'a': 33,
	'b': 34,
	'c': 35,
	'd': 36,
	'e': 37,
	'f': 38,
	'g': 39,
	'h': 40,
	'i': 41,
	'j': 42,
	'k': 43,
	'm': 44,
	'n': 45,
	'o': 46,
	'p': 47,
	'q': 48,
	'r': 49,
	's': 50,
	't': 51,
	'u': 52,
	'v': 53,
	'w': 54,
	'x': 55,
	'y': 56,
	'z': 57,
}
