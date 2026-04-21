// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package key

import (
	"context"
	"testing"
)

// TestGenerateDataKey verifies generation and decryption for supported key
// sizes.
func TestGenerateDataKey(t *testing.T) {

	kms, err := New("DJ0UMRTROH4pjY/Esh3fAErsPbdYmvsnfDCZtc9K4iU")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	ctx := context.Background()

	for _, keyLen := range []int{32, 64} {
		dataKey, encryptedDataKey, err := kms.GenerateDataKey(ctx, keyLen)
		if err != nil {
			t.Fatalf("expected nil error from GenerateDataKey(%d), got %v", keyLen, err)
		}
		if len(dataKey) != keyLen {
			t.Fatalf("expected %d-byte key from GenerateDataKey(%d), got %d-byte key", keyLen, keyLen, len(dataKey))
		}
		if len(encryptedDataKey) == 0 {
			t.Fatalf("expected non-empty encrypted key from GenerateDataKey(%d), got empty key", keyLen)
		}
		decryptedDataKey, err := kms.DecryptDataKey(ctx, encryptedDataKey)
		if err != nil {
			t.Fatalf("expected nil error from DecryptDataKey(%d), got %v", keyLen, err)
		}
		if len(decryptedDataKey) != keyLen {
			t.Fatalf("expected %d-byte key from DecryptDataKey(%d), got %d-byte key", keyLen, keyLen, len(decryptedDataKey))
		}
		if string(decryptedDataKey) != string(dataKey) {
			t.Fatalf("expected decrypted key to match original key for %d-byte key, got different key", keyLen)
		}
	}

}

// TestGenerateDataKeyInvalidLength rejects unsupported key sizes.
func TestGenerateDataKeyInvalidLength(t *testing.T) {
	kms, err := New("zTN/cldUcn2QnOSmoLpzPXZMweoOzUUWashzpXQk0NA=")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if _, _, err := kms.GenerateDataKey(context.Background(), 63); err == nil {
		t.Fatal("expected error from GenerateDataKey(63), got nil")
	}
}
