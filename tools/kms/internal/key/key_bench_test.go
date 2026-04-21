// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package key

import (
	"context"
	"testing"
)

// BenchmarkGenerateDataKey32 measures GenerateDataKey for 32-byte keys.
func BenchmarkGenerateDataKey32(b *testing.B) {
	benchmarkGenerateDataKey(b, 32)
}

// BenchmarkGenerateDataKey64 measures GenerateDataKey for 64-byte keys.
func BenchmarkGenerateDataKey64(b *testing.B) {
	benchmarkGenerateDataKey(b, 64)
}

func benchmarkGenerateDataKey(b *testing.B, keyLen int) {
	kms, err := New("N0XrhiYaVLpzkmaKHMsHJwqxdplRG1IXUY0jBkXnkHE=")
	if err != nil {
		b.Fatalf("expected nil error, got %v", err)
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		dataKey, encryptedDataKey, err := kms.GenerateDataKey(ctx, keyLen)
		if err != nil {
			b.Fatalf("expected nil error from GenerateDataKey(%d), got %v", keyLen, err)
		}
		if len(dataKey) != keyLen || len(encryptedDataKey) == 0 {
			b.Fatalf("expected %d-byte key and non-empty encrypted key from GenerateDataKey(%d), got %d-byte key and %d-byte encrypted key", keyLen, keyLen, len(dataKey), len(encryptedDataKey))
		}
	}
}

// BenchmarkDecrypt32 measures DecryptDataKey for 32-byte keys.
func BenchmarkDecrypt32(b *testing.B) {
	benchmarkDecrypt(b, 32)
}

// BenchmarkDecrypt64 measures DecryptDataKey for 64-byte keys.
func BenchmarkDecrypt64(b *testing.B) {
	benchmarkDecrypt(b, 64)
}

func benchmarkDecrypt(b *testing.B, keyLen int) {

	kms, err := New("UhVmGa8/q/AdK+YOLo/XAwPSmy81LSKnMLD3dSV3JOU")
	if err != nil {
		b.Fatalf("expected nil error, got %v", err)
	}

	ctx := context.Background()
	_, encryptedDataKey, err := kms.GenerateDataKey(ctx, keyLen)
	if err != nil {
		b.Fatalf("expected nil error from GenerateDataKey(%d), got %v", keyLen, err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		dataKey, err := kms.DecryptDataKey(ctx, encryptedDataKey)
		if err != nil {
			b.Fatalf("expected nil error from DecryptDataKey(%d), got %v", keyLen, err)
		}
		if len(dataKey) != keyLen {
			b.Fatalf("expected %d-byte key from DecryptDataKey(%d), got %d-byte key", keyLen, keyLen, len(dataKey))
		}
	}

}
