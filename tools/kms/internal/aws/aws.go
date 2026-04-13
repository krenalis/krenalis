// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type Kms struct {
	client *kms.Client
	keyID  string
}

// New creates an AWS KMS-backed implementation of the Kms interface.
//
// It builds the AWS KMS client using the AWS SDK default configuration chain,
// so credentials, region, and related settings are resolved through the
// standard AWS mechanisms.
//
// keyID identifies the AWS KMS key used to manage data keys.
func New(ctx context.Context, keyID string) (*Kms, error) {
	if keyID == "" {
		return nil, errors.New("kms/aws: empty key ID")
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("kms/aws: %s", err)
	}
	return &Kms{
		client: kms.NewFromConfig(cfg),
		keyID:  keyID,
	}, nil
}

func (k *Kms) GenerateDataKey(ctx context.Context, keyLen int) ([]byte, []byte, error) {

	params := &kms.GenerateDataKeyInput{
		KeyId: &k.keyID,
	}
	switch keyLen {
	case 32:
		params.KeySpec = types.DataKeySpecAes256
	case 64:
		params.NumberOfBytes = new(int32(keyLen))
	default:
		return nil, nil, errors.New("kms/aws: invalid key length")
	}

	out, err := k.client.GenerateDataKey(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("kms/aws: %s", err)
	}
	if len(out.Plaintext) != keyLen {
		return nil, nil, errors.New("kms/aws: unexpected plaintext data key length")
	}
	if len(out.CiphertextBlob) == 0 {
		return nil, nil, errors.New("kms/aws: empty encrypted data key")
	}

	return out.Plaintext, out.CiphertextBlob, nil
}

func (k *Kms) GenerateDataKeyWithoutPlaintext(ctx context.Context, keyLen int) ([]byte, error) {
	params := &kms.GenerateDataKeyWithoutPlaintextInput{
		KeyId: &k.keyID,
	}
	switch keyLen {
	case 32:
		params.KeySpec = types.DataKeySpecAes256
	case 64:
		params.NumberOfBytes = new(int32)
		*params.NumberOfBytes = int32(keyLen)
	default:
		return nil, errors.New("kms/aws: invalid key length")
	}

	out, err := k.client.GenerateDataKeyWithoutPlaintext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("kms/aws: %s", err)
	}
	if len(out.CiphertextBlob) == 0 {
		return nil, errors.New("kms/aws: empty encrypted data key")
	}

	return out.CiphertextBlob, nil
}

func (k *Kms) DecryptDataKey(ctx context.Context, encryptedDataKey []byte) ([]byte, error) {

	if len(encryptedDataKey) == 0 {
		return nil, errors.New("kms/aws: empty encrypted data key")
	}

	out, err := k.client.Decrypt(ctx, &kms.DecryptInput{
		KeyId:          &k.keyID,
		CiphertextBlob: encryptedDataKey,
	})
	if err != nil {
		return nil, fmt.Errorf("kms/aws: %s", err)
	}

	switch len(out.Plaintext) {
	case 32, 64:
	default:
		return nil, errors.New("kms/aws: unexpected plaintext data key length")
	}

	return out.Plaintext, nil
}
