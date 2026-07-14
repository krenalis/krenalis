// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package synctoken

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	stderrors "errors"
	"math"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/base58"
)

// TestNewCodec verifies codec construction from raw key bytes.
func TestNewCodec(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		codec, err := NewCodec(bytes.Repeat([]byte{0x33}, keySize))
		if err != nil {
			t.Fatalf("expected NewCodec to succeed, got %v", err)
		}
		if codec == nil {
			t.Fatalf("expected codec, got nil")
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		_, err := NewCodec(bytes.Repeat([]byte{0x33}, keySize-1))
		if err == nil {
			t.Fatalf("expected invalid key error, got nil")
		}
	})
}

// TestCodecEncode verifies that Encode produces the expected Sync-Token wire
// format.
func TestCodecEncode(t *testing.T) {
	key := testKey()
	codec, err := NewCodec(key[:])
	if err != nil {
		t.Fatalf("expected NewCodec to succeed, got %v", err)
	}
	nonce := testNonce()

	token, err := codec.Encode(77, nonce[:])
	if err != nil {
		t.Fatalf("expected Encode to succeed, got %v", err)
	}

	want := expectedSyncToken(t, key, nonce, 77)
	if token != want {
		t.Fatalf("expected Sync-Token %q, got %q", want, token)
	}
	assertSyncTokenEncoding(t, token)
	assertSyncTokenNoncePrefix(t, token, nonce)
}

// TestCodecEncodeInvalidNonceLength verifies that Encode rejects nonces with
// an unexpected size.
func TestCodecEncodeInvalidNonceLength(t *testing.T) {
	codec := newTestCodec(t, 0x42)

	tests := []struct {
		name  string
		nonce []byte
	}{
		{
			name:  "short",
			nonce: bytes.Repeat([]byte{0x01}, NonceSize-1),
		},
		{
			name:  "long",
			nonce: bytes.Repeat([]byte{0x01}, NonceSize+1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.Encode(1, tt.nonce)
			if err == nil {
				t.Fatalf("expected invalid nonce length error, got nil")
			}
		})
	}
}

// TestCodecDecode verifies that Decode returns the state version encrypted in
// a Sync-Token.
func TestCodecDecode(t *testing.T) {
	codec := newTestCodec(t, 0x42)
	nonce := testNonce()

	tests := []struct {
		name    string
		version int
	}{
		{
			name:    "zero",
			version: 0,
		},
		{
			name:    "positive",
			version: 123,
		},
		{
			name:    "max int",
			version: math.MaxInt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := codec.Encode(tt.version, nonce[:])
			if err != nil {
				t.Fatalf("expected Encode to succeed, got %v", err)
			}

			got, err := codec.Decode(token)
			if err != nil {
				t.Fatalf("expected Decode to succeed, got %v", err)
			}
			if got != tt.version {
				t.Fatalf("expected state version %d, got %d", tt.version, got)
			}
		})
	}
}

// TestCodecDecodeErrors verifies malformed, tampered, and unauthentic
// Sync-Token values.
func TestCodecDecodeErrors(t *testing.T) {
	codec := newTestCodec(t, 0x42)

	t.Run("malformed format", func(t *testing.T) {
		tests := []string{
			"",
			"abc",
			strings.Repeat("1", syncTokenSize()-1),
			strings.Repeat("1", syncTokenSize()+1),
		}

		for _, value := range tests {
			t.Run(value, func(t *testing.T) {
				_, err := codec.Decode(value)
				if !stderrors.Is(err, errInvalidSyncToken) {
					t.Fatalf("expected invalid Sync-Token error, got %v", err)
				}
			})
		}
	})

	t.Run("invalid Base58 character", func(t *testing.T) {
		_, err := codec.Decode("!")
		if !stderrors.Is(err, errInvalidSyncToken) {
			t.Fatalf("expected invalid Sync-Token error, got %v", err)
		}
	})

	t.Run("invalid binary length", func(t *testing.T) {
		_, err := codec.Decode(base58.EncodeToString(bytes.Repeat([]byte{0x00}, syncTokenSize()-1)))
		if !stderrors.Is(err, errInvalidSyncToken) {
			t.Fatalf("expected invalid Sync-Token error, got %v", err)
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		token := mustEncode(t, codec, 123, testNonce())

		_, err := codec.Decode(tamperByte(t, token, NonceSize))
		if !stderrors.Is(err, errInvalidSyncToken) {
			t.Fatalf("expected invalid Sync-Token error, got %v", err)
		}
	})

	t.Run("tampered nonce", func(t *testing.T) {
		token := mustEncode(t, codec, 123, testNonce())

		_, err := codec.Decode(tamperByte(t, token, NonceSize-1))
		if !stderrors.Is(err, errInvalidSyncToken) {
			t.Fatalf("expected invalid Sync-Token error, got %v", err)
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		codecA := newTestCodec(t, 0x11)
		codecB := newTestCodec(t, 0x22)
		token := mustEncode(t, codecA, 123, testNonce())

		_, err := codecB.Decode(token)
		if !stderrors.Is(err, errInvalidSyncToken) {
			t.Fatalf("expected invalid Sync-Token error, got %v", err)
		}
	})
}

// newTestCodec returns a codec initialized with a repeated-byte key.
func newTestCodec(t *testing.T, b byte) *Codec {
	t.Helper()

	codec, err := NewCodec(bytes.Repeat([]byte{b}, keySize))
	if err != nil {
		t.Fatalf("expected NewCodec to succeed, got %v", err)
	}

	return codec
}

// testKey returns a deterministic test key.
func testKey() [keySize]byte {
	var key [keySize]byte
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

// testNonce returns a deterministic Sync-Token nonce.
func testNonce() [NonceSize]byte {
	return [NonceSize]byte{
		0xf0, 0xf1, 0xf2, 0xf3,
		0xf4, 0xf5, 0xf6, 0xf7,
		0xf8, 0xf9, 0xfa, 0xfb,
	}
}

// expectedSyncToken returns the expected wire value for version.
func expectedSyncToken(t *testing.T, key [keySize]byte, nonce [NonceSize]byte, version int) string {
	t.Helper()

	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("expected AES cipher to initialize, got %v", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("expected GCM to initialize, got %v", err)
	}

	var plaintext [versionSize]byte
	binary.BigEndian.PutUint64(plaintext[:], uint64(int64(version)))

	payload := make([]byte, NonceSize, NonceSize+versionSize+aead.Overhead())
	copy(payload, nonce[:])
	payload = aead.Seal(payload, nonce[:], plaintext[:], []byte(associatedData))

	return base58.EncodeToString(payload)
}

// mustEncode returns a Sync-Token or fails the test.
func mustEncode(t *testing.T, codec *Codec, version int, nonce [NonceSize]byte) string {
	t.Helper()

	token, err := codec.Encode(version, nonce[:])
	if err != nil {
		t.Fatalf("expected Encode to succeed, got %v", err)
	}
	return token
}

// syncTokenSize returns the binary size of a Sync-Token.
func syncTokenSize() int {
	codec, err := NewCodec(bytes.Repeat([]byte{0x42}, keySize))
	if err != nil {
		panic(err)
	}
	return NonceSize + versionSize + codec.aead.Overhead()
}

// assertSyncTokenEncoding verifies that token is valid Base58 with the
// expected binary size.
func assertSyncTokenEncoding(t *testing.T, token string) {
	t.Helper()

	payload, err := base58.DecodeString(token)
	if err != nil {
		t.Fatalf("expected decoded Sync-Token, got %v", err)
	}
	if len(payload) != syncTokenSize() {
		t.Fatalf("expected Sync-Token binary length %d, got %d", syncTokenSize(), len(payload))
	}
}

// assertSyncTokenNoncePrefix verifies that token's binary payload starts with
// nonce.
func assertSyncTokenNoncePrefix(t *testing.T, token string, nonce [NonceSize]byte) {
	t.Helper()

	payload, err := base58.DecodeString(token)
	if err != nil {
		t.Fatalf("expected decoded Sync-Token, got %v", err)
	}
	if len(payload) != syncTokenSize() {
		t.Fatalf("expected Sync-Token binary length %d, got %d", syncTokenSize(), len(payload))
	}
	if !bytes.Equal(payload[:NonceSize], nonce[:]) {
		t.Fatalf("expected nonce prefix %v, got %v", nonce, payload[:NonceSize])
	}
}

// tamperByte flips one byte in a Sync-Token binary payload and returns the
// re-encoded token.
func tamperByte(t *testing.T, token string, index int) string {
	t.Helper()

	payload, err := base58.DecodeString(token)
	if err != nil {
		t.Fatalf("expected decoded Sync-Token, got %v", err)
	}
	if len(payload) != syncTokenSize() {
		t.Fatalf("expected Sync-Token binary length %d, got %d", syncTokenSize(), len(payload))
	}

	payload[index] ^= 0xff
	return base58.EncodeToString(payload)
}
