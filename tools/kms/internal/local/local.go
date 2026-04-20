// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package local

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

type Kms struct {
	aead cipher.AEAD
}

func New(base64Key string) (*Kms, error) {
	key, err := base64.RawStdEncoding.DecodeString(strings.TrimSuffix(base64Key, "="))
	if err != nil {
		return nil, errors.New("kms/local: key is not valid Base64")
	}
	defer clear(key)
	if len(key) != 32 {
		return nil, errors.New("kms/local: key must be 32 bytes long")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("kms/local: %s", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("kms/local: %s", err)
	}
	return &Kms{aead: aead}, nil
}

func (k *Kms) GenerateDataKey(_ context.Context, keyLen int) ([]byte, []byte, error) {

	if keyLen != 32 && keyLen != 64 {
		return nil, nil, errors.New("kms/local: invalid key length")
	}

	nonceSize := k.aead.NonceSize()
	overhead := k.aead.Overhead()

	dataKey := make([]byte, keyLen)
	rand.Read(dataKey)

	encryptedDataKey := make([]byte, nonceSize+keyLen+overhead)
	nonce := encryptedDataKey[:nonceSize]
	rand.Read(nonce)

	encryptedDataKey = k.aead.Seal(encryptedDataKey[:nonceSize], nonce, dataKey, nil)

	return dataKey, encryptedDataKey, nil
}

func (k *Kms) GenerateDataKeyWithoutPlaintext(ctx context.Context, keyLen int) ([]byte, error) {
	dataKey, encryptedDataKey, err := k.GenerateDataKey(ctx, keyLen)
	if err != nil {
		return nil, err
	}
	clear(dataKey)
	return encryptedDataKey, nil
}

func (k *Kms) DecryptDataKey(_ context.Context, encryptedDataKey []byte) ([]byte, error) {

	nonceSize := k.aead.NonceSize()
	overhead := k.aead.Overhead()
	if len(encryptedDataKey) < nonceSize+overhead {
		return nil, errors.New("kms/local: encrypted key too short")
	}

	nonce := encryptedDataKey[:nonceSize]
	ciphertext := encryptedDataKey[nonceSize:]
	dataKeyLen := len(ciphertext) - overhead
	if dataKeyLen != 32 && dataKeyLen != 64 {
		return nil, errors.New("kms/local: unexpected plaintext data key length")
	}

	dataKey := make([]byte, dataKeyLen)
	return k.aead.Open(dataKey[:0], nonce, ciphertext, nil)
}
