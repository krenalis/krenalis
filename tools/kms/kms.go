// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package kms provides data key generation and decryption backed by a key
// management service.
//
// It is intended for envelope encryption: callers use a plaintext data key to
// encrypt application data, store only the ciphertext and the KMS-encrypted
// form of the data key, and later decrypt that stored key to recover the same
// plaintext key.
//
// The package selects an implementation from a URI of the form
// "<backend>:<identifier>". The standard backends are "local", for a
// process-local key, and "aws", for AWS KMS.
//
// Package kms does not encrypt application data itself; it only manages data
// encryption keys.
package kms

import (
	"context"
	"errors"
	"strings"

	"github.com/krenalis/krenalis/tools/kms/internal/aws"
	"github.com/krenalis/krenalis/tools/kms/internal/local"
)

// Kms provides data key generation and decryption operations.
// It is safe for concurrent use by multiple goroutines.
type Kms interface {

	// DecryptDataKey decrypts encryptedDataKey and returns the plaintext data key.
	DecryptDataKey(ctx context.Context, encryptedDataKey []byte) (dataKey []byte, err error)

	// GenerateDataKey generates and returns a plaintext data key together with its
	// encrypted form.
	GenerateDataKey(ctx context.Context, keyLen int) (dataKey []byte, encryptedDataKey []byte, err error)

	// GenerateDataKeyWithoutPlaintext generates and returns an encrypted data key
	// without returning its plaintext form.
	GenerateDataKeyWithoutPlaintext(ctx context.Context, keyLen int) (encryptedDataKey []byte, err error)
}

// New returns a Kms selected by uri, which must have the form
// "<backend>:<identifier>".
func New(ctx context.Context, uri string) (Kms, error) {
	backend, identifier, found := strings.Cut(uri, ":")
	if !found {
		return nil, errors.New("kms: uri is invalid")
	}
	var kms Kms
	var err error
	switch backend {
	case "local":
		kms, err = local.New(identifier)
	case "aws":
		kms, err = aws.New(ctx, identifier)
	default:
		err = errors.New("kms: unsupported backend")
	}
	if err != nil {
		return nil, err
	}
	return kms, nil
}
