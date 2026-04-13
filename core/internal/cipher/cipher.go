// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cipher

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"slices"

	"github.com/krenalis/krenalis/tools/kms"
)

type Cipher struct {
	keyManager kms.Kms
}

// New creates a cipher that encrypts and decrypts state values using the given
// key manager.
func New(keyManager kms.Kms) *Cipher {
	return &Cipher{keyManager: keyManager}
}

// Encrypt uses a newly generated data key of keyLen bytes to encrypt plaintext.
// It returns the ciphertext and the KMS-encrypted data key.
func (c *Cipher) Encrypt(ctx context.Context, plaintext []byte, keyLen int) ([]byte, []byte, error) {

	dataKey, encryptedDataKey, err := c.keyManager.GenerateDataKey(ctx, keyLen)
	if err != nil {
		return nil, nil, err
	}
	defer clear(dataKey)

	if len(encryptedDataKey) == 0 {
		return nil, nil, errors.New("encrypted data key is empty")
	}

	ciphertext, err := encryptWithDataKey(plaintext, dataKey)
	if err != nil {
		return nil, nil, err
	}

	return ciphertext, encryptedDataKey, nil
}

// EncryptWithExistingKey encrypts plaintext using the data key represented by
// encryptedDataKey.
func (c *Cipher) EncryptWithExistingKey(ctx context.Context, plaintext, encryptedDataKey []byte) ([]byte, error) {
	dataKey, err := c.decryptDataKey(ctx, encryptedDataKey)
	if err != nil {
		return nil, err
	}
	defer clear(dataKey)
	return encryptWithDataKey(plaintext, dataKey)
}

// Decrypt decrypts ciphertext using the given encrypted data key.
func (c *Cipher) Decrypt(ctx context.Context, ciphertext, encryptedDataKey []byte) ([]byte, error) {
	dataKey, err := c.decryptDataKey(ctx, encryptedDataKey)
	if err != nil {
		return nil, err
	}
	defer clear(dataKey)
	return decryptWithDataKey(ciphertext, dataKey)
}

func (c *Cipher) Key(encryptedDataKey []byte) *Key {
	return &Key{slices.Clone(encryptedDataKey), c}
}

func (c *Cipher) KeyManager() kms.Kms {
	return c.keyManager
}

// decryptDataKey decrypts encryptedDataKey through the configured key manager.
func (c *Cipher) decryptDataKey(ctx context.Context, encryptedDataKey []byte) ([]byte, error) {
	if len(encryptedDataKey) == 0 {
		return nil, errors.New("encrypted data key is empty")
	}
	return c.keyManager.DecryptDataKey(ctx, encryptedDataKey)
}

// encryptWithDataKey encrypts plaintext with dataKey using AES-GCM.
// The returned ciphertext is prefixed with the nonce.
func encryptWithDataKey(plaintext, dataKey []byte) ([]byte, error) {
	if len(dataKey) != 32 {
		return nil, errors.New("data key must be 32 bytes")
	}
	return encryptWithAESGCM(plaintext, dataKey)
}

func encryptWithAESGCM(plaintext, dataKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	out := make([]byte, aead.NonceSize(), aead.NonceSize()+len(plaintext)+aead.Overhead())
	rand.Read(out[:aead.NonceSize()])
	out = aead.Seal(out, out[:aead.NonceSize()], plaintext, nil)
	return out, nil
}

// decryptWithDataKey decrypts ciphertext with dataKey using AES-GCM.
// ciphertext must be prefixed with the nonce.
func decryptWithDataKey(ciphertext, dataKey []byte) ([]byte, error) {
	if len(dataKey) != 32 {
		return nil, errors.New("data key must be 32 bytes")
	}
	return decryptWithAESGCM(ciphertext, dataKey)
}

func decryptWithAESGCM(ciphertext, dataKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	minSize := aead.NonceSize() + aead.Overhead()
	if len(ciphertext) < minSize {
		return nil, errors.New("ciphertext is too short")
	}

	nonce := ciphertext[:aead.NonceSize()]
	sealed := ciphertext[aead.NonceSize():]

	return aead.Open(make([]byte, 0, len(sealed)-aead.Overhead()), nonce, sealed, nil)
}

type Key struct {
	key    []byte
	cipher *Cipher
}

func (c *Key) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	return c.cipher.EncryptWithExistingKey(ctx, plaintext, c.key)
}

func (c *Key) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	return c.cipher.Decrypt(ctx, ciphertext, c.key)
}

func (c *Key) IsValid(ctx context.Context) error {
	dataKey, err := c.cipher.keyManager.DecryptDataKey(ctx, c.key)
	clear(dataKey)
	return err
}
