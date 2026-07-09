// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package requestid

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"log/slog"
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

// TestCodecNew verifies new RequestID generation.
func TestCodecNew(t *testing.T) {
	t.Run("with state version", func(t *testing.T) {
		key := testKey()
		codec, err := NewCodec(key[:])
		if err != nil {
			t.Fatalf("expected NewCodec to succeed, got %v", err)
		}
		randomID := testRandomID()

		id := codec.new(bytes.NewReader(randomID), 77)

		want := expectedRequestID(t, key, randomID, 77)
		if id.String() != want {
			t.Fatalf("expected RequestID string %q, got %q", want, id.String())
		}
		assertRequestIDEncoding(t, id.String())
		assertRequestIDRandomIDPrefix(t, id.String(), randomID)
		gotRandomID, err := base58.Decode(id.RandomID())
		if err != nil {
			t.Fatalf("expected decoded random ID, got %v", err)
		}
		if !bytes.Equal(gotRandomID, randomID) {
			t.Fatalf("expected random ID %v, got %v", randomID, gotRandomID)
		}
		gotVersion := id.StateVersion()
		if gotVersion != 77 {
			t.Fatalf("expected state version 77, got %d", gotVersion)
		}
	})
}

// TestRequestIDPayloadOrder verifies the binary payload order.
func TestRequestIDPayloadOrder(t *testing.T) {
	codec := newTestCodec(t, 0x42)
	randomID := testRandomID()

	id := codec.new(bytes.NewReader(randomID), 77)

	assertRequestIDRandomIDPrefix(t, id.String(), randomID)
}

// TestRequestIDSetStateVersion verifies state-version encryption and string
// format.
func TestRequestIDSetStateVersion(t *testing.T) {
	key := testKey()
	codec, err := NewCodec(key[:])
	if err != nil {
		t.Fatalf("expected NewCodec to succeed, got %v", err)
	}

	randomID := testRandomID()
	id := codec.new(bytes.NewReader(randomID), 0)

	id.SetStateVersion(99)

	want := expectedRequestID(t, key, randomID, 99)
	if id.String() != want {
		t.Fatalf("expected RequestID %q, got %q", want, id.String())
	}
	assertRequestIDEncoding(t, id.String())
	gotRandomID, err := base58.Decode(id.RandomID())
	if err != nil {
		t.Fatalf("expected decoded random ID, got %v", err)
	}
	if !bytes.Equal(gotRandomID, randomID) {
		t.Fatalf("expected random ID %v, got %v", randomID, gotRandomID)
	}
	gotVersion := id.StateVersion()
	if gotVersion != 99 {
		t.Fatalf("expected state version 99, got %d", gotVersion)
	}
}

// TestRequestIDSetStateVersionAgain verifies that SetStateVersion updates the
// version.
func TestRequestIDSetStateVersionAgain(t *testing.T) {
	codec := newTestCodec(t, 0x42)
	id := codec.new(bytes.NewReader(testRandomID()), 10)

	id.SetStateVersion(11)

	gotVersion := id.StateVersion()
	if gotVersion != 11 {
		t.Fatalf("expected state version 11, got %d", gotVersion)
	}
}

// TestCodecParse verifies parsing and decrypting a complete Request-Id.
func TestCodecParse(t *testing.T) {
	key := testKey()
	codec, err := NewCodec(key[:])
	if err != nil {
		t.Fatalf("expected NewCodec to succeed, got %v", err)
	}

	t.Run("with state version", func(t *testing.T) {
		randomID := testRandomID()
		value := expectedRequestID(t, key, randomID, 123)

		id, err := codec.Parse(value)
		if err != nil {
			t.Fatalf("expected Parse to succeed, got %v", err)
		}
		if id.String() != value {
			t.Fatalf("expected RequestID string %q, got %q", value, id.String())
		}
		assertRequestIDEncoding(t, id.String())
		gotRandomID, err := base58.Decode(id.RandomID())
		if err != nil {
			t.Fatalf("expected decoded random ID, got %v", err)
		}
		if !bytes.Equal(gotRandomID, randomID) {
			t.Fatalf("expected random ID %v, got %v", randomID, gotRandomID)
		}
		gotVersion := id.StateVersion()
		if gotVersion != 123 {
			t.Fatalf("expected state version 123, got %d", gotVersion)
		}
	})

	t.Run("with max int state version", func(t *testing.T) {
		randomID := testRandomID()
		value := expectedRequestIDUint64(t, key, randomID, uint64(math.MaxInt))

		id, err := codec.Parse(value)
		if err != nil {
			t.Fatalf("expected Parse to succeed, got %v", err)
		}
		if id.StateVersion() != math.MaxInt {
			t.Fatalf("expected state version %d, got %d", math.MaxInt, id.StateVersion())
		}
	})
}

// TestCodecParseErrors verifies malformed, tampered, and unauthentic values.
func TestCodecParseErrors(t *testing.T) {
	codec := newTestCodec(t, 0x42)

	t.Run("malformed format", func(t *testing.T) {
		tests := []string{
			"",
			"abc",
			strings.Repeat("1", 1),
			strings.Repeat("1", requestIDSize-1),
			strings.Repeat("1", requestIDSize+1),
		}

		for _, value := range tests {
			t.Run(value, func(t *testing.T) {
				_, err := codec.Parse(value)
				if err == nil {
					t.Fatalf("expected invalid format error, got nil")
				}
			})
		}
	})

	t.Run("invalid Base58 character", func(t *testing.T) {
		_, err := codec.Parse("!")
		if err == nil {
			t.Fatalf("expected invalid Base58 error, got nil")
		}
	})

	t.Run("invalid base58", func(t *testing.T) {
		_, err := codec.Parse(strings.Repeat("1", requestIDSize-1) + "!")
		if err == nil {
			t.Fatalf("expected invalid Base58 error, got nil")
		}
	})

	t.Run("invalid binary length", func(t *testing.T) {
		_, err := codec.Parse(base58.Encode(bytes.Repeat([]byte{0x00}, requestIDSize-1)))
		if err == nil {
			t.Fatalf("expected invalid RequestID length error, got nil")
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		id := codec.new(bytes.NewReader(testRandomID()), 0)
		id.SetStateVersion(123)

		_, err := codec.Parse(tamperEncryptedPart(t, id.String()))
		if err == nil {
			t.Fatalf("expected invalid ciphertext error, got nil")
		}
	})

	t.Run("tampered random id", func(t *testing.T) {
		id := codec.new(bytes.NewReader(testRandomID()), 123)

		_, err := codec.Parse(tamperRandomIDPart(t, id.String()))
		if err == nil {
			t.Fatalf("expected invalid random ID error, got nil")
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		codecA := newTestCodec(t, 0x11)
		codecB := newTestCodec(t, 0x22)

		id := codecA.new(bytes.NewReader(testRandomID()), 0)
		id.SetStateVersion(123)

		_, err := codecB.Parse(id.String())
		if err == nil {
			t.Fatalf("expected invalid ciphertext error, got nil")
		}
	})

	t.Run("state version overflow", func(t *testing.T) {
		key := testKey()
		codec, err := NewCodec(key[:])
		if err != nil {
			t.Fatalf("expected NewCodec to succeed, got %v", err)
		}
		value := expectedRequestIDUint64(t, key, testRandomID(), math.MaxUint64)

		_, err = codec.Parse(value)
		if err == nil {
			t.Fatalf("expected invalid state version error, got nil")
		}
	})
}

// TestRequestIDLogValue verifies slog rendering.
func TestRequestIDLogValue(t *testing.T) {
	codec := newTestCodec(t, 0x42)
	id := codec.new(bytes.NewReader(testRandomID()), 12)

	got := id.LogValue()
	if got.Kind() != slog.KindString {
		t.Fatalf("expected slog string value, got %s", got.Kind())
	}
	if got.String() != id.String() {
		t.Fatalf("expected slog value %q, got %q", id.String(), got.String())
	}
}

// TestRequestIDPanics verifies incorrect RequestID usage panics.
func TestRequestIDPanics(t *testing.T) {
	t.Run("SetStateVersion with negative state version", func(t *testing.T) {
		codec := newTestCodec(t, 0x42)
		id := codec.new(bytes.NewReader(testRandomID()), 0)
		expectPanic(t, func() { id.SetStateVersion(-1) })
	})

	t.Run("New with negative state version", func(t *testing.T) {
		codec := newTestCodec(t, 0x42)
		expectPanic(t, func() { codec.New(-1) })
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

// testRandomID returns a deterministic random ID.
func testRandomID() []byte {
	return []byte{
		0xf0, 0xf1, 0xf2, 0xf3,
		0xf4, 0xf5, 0xf6, 0xf7,
		0xf8, 0xf9, 0xfa, 0xfb,
	}
}

// expectedRequestID returns the expected wire value for stateVersion.
func expectedRequestID(t *testing.T, key [keySize]byte, randomID []byte, stateVersion int) string {
	t.Helper()
	return expectedRequestIDUint64(t, key, randomID, uint64(stateVersion))
}

// expectedRequestIDUint64 returns the expected wire value for stateVersion.
func expectedRequestIDUint64(t *testing.T, key [keySize]byte, randomID []byte, stateVersion uint64) string {
	t.Helper()

	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("expected AES cipher to initialize, got %v", err)
	}

	aead, err := cipher.NewGCMWithTagSize(block, tagSize)
	if err != nil {
		t.Fatalf("expected GCM to initialize, got %v", err)
	}

	var plaintext [plaintextSize]byte
	binary.BigEndian.PutUint64(plaintext[:], stateVersion)

	ciphertext := aead.Seal(nil, randomID[:], plaintext[:], nil)
	payload := make([]byte, 0, requestIDSize)
	payload = append(payload, randomID...)
	payload = append(payload, ciphertext...)
	return base58.Encode(payload)
}

// assertRequestIDEncoding verifies that requestID is valid Base58 for a
// RequestID.
func assertRequestIDEncoding(t *testing.T, requestID string) {
	t.Helper()

	payload, err := base58.Decode(requestID)
	if err != nil {
		t.Fatalf("expected decoded RequestID, got %v", err)
	}
	if len(payload) != requestIDSize {
		t.Fatalf("expected RequestID binary length %d, got %d", requestIDSize, len(payload))
	}
}

// assertRequestIDRandomIDPrefix verifies that requestID's binary payload starts
// with randomID.
func assertRequestIDRandomIDPrefix(t *testing.T, requestID string, randomID []byte) {
	t.Helper()

	payload, err := base58.Decode(requestID)
	if err != nil {
		t.Fatalf("expected decoded RequestID, got %v", err)
	}
	if len(payload) != requestIDSize {
		t.Fatalf("expected RequestID binary length %d, got %d", requestIDSize, len(payload))
	}
	if !bytes.Equal(payload[:randomIDSize], randomID) {
		t.Fatalf("expected random ID prefix %v, got %v", randomID, payload[:randomIDSize])
	}
}

// expectPanic verifies that f panics.
func expectPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic, got nil")
		}
	}()
	f()
}

// tamperEncryptedPart flips one byte in the encrypted-state-version part.
func tamperEncryptedPart(t *testing.T, requestID string) string {
	t.Helper()

	payload, err := base58.Decode(requestID)
	if err != nil {
		t.Fatalf("expected decoded RequestID, got %v", err)
	}
	if len(payload) != requestIDSize {
		t.Fatalf("expected RequestID binary length %d, got %d", requestIDSize, len(payload))
	}
	ciphertext := payload[randomIDSize:]
	if len(ciphertext) == 0 {
		t.Fatalf("expected non-empty ciphertext, got empty slice")
	}

	ciphertext[0] ^= 0xff

	return base58.Encode(payload)
}

// tamperRandomIDPart flips one byte in the random-id part.
func tamperRandomIDPart(t *testing.T, requestID string) string {
	t.Helper()

	payload, err := base58.Decode(requestID)
	if err != nil {
		t.Fatalf("expected decoded RequestID, got %v", err)
	}
	if len(payload) != requestIDSize {
		t.Fatalf("expected RequestID binary length %d, got %d", requestIDSize, len(payload))
	}
	randomID := payload[:randomIDSize]
	if len(randomID) == 0 {
		t.Fatalf("expected non-empty random ID, got empty slice")
	}

	randomID[len(randomID)-1] ^= 0xff

	return base58.Encode(payload)
}
