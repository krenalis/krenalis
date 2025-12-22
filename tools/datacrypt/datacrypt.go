// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package datacrypt provides an API for encrypting and decrypting data using a
// shared master key across multiple, independent purposes.
//
// A Cipher is created from a 64-byte master key and a purpose string.
// The purpose identifies the context in which the data is used (for example,
// tokens, identifiers, or internal references) and keeps encrypted values
// from different contexts isolated from each other.
//
// Data encrypted for one purpose can only be decrypted using a Cipher created
// with the same master key and the same purpose. This allows a single master
// key to be safely reused without mixing or confusing different kinds of
// encrypted data.
package datacrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Cipher encrypts and decrypts data using a key derived from a master key and a
// specific purpose.
//
// A Cipher instance is created once and can be reused to encrypt and decrypt
// byte slices or strings. Data encrypted with a Cipher can only be decrypted by
// another Cipher created with the same master key and purpose.
type Cipher struct {
	aead      cipher.AEAD
	nonceSize int
	overhead  int
	aad       []byte // purpose-bound AAD, precomputed to avoid per-call allocations
}

// New creates a Cipher bound to the given master key and purpose.
//
// The master key must be exactly 64 bytes long, and purpose must be non-empty.
// The returned Cipher can be used to encrypt and decrypt data. Only a Cipher
// created with the same master key and purpose can decrypt the encrypted values.
//
// New returns an error if the input parameters are invalid or if the Cipher
// cannot be initialized.
func New(masterKey64 []byte, purpose string) (*Cipher, error) {

	if len(masterKey64) != 64 {
		return nil, errors.New("datacrypt: master key must be 64 bytes")
	}
	if purpose == "" {
		return nil, errors.New("datacrypt: purpose is empty")
	}

	// Derive a 32-byte subkey (AES-256) from the 64-byte master key.
	// "info" domain-separates this subkey from other uses.
	info := []byte("datacrypt:" + purpose + ":v1")
	h := hkdf.New(sha256.New, masterKey64, nil, info)

	subKey := make([]byte, 32)
	if _, err := io.ReadFull(h, subKey); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(subKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Precompute AAD once; Open/Seal will authenticate it.
	aad := []byte(purpose)

	c := &Cipher{
		aead:      aead,
		nonceSize: aead.NonceSize(),
		overhead:  aead.Overhead(),
		aad:       aad,
	}

	return c, nil
}

// Encrypt encrypts the given data and returns an opaque byte slice.
//
// The returned value can be safely stored or transmitted and later passed to
// Decrypt to recover the original data.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	totalLen := c.nonceSize + len(plaintext) + c.overhead

	// Create dst with len=nonceSize (to hold nonce) and cap=totalLen (final size).
	out := make([]byte, c.nonceSize, totalLen)

	// Fill nonce in-place.
	if _, err := rand.Read(out[:c.nonceSize]); err != nil {
		return nil, err
	}

	// Append ciphertext+tag after the nonce into the same buffer (no extra allocation).
	out = c.aead.Seal(out, out[:c.nonceSize], plaintext, c.aad)
	return out, nil
}

// Decrypt decrypts data previously returned by Encrypt.
//
// If the data was modified, corrupted, or encrypted with a different key or
// purpose, Decrypt returns an error.
func (c *Cipher) Decrypt(encrypted []byte) ([]byte, error) {
	if len(encrypted) < c.nonceSize {
		return nil, errors.New("datacrypt: encrypted is too short")
	}
	dst := make([]byte, 0, len(encrypted)-c.nonceSize)
	return c.DecryptAppend(dst, encrypted)
}

// DecryptAppend decrypts data previously returned by Encrypt and appends the
// plaintext to dst, returning the extended buffer.
//
// If the data was modified, corrupted, or encrypted with a different key or
// purpose, Decrypt returns an error.
func (c *Cipher) DecryptAppend(dst, encrypted []byte) ([]byte, error) {
	if len(encrypted) < c.nonceSize {
		return nil, errors.New("datacrypt: encrypted is too short")
	}
	nonce, ct := encrypted[:c.nonceSize], encrypted[c.nonceSize:]
	return c.aead.Open(dst, nonce, ct, c.aad)
}
