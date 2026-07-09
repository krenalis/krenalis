// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package requestid provides Request-Id values for HTTP APIs.
//
// A RequestID is Base58-encoded. Its binary payload contains a 12-byte
// cryptographically random ID followed by an encrypted state version.
// The random ID is also used as the AES-GCM nonce when encrypting the state
// version.
package requestid

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"math"

	"github.com/krenalis/krenalis/tools/base58"
)

const (
	keySize                   = 32                                       // keySize is the required AES-256-GCM key length.
	randomIDSize              = 12                                       // randomIDSize is the length of the random part of a RequestID.
	tagSize                   = 12                                       // tagSize is the AES-GCM authentication tag length.
	plaintextSize             = 8                                        // plaintextSize is the serialized length of a state version.
	encryptedStateVersionSize = plaintextSize + tagSize                  // encryptedStateVersionSize is the length of the encrypted state version.
	requestIDSize             = encryptedStateVersionSize + randomIDSize // requestIDSize is the binary length of a RequestID.
)

// errInvalidRequestID reports an invalid Request-Id value.
var errInvalidRequestID = errors.New("invalid request id")

// Codec creates and parses RequestID values using one encryption key.
//
// A Codec is safe for concurrent use by multiple goroutines after construction.
type Codec struct {
	// aead encrypts and authenticates the state version.
	aead cipher.AEAD
}

// RequestID is the request identifier associated with one HTTP request.
// A zero RequestID is not valid.
type RequestID struct {
	codec                 *Codec             // codec encrypts and decrypts the state version stored in the RequestID.
	randomID              [randomIDSize]byte // randomID is the raw random part of the RequestID.
	randomIDString        string             // randomIDString is the encoded random part of the RequestID.
	encryptedStateVersion []byte             // encryptedStateVersion is the encrypted state version.
	stateVersion          int                // stateVersion is the decrypted state version.
}

// NewCodec returns a Codec using a 32-byte key.
// It returns an error if key is not 32 bytes long.
func NewCodec(key []byte) (*Codec, error) {
	if len(key) != keySize {
		return nil, errors.New("key is not 32-byte long")
	}
	var k [keySize]byte
	copy(k[:], key)
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCMWithTagSize(block, tagSize)
	if err != nil {
		return nil, err
	}
	return &Codec{aead: aead}, nil
}

// New returns a new RequestID containing stateVersion.
// It panics if the state version is negative.
func (c *Codec) New(stateVersion int) *RequestID {
	return c.new(rand.Reader, stateVersion)
}

// Parse parses a complete Request-Id value.
// The returned RequestID has a state version.
func (c *Codec) Parse(value string) (*RequestID, error) {
	raw, err := base58.Decode(value)
	if err != nil {
		return nil, err
	}
	if len(raw) != requestIDSize {
		return nil, errInvalidRequestID
	}
	randomID := parseRandomID(raw[:randomIDSize])
	encryptedStateVersion := raw[randomIDSize:]

	plaintext, err := c.aead.Open(
		nil,
		randomID[:],
		encryptedStateVersion,
		nil,
	)
	if err != nil {
		return nil, err
	}
	if len(plaintext) != plaintextSize {
		return nil, errInvalidRequestID
	}
	stateVersion := binary.BigEndian.Uint64(plaintext)
	if stateVersion > uint64(math.MaxInt) {
		return nil, errInvalidRequestID
	}

	requestID := &RequestID{
		codec:                 c,
		randomID:              randomID,
		randomIDString:        base58.Encode(randomID[:]),
		encryptedStateVersion: append([]byte(nil), encryptedStateVersion...),
		stateVersion:          int(stateVersion),
	}

	return requestID, nil
}

// RandomID returns the encoded random part of r.
func (r *RequestID) RandomID() string {
	return r.randomIDString
}

// LogValue returns r as a slog string value.
func (r *RequestID) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

// SetStateVersion encrypts stateVersion and stores it in r.
func (r *RequestID) SetStateVersion(version int) {
	if version < 0 {
		panic("negative state version")
	}
	var plaintext [plaintextSize]byte
	binary.BigEndian.PutUint64(plaintext[:], uint64(version))
	ciphertext := r.codec.aead.Seal(
		nil,
		r.randomID[:],
		plaintext[:],
		nil,
	)
	r.encryptedStateVersion = ciphertext
	r.stateVersion = version
}

// String returns r as a Request-Id value.
func (r *RequestID) String() string {
	payload := make([]byte, 0, requestIDSize)
	payload = append(payload, r.randomID[:]...)
	payload = append(payload, r.encryptedStateVersion...)
	return base58.Encode(payload)
}

// StateVersion returns the state version stored in r.
func (r *RequestID) StateVersion() int {
	return r.stateVersion
}

// new returns a new RequestID using random as the random source.
// It panics if the state version is negative.
func (c *Codec) new(random io.Reader, stateVersion int) *RequestID {
	var randomID [randomIDSize]byte
	_, _ = io.ReadFull(random, randomID[:])
	requestID := &RequestID{
		codec:          c,
		randomID:       randomID,
		randomIDString: base58.Encode(randomID[:]),
	}
	requestID.SetStateVersion(stateVersion)
	return requestID
}

// parseRandomID parses the binary random-id part of a Request-Id.
func parseRandomID(raw []byte) [randomIDSize]byte {
	var randomID [randomIDSize]byte
	copy(randomID[:], raw)
	return randomID
}
