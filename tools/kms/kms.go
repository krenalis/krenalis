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

var InvalidKeyError = errors.New("invalid key")

// Kms provides data key generation and decryption operations.
type Kms interface {
	DecryptDataKey(ctx context.Context, encryptedDataKey []byte) (dataKey []byte, err error)
	GenerateDataKey(ctx context.Context, keyLen int) (dataKey []byte, encryptedDataKey []byte, err error)
	GenerateDataKeyWithoutPlaintext(ctx context.Context, keyLen int) (encryptedDataKey []byte, err error)
}

// New returns a Kms selected by uri, which must have the form
// "<backend>:<identifier>".
func New(ctx context.Context, uri string) (Kms, error) {
	backend, identifier, found := strings.Cut(uri, ":")
	if !found {
		return nil, errors.New("kms: dns is invalid")
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
