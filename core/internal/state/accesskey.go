// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const (
	base62Alphabet             = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	accessKeyPayloadSize       = 32
	accessKeyPayloadBase62Size = 43
	accessKeyCRC32Base62Size   = 6

	accessKeyBodySize  = accessKeyPayloadBase62Size + accessKeyCRC32Base62Size
	invalidBase62Digit = 0xff
)

var base62Digits = [256]byte{
	'A': 0, 'B': 1, 'C': 2, 'D': 3, 'E': 4, 'F': 5, 'G': 6, 'H': 7, 'I': 8, 'J': 9,
	'K': 10, 'L': 11, 'M': 12, 'N': 13, 'O': 14, 'P': 15, 'Q': 16, 'R': 17,
	'S': 18, 'T': 19, 'U': 20, 'V': 21, 'W': 22, 'X': 23, 'Y': 24, 'Z': 25,
	'a': 26, 'b': 27, 'c': 28, 'd': 29, 'e': 30, 'f': 31, 'g': 32, 'h': 33,
	'i': 34, 'j': 35, 'k': 36, 'l': 37, 'm': 38, 'n': 39, 'o': 40, 'p': 41,
	'q': 42, 'r': 43, 's': 44, 't': 45, 'u': 46, 'v': 47, 'w': 48, 'x': 49,
	'y': 50, 'z': 51, '0': 52, '1': 53, '2': 54, '3': 55, '4': 56, '5': 57,
	'6': 58, '7': 59, '8': 60, '9': 61,
}

func init() {
	for i := range base62Digits {
		if base62Digits[i] == 0 && i != 'A' {
			base62Digits[i] = invalidBase62Digit
		}
	}
}

var errInvalidToken = errors.New("access key token is invalid")

// GenerateAccessKey returns a new random access key and the HMAC of its raw
// payload.
func (state *State) GenerateAccessKey(ctx context.Context) (string, []byte, error) {
	payload := make([]byte, accessKeyPayloadSize)
	_, _ = rand.Read(payload)
	defer clear(payload)
	hmac, err := state.metadata.apiKeyPepper.HMAC(ctx, payload)
	if err != nil {
		return "", nil, err
	}
	return formatAccessKey(payload), hmac[:], nil
}

// formatAccessKey returns the fixed-length base62 token body for payload.
func formatAccessKey(payload []byte) string {
	if len(payload) != accessKeyPayloadSize {
		panic("state: payload must be 32 bytes")
	}
	var crc [4]byte
	defer clear(crc[:])
	binary.BigEndian.PutUint32(crc[:], crc32.ChecksumIEEE(payload))
	out := make([]byte, accessKeyBodySize)
	defer clear(out)
	encodeFixedBase62(out[:accessKeyPayloadBase62Size], payload)
	encodeFixedBase62(out[accessKeyPayloadBase62Size:], crc[:])
	return string(out)
}

// parseAccessKey parses token as an access key and returns its 32-byte payload.
// It returns errInvalidToken if token is malformed or fails its CRC check.
func parseAccessKey(token string) (payload []byte, err error) {
	if len(token) != accessKeyBodySize {
		return nil, errInvalidToken
	}
	payload = make([]byte, accessKeyPayloadSize)
	defer func() {
		if err != nil {
			clear(payload)
		}
	}()
	err = decodeFixedBase62(payload, token[:accessKeyPayloadBase62Size])
	if err != nil {
		return
	}
	var crc [4]byte
	err = decodeFixedBase62(crc[:], token[accessKeyPayloadBase62Size:])
	if err != nil {
		return
	}
	if crc32.ChecksumIEEE(payload) != binary.BigEndian.Uint32(crc[:]) {
		err = errInvalidToken
		return
	}
	return payload, nil
}

// encodeFixedBase62 encodes data into dst using base62. data must be at most 32
// bytes.
//
// The output is left-padded with the zero digit. dst must be large enough to
// hold the encoded value.
func encodeFixedBase62(dst, data []byte) {
	for i := range dst {
		dst[i] = base62Alphabet[0]
	}
	var n [accessKeyPayloadSize]byte
	defer clear(n[:])
	copy(n[len(n)-len(data):], data)
	for i := len(dst) - 1; i >= 0; i-- {
		rem := 0
		nonzero := false
		for j := range n {
			v := rem<<8 + int(n[j])
			q := v / 62
			rem = v % 62
			n[j] = byte(q)
			nonzero = nonzero || q != 0
		}
		dst[i] = base62Alphabet[rem]
		if !nonzero {
			return
		}
	}
	panic("state: base62 buffer is too small")
}

// decodeFixedBase62 decodes s from base62 into dst. The output is left-padded
// with zero bytes.
//
// It returns errInvalidToken if s contains a non-base62 byte or if the decoded
// value does not fit in dst.
func decodeFixedBase62(dst []byte, s string) error {
	clear(dst) // initialize the base62 accumulator
	for i := 0; i < len(s); i++ {
		digit := base62Digits[s[i]]
		if digit == invalidBase62Digit {
			return errInvalidToken
		}
		carry := int(digit)
		for j := len(dst) - 1; j >= 0; j-- {
			v := int(dst[j])*62 + carry
			dst[j] = byte(v)
			carry = v >> 8
		}
		if carry != 0 {
			return errInvalidToken
		}
	}
	return nil
}
