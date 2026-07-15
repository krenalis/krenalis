// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package synctoken creates, parses, and writes Sync-Token HTTP header values.
package synctoken

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/krenalis/krenalis/tools/base58"
)

// NonceSize is the number of random bytes required as the nonce passed to
// Encode.
const NonceSize = 12

const (
	keySize        = 32
	versionSize    = 8
	maxTokenSize   = 50
	associatedData = "Sync-Token"
)

// errInvalidSyncToken reports an invalid Sync-Token value.
var errInvalidSyncToken = errors.New("invalid sync token")

// Codec creates and parses Sync-Token values using one encryption key.
//
// A Codec is safe for concurrent use by multiple goroutines after construction.
type Codec struct {
	// aead encrypts and authenticates the state version.
	aead cipher.AEAD
}

// NewCodec returns a Codec using a 32-byte AES-256 key.
//
// It returns an error if key is not exactly 32 bytes long.
func NewCodec(key []byte) (*Codec, error) {
	if len(key) != keySize {
		return nil, errors.New("key is not 32 bytes long")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	if aead.NonceSize() != NonceSize {
		return nil, fmt.Errorf("unexpected GCM nonce size: got %d bytes, want %d", aead.NonceSize(), NonceSize)
	}
	return &Codec{aead: aead}, nil
}

// Encode encodes a state version and returns a Sync-Token value.
//
// nonce must contain exactly NonceSize raw bytes. It must be unique for every
// token encrypted with the same key.
//
// Before Base58 encoding, the token has the following binary structure:
//
//	nonce || encrypted-version || authentication-tag
func (c *Codec) Encode(version int, nonce []byte) (string, error) {
	if version < 0 {
		return "", fmt.Errorf("invalid negative version: %d", version)
	}
	if len(nonce) != NonceSize {
		return "", fmt.Errorf("invalid nonce length: got %d bytes, want %d", len(nonce), NonceSize)
	}
	var plaintext [versionSize]byte
	binary.BigEndian.PutUint64(plaintext[:], uint64(int64(version)))
	token := make([]byte, NonceSize, NonceSize+versionSize+c.aead.Overhead())
	copy(token, nonce)
	token = c.aead.Seal(token, nonce, plaintext[:], []byte(associatedData))
	return base58.EncodeToString(token), nil
}

// Decode decodes a Sync-Token value and returns the state version.
//
// It returns errInvalidSyncToken if value is malformed, has an unexpected
// size, was generated with another key, or fails authentication.
func (c *Codec) Decode(value string) (int, error) {
	if len(value) > maxTokenSize {
		return 0, errInvalidSyncToken
	}
	token, err := base58.DecodeString(value)
	if err != nil {
		return 0, errInvalidSyncToken
	}
	expectedSize := NonceSize + versionSize + c.aead.Overhead()
	if len(token) != expectedSize {
		return 0, errInvalidSyncToken
	}
	nonce := token[:NonceSize]
	ciphertext := token[NonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, []byte(associatedData))
	if err != nil || len(plaintext) != versionSize {
		return 0, errInvalidSyncToken
	}
	version := int64(binary.BigEndian.Uint64(plaintext))
	return int(version), nil
}
