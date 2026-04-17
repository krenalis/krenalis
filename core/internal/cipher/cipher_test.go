package cipher

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
)

type testKMS struct {
	keys map[string][]byte
}

func (k *testKMS) GenerateDataKey(_ context.Context, keyLen int) ([]byte, []byte, error) {
	if keyLen != 32 && keyLen != 64 {
		return nil, nil, errors.New("invalid key length")
	}
	dataKey := make([]byte, keyLen)
	for i := range dataKey {
		dataKey[i] = byte(i)
	}
	encryptedDataKey := []byte{byte(keyLen)}
	if k.keys == nil {
		k.keys = map[string][]byte{}
	}
	k.keys[string(encryptedDataKey)] = slices.Clone(dataKey)
	return dataKey, encryptedDataKey, nil
}

func (k *testKMS) DecryptDataKey(_ context.Context, encryptedDataKey []byte) ([]byte, error) {
	dataKey, ok := k.keys[string(encryptedDataKey)]
	if !ok {
		return nil, errors.New("missing encrypted data key")
	}
	return slices.Clone(dataKey), nil
}

// GenerateDataKeyWithoutPlaintext is implemented only to satisfy kms.Kms in
// tests.
func (k *testKMS) GenerateDataKeyWithoutPlaintext(ctx context.Context, keyLen int) ([]byte, error) {
	panic("unexpected GenerateDataKeyWithoutPlaintext call")
}

// TestEncryptDecryptRoundTrip verifies round-trip encryption with a generated
// 32-byte data key.
func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	c := New(&testKMS{})
	t.Cleanup(c.Close)
	ctx := context.Background()
	plaintext := []byte("secret")

	ciphertext, encryptedDataKey, err := c.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(encryptedDataKey) == 0 {
		t.Fatal("expected non-empty encrypted data key, got empty key")
	}

	decrypted, err := c.Decrypt(ctx, ciphertext, encryptedDataKey)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("expected plaintext %q, got %q", plaintext, decrypted)
	}
}

// TestKeyEncryptDecryptRoundTrip verifies round-trip encryption through Key with a 32-byte data key.
func TestKeyEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	c := New(&testKMS{})
	t.Cleanup(c.Close)
	ctx := context.Background()
	plaintext := []byte("payload")

	_, encryptedDataKey, err := c.keyManager.GenerateDataKey(ctx, 32)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	key := c.Key(encryptedDataKey)

	ciphertext, err := key.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	decrypted, err := key.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("expected plaintext %q, got %q", plaintext, decrypted)
	}
}

// TestEncryptUses32ByteDataKey verifies Encrypt always requests a 32-byte data
// key from the KMS.
func TestEncryptUses32ByteDataKey(t *testing.T) {
	t.Parallel()

	c := New(&testKMS{})
	t.Cleanup(c.Close)

	_, encryptedDataKey, err := c.Encrypt(context.Background(), []byte("payload"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !bytes.Equal(encryptedDataKey, []byte{32}) {
		t.Fatalf("expected encrypted data key %v, got %v", []byte{32}, encryptedDataKey)
	}
}

// TestEncryptWithInvalidDataKey rejects unsupported decrypted key lengths.
func TestEncryptWithInvalidDataKey(t *testing.T) {
	t.Parallel()

	c := New(&testKMS{
		keys: map[string][]byte{
			"bad": make([]byte, 63),
			"64":  make([]byte, 64),
		},
	})
	t.Cleanup(c.Close)

	for _, encryptedDataKey := range [][]byte{[]byte("bad"), []byte("64")} {
		_, err := c.EncryptWithExistingKey(context.Background(), []byte("payload"), encryptedDataKey)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "cipher: data key must be 32 bytes" {
			t.Fatalf("expected error %q, got %q", "cipher: data key must be 32 bytes", err.Error())
		}
	}
}
