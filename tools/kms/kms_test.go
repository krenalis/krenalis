// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package kms

import (
	"context"
	"testing"
)

// TestNewErrors verifies New rejects invalid or unsupported URIs.
func TestNewErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		uri  string
		want string
	}{
		{name: "invalid URI", uri: "local", want: "kms: uri is invalid"},
		{name: "unsupported backend", uri: "gcp:key", want: "kms: unsupported backend"},
		{name: "empty AWS key ID", uri: "aws:", want: "kms/aws: empty key ID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kms, err := New(context.Background(), tc.uri)
			if err == nil {
				t.Fatalf("expected error from New(%q), got nil", tc.uri)
			}
			if err.Error() != tc.want {
				t.Fatalf("expected error %q from New(%q), got %v", tc.want, tc.uri, err)
			}
			if kms != nil {
				t.Fatalf("expected nil Kms from New(%q), got %#v", tc.uri, kms)
			}
		})
	}
}

// TestNewLocalSuccess verifies New returns a working local backend.
func TestNewLocalSuccess(t *testing.T) {

	kms, err := New(context.Background(), "local:DJ0UMRTROH4pjY/Esh3fAErsPbdYmvsnfDCZtc9K4iU")
	if err != nil {
		t.Fatalf("expected nil error from New(local), got %v", err)
	}
	if kms == nil {
		t.Fatal("expected non-nil Kms from New(local), got nil")
	}

	dataKey, encryptedDataKey, err := kms.GenerateDataKey(context.Background(), 32)
	if err != nil {
		t.Fatalf("expected nil error from GenerateDataKey(32), got %v", err)
	}
	if len(dataKey) != 32 {
		t.Fatalf("expected 32-byte key from GenerateDataKey(32), got %d-byte key", len(dataKey))
	}
	if len(encryptedDataKey) == 0 {
		t.Fatal("expected non-empty encrypted key from GenerateDataKey(32), got empty key")
	}

	decryptedDataKey, err := kms.DecryptDataKey(context.Background(), encryptedDataKey)
	if err != nil {
		t.Fatalf("expected nil error from DecryptDataKey, got %v", err)
	}
	if string(decryptedDataKey) != string(dataKey) {
		t.Fatalf("expected decrypted key to match original key, got %q and %q", decryptedDataKey, dataKey)
	}

}
