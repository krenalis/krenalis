// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datacrypt

import (
	"bytes"
	"testing"
)

func testKey64() []byte {
	key := make([]byte, 64)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestNewErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       []byte
		purpose   string
		wantError string
	}{
		{
			name:      "short key",
			key:       make([]byte, 63),
			purpose:   "purpose",
			wantError: "datacrypt: master key must be 64 bytes",
		},
		{
			name:      "long key",
			key:       make([]byte, 65),
			purpose:   "purpose",
			wantError: "datacrypt: master key must be 64 bytes",
		},
		{
			name:      "empty purpose",
			key:       testKey64(),
			purpose:   "",
			wantError: "datacrypt: purpose is empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(test.key, test.purpose)
			if err == nil {
				t.Fatalf("expected error %q, got nil", test.wantError)
			}
			if err.Error() != test.wantError {
				t.Fatalf("expected error %q, got %q", test.wantError, err.Error())
			}
		})
	}

}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	c, err := New(testKey64(), "unit-test")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{name: "empty", plaintext: []byte{}},
		{name: "small", plaintext: []byte("hello")},
		{name: "large", plaintext: bytes.Repeat([]byte("abc"), 1024)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ct, err := c.Encrypt(test.plaintext)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if len(ct) != c.nonceSize+len(test.plaintext)+c.overhead {
				t.Fatalf("expected ciphertext length %d, got %d", c.nonceSize+len(test.plaintext)+c.overhead, len(ct))
			}

			pt, err := c.Decrypt(ct)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if !bytes.Equal(pt, test.plaintext) {
				t.Fatalf("expected plaintext %v, got %v", test.plaintext, pt)
			}
		})
	}
}

func TestDecryptErrors(t *testing.T) {
	t.Parallel()

	c, err := New(testKey64(), "unit-test")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	t.Run("encrypted is too short", func(t *testing.T) {
		t.Parallel()
		const msg = "datacrypt: encrypted is too short"
		_, err := c.Decrypt([]byte{1, 2, 3})
		if err == nil {
			t.Fatalf("expected error %q, got nil", msg)
		}
		if err.Error() != msg {
			t.Fatalf("expected error %q, got %q", msg, err.Error())
		}
	})

	t.Run("tampered encrypted", func(t *testing.T) {
		t.Parallel()

		ct, err := c.Encrypt([]byte("secret"))
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		ct[len(ct)-1] ^= 0x01

		_, err = c.Decrypt(ct)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestDecryptWrongKeyOrPurpose(t *testing.T) {
	t.Parallel()

	key := testKey64()
	c1, err := New(key, "purpose-a")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	ct, err := c1.Encrypt([]byte("data"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	t.Run("wrong key", func(t *testing.T) {
		t.Parallel()
		otherKey := testKey64()
		otherKey[0] ^= 0x01
		c2, err := New(otherKey, "purpose-a")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		_, err = c2.Decrypt(ct)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("wrong purpose", func(t *testing.T) {
		t.Parallel()
		c2, err := New(key, "purpose-b")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		_, err = c2.Decrypt(ct)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestDecryptAppend(t *testing.T) {
	t.Parallel()

	c, err := New(testKey64(), "unit-test")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	t.Run("append to dst", func(t *testing.T) {
		t.Parallel()

		plaintext := []byte("payload")
		ct, err := c.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		dst := []byte("prefix-")
		got, err := c.DecryptAppend(dst, ct)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		want := append([]byte("prefix-"), plaintext...)
		if !bytes.Equal(got, want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("short ciphertext", func(t *testing.T) {
		t.Parallel()
		const msg = "datacrypt: master key must be 64 bytes"
		short := make([]byte, c.nonceSize-1)
		_, err := c.DecryptAppend(nil, short)
		if err == nil {
			t.Fatalf("expected error %q, got nil", msg)
		}
		if err.Error() != msg {
			t.Fatalf("expected error %q, got %q", msg, err.Error())
		}
	})

}
