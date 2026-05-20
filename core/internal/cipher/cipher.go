// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cipher

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"slices"

	"github.com/krenalis/krenalis/tools/kms"
)

// Cipher encrypts and decrypts values using KMS-managed data keys.
// It is safe for concurrent use by multiple goroutines.
type Cipher struct {
	kms   kms.Kms
	cache *cache
}

// New creates a cipher that encrypts and decrypts state values using the given
// KMS.
func New(kms kms.Kms) *Cipher {
	return &Cipher{kms: kms, cache: newCache()}
}

// Close releases resources and clears any remaining cached keys.
//
// Close must not be called concurrently with any other Cipher method, and it
// must be called at most once. The Cipher must not be used after Close.
func (c *Cipher) Close() {
	c.cache.Close()
}

// Encrypt uses a newly generated data key of 32 bytes to encrypt plaintext.
// It returns the ciphertext and the KMS-encrypted data key.
func (c *Cipher) Encrypt(ctx context.Context, plaintext []byte) ([]byte, []byte, error) {

	dataKey, encryptedDataKey, err := c.kms.GenerateDataKey(ctx, 32)
	if err != nil {
		return nil, nil, err
	}

	clearKey := c.cache.Put(encryptedDataKey, dataKey)
	defer clearKey.Done()

	ciphertext, err := encryptWithAESGCM(plaintext, clearKey.Value)
	if err != nil {
		return nil, nil, err
	}

	return ciphertext, encryptedDataKey, nil
}

// EncryptWithExistingKey encrypts plaintext using the data key, that must be
// 32 bytes long, represented by encryptedDataKey.
func (c *Cipher) EncryptWithExistingKey(ctx context.Context, plaintext, encryptedDataKey []byte) ([]byte, error) {
	clearKey, err := c.decryptDataKey(ctx, encryptedDataKey)
	if err != nil {
		return nil, err
	}
	defer clearKey.Done()
	if len(clearKey.Value) != 32 {
		return nil, errors.New("cipher: data key must be 32 bytes")
	}
	return encryptWithAESGCM(plaintext, clearKey.Value)
}

// Decrypt decrypts ciphertext using the given encrypted data key.
func (c *Cipher) Decrypt(ctx context.Context, ciphertext, encryptedDataKey []byte) ([]byte, error) {
	clearKey, err := c.decryptDataKey(ctx, encryptedDataKey)
	if err != nil {
		return nil, err
	}
	defer clearKey.Done()
	return decryptWithDataKey(ciphertext, clearKey.Value)
}

// HMAC computes the HMAC-SHA-256 of data using the data key represented by
// encryptedDataKey.
func (c *Cipher) HMAC(ctx context.Context, data, encryptedDataKey []byte) ([32]byte, error) {
	clearKey, err := c.decryptDataKey(ctx, encryptedDataKey)
	if err != nil {
		return [32]byte{}, err
	}
	defer clearKey.Done()
	if len(clearKey.Value) != 32 {
		return [32]byte{}, errors.New("cipher: data key must be 32 bytes")
	}
	return hmacSHA256(data, clearKey.Value), nil
}

// KMS returns the KMS used by the cipher.
func (c *Cipher) KMS() kms.Kms {
	return c.kms
}

// Key returns a Key that uses the data key, that must be 32 bytes long,
// represented by encryptedDataKey.
func (c *Cipher) Key(encryptedDataKey []byte) *Key {
	return &Key{slices.Clone(encryptedDataKey), c}
}

// decryptDataKey decrypts encryptedDataKey through the configured KMS.
func (c *Cipher) decryptDataKey(ctx context.Context, encryptedDataKey []byte) (*clearKey, error) {
	if len(encryptedDataKey) == 0 {
		return nil, errors.New("cipher: encrypted data key is empty")
	}
	if clearKey, ok := c.cache.Get(encryptedDataKey); ok {
		return clearKey, nil
	}
	key, err := c.kms.DecryptDataKey(ctx, encryptedDataKey)
	if err != nil {
		return nil, err
	}
	return c.cache.Put(encryptedDataKey, key), nil
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
	_, _ = rand.Read(out[:aead.NonceSize()])
	out = aead.Seal(out, out[:aead.NonceSize()], plaintext, nil)
	return out, nil
}

// decryptWithDataKey decrypts ciphertext with dataKey using AES-GCM.
// ciphertext must be prefixed with the nonce.
func decryptWithDataKey(ciphertext, dataKey []byte) ([]byte, error) {
	if len(dataKey) != 32 {
		return nil, errors.New("cipher: data key must be 32 bytes")
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
		return nil, errors.New("cipher: ciphertext is too short")
	}

	nonce := ciphertext[:aead.NonceSize()]
	sealed := ciphertext[aead.NonceSize():]

	return aead.Open(make([]byte, 0, len(sealed)-aead.Overhead()), nonce, sealed, nil)
}

func hmacSHA256(data, key []byte) [32]byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	var out [32]byte
	mac.Sum(out[:0])
	return out
}

// Key encrypts and decrypts values using an encrypted data key.
// It is safe for concurrent use by multiple goroutines.
type Key struct {
	key    []byte
	cipher *Cipher
}

// Encrypt encrypts plaintext using the key.
func (k *Key) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	return k.cipher.EncryptWithExistingKey(ctx, plaintext, k.key)
}

// Decrypt decrypts ciphertext using the key.
func (k *Key) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	return k.cipher.Decrypt(ctx, ciphertext, k.key)
}

// HMAC computes the HMAC-SHA-256 of data using the key.
func (k *Key) HMAC(ctx context.Context, data []byte) ([32]byte, error) {
	return k.cipher.HMAC(ctx, data, k.key)
}

// IsValid reports whether the key is valid
// It is safe for concurrent use by multiple goroutines.
func (k *Key) IsValid(ctx context.Context) error {
	if size, ok := k.cipher.cache.Exists(k.key); ok {
		if size != 32 {
			return errors.New("cipher: data key must be 32 bytes")
		}
		return nil
	}
	dataKey, err := k.cipher.kms.DecryptDataKey(ctx, k.key)
	if err != nil {
		return err
	}
	defer clear(dataKey)
	if len(dataKey) != 32 {
		return errors.New("cipher: data key must be 32 bytes")
	}
	return nil
}
