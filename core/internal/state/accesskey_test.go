// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/core/internal/cipher"
)

var (
	accessKeyTestEncryptedPepper = []byte("api-key-pepper")
	accessKeyTestPepper          = []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	}
)

type accessKeyTestKMS struct{}

func (accessKeyTestKMS) DecryptDataKey(_ context.Context, encryptedDataKey []byte) ([]byte, error) {
	if !bytes.Equal(encryptedDataKey, accessKeyTestEncryptedPepper) {
		return nil, errors.New("unexpected encrypted data key")
	}
	return append([]byte(nil), accessKeyTestPepper...), nil
}

func (accessKeyTestKMS) GenerateDataKey(_ context.Context, keyLen int) ([]byte, []byte, error) {
	panic("unexpected GenerateDataKey call")
}

func (accessKeyTestKMS) GenerateDataKeyWithoutPlaintext(_ context.Context, keyLen int) ([]byte, error) {
	panic("unexpected GenerateDataKeyWithoutPlaintext call")
}

// TestTokenAllocations checks the allocation budget of access key helpers.
func TestTokenAllocations(t *testing.T) {
	payload := testPayload()
	var payloadBase62 [accessKeyPayloadBase62Size]byte
	var body string
	formatAllocs := testing.AllocsPerRun(1000, func() {
		body = formatAccessKey(payload[:])
	})
	if formatAllocs > 2 {
		t.Fatalf("expected Format to allocate at most twice, got %.1f", formatAllocs)
	}
	if len(body) != accessKeyBodySize {
		t.Fatalf("expected %d-byte body, got %d", accessKeyBodySize, len(body))
	}
	encodeAllocs := testing.AllocsPerRun(1000, func() {
		encodeFixedBase62(payloadBase62[:], payload[:])
	})
	if encodeAllocs != 0 {
		t.Fatalf("expected encodeFixedBase62 not to allocate, got %.1f", encodeAllocs)
	}

	var parsed [32]byte
	parseAllocs := testing.AllocsPerRun(1000, func() {
		if err := parseAccessKey(parsed[:], body); err != nil {
			t.Fatal(err)
		}
	})
	if parseAllocs != 0 {
		t.Fatalf("expected Parse not to allocate, got %.1f", parseAllocs)
	}
	if parsed != payload {
		t.Fatalf("expected payload %x, got %x", payload, parsed)
	}
	decodeAllocs := testing.AllocsPerRun(1000, func() {
		if err := decodeFixedBase62(parsed[:], body[:accessKeyPayloadBase62Size]); err != nil {
			t.Fatal(err)
		}
	})
	if decodeAllocs != 0 {
		t.Fatalf("expected decodeFixedBase62 not to allocate, got %.1f", decodeAllocs)
	}
}

// TestGenerateAccessKey checks that generated keys authenticate their payload.
func TestGenerateAccessKey(t *testing.T) {
	ctx := context.Background()
	state := newAccessKeyTestState(t)
	body, hmac, err := state.GenerateAccessKey(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(body) != accessKeyBodySize {
		t.Fatalf("expected %d-byte body, got %d", accessKeyBodySize, len(body))
	}
	if len(hmac) != 32 {
		t.Fatalf("expected 32-byte HMAC, got %d", len(hmac))
	}
	var payload [32]byte
	if err := parseAccessKey(payload[:], body); err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
	expected, err := state.metadata.apiKeyPepper.HMAC(ctx, payload[:])
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !bytes.Equal(hmac, expected[:]) {
		t.Fatal("expected HMAC of generated payload")
	}
}

// TestStateAccessKey checks HMAC-backed access key lookup.
func TestStateAccessKey(t *testing.T) {
	ctx := context.Background()
	state := newAccessKeyTestState(t)
	payload := testPayload()
	body := formatAccessKey(payload[:])
	hmac, err := state.metadata.apiKeyPepper.HMAC(ctx, payload[:])
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	key := &AccessKey{ID: 42, Type: AccessKeyTypeAPI}
	state.accessKeyByHMAC[string(hmac[:])] = key
	got, err := state.AccessKey(ctx, body)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != key {
		t.Fatal("expected access key from HMAC lookup")
	}

	if _, err := state.AccessKey(ctx, body[:len(body)-1]); err != ErrInvalidAccessKeyFormat {
		t.Fatalf("expected ErrInvalidAccessKeyFormat for short token, got %v", err)
	}
	badCRC := body[:len(body)-1] + string(otherBase62Char(body[len(body)-1]))
	if _, err := state.AccessKey(ctx, badCRC); err != ErrInvalidAccessKeyFormat {
		t.Fatalf("expected ErrInvalidAccessKeyFormat for bad CRC, got %v", err)
	}
	other := maxPayload()
	otherBody := formatAccessKey(other[:])
	if _, err := state.AccessKey(ctx, otherBody); err != ErrAccessKeyNotFound {
		t.Fatalf("expected ErrAccessKeyNotFound, got %v", err)
	}
}

// TestFormatParseRoundTrip checks that access key bodies round-trip.
func TestFormatParseRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		payload [32]byte
	}{
		{name: "zero"},
		{name: "sequence", payload: testPayload()},
		{name: "max", payload: maxPayload()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := formatAccessKey(tt.payload[:])
			if len(body) != accessKeyBodySize {
				t.Fatalf("expected %d-byte body, got %d", accessKeyBodySize, len(body))
			}
			var got [32]byte
			if err := parseAccessKey(got[:], body); err != nil {
				t.Fatalf("expected valid token, got %v", err)
			}
			if got != tt.payload {
				t.Fatalf("expected payload %x, got %x", tt.payload, got)
			}
		})
	}
}

// TestFormatAccessKeyKnownVector checks the fixed access key wire format.
func TestFormatAccessKeyKnownVector(t *testing.T) {
	payload := testPayload()
	const expected = "AOyrCBBQEmI63fgWAAw8F12XcPvUqkg0aEwlIqnO0xyCeS3NT"
	if got := formatAccessKey(payload[:]); got != expected {
		t.Fatalf("expected token body %q, got %q", expected, got)
	}
	var got [32]byte
	if err := parseAccessKey(got[:], expected); err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
	if got != payload {
		t.Fatalf("expected payload %x, got %x", payload, got)
	}
}

// TestFormatAccessKeyRejectsWrongPayloadSize checks its payload precondition.
func TestFormatAccessKeyRejectsWrongPayloadSize(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	formatAccessKey(make([]byte, accessKeyPayloadSize-1))
}

// TestParseRejectsInvalidToken checks malformed access key bodies.
func TestParseRejectsInvalidToken(t *testing.T) {
	payload := testPayload()
	body := formatAccessKey(payload[:])
	tests := []struct {
		name  string
		token string
	}{
		{name: "invalid character", token: body[:10] + "_" + body[11:]},
		{name: "crc mismatch", token: body[:len(body)-1] + string(otherBase62Char(body[len(body)-1]))},
		{name: "payload overflow", token: "9" + body[1:]},
		{name: "crc overflow", token: body[:accessKeyPayloadBase62Size] + "9" + body[accessKeyPayloadBase62Size+1:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := payload
			if err := parseAccessKey(got[:], tt.token); err == nil {
				t.Fatal("expected invalid token error, got nil")
			}
		})
	}
}

// TestEncodeFixedBase62ReportsOverflow checks that encoding panics on overflow.
func TestEncodeFixedBase62ReportsOverflow(t *testing.T) {
	payload := maxPayload()
	var out [accessKeyPayloadBase62Size - 1]byte
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	encodeFixedBase62(out[:], payload[:])
}

// TestEncodeFixedBase62Fits checks that a 32-byte value fits in its token
// field.
func TestEncodeFixedBase62Fits(t *testing.T) {
	payload := maxPayload()
	var out [accessKeyPayloadBase62Size]byte
	defer func() {
		if value := recover(); value != nil {
			t.Fatalf("expected no panic, got %v", value)
		}
	}()
	encodeFixedBase62(out[:], payload[:])
}

// TestEncodeFixedBase62CRC32Fits checks that a CRC32 fits in its token field.
func TestEncodeFixedBase62CRC32Fits(t *testing.T) {
	var crc [4]byte
	var out [accessKeyCRC32Base62Size]byte
	defer func() {
		if value := recover(); value != nil {
			t.Fatalf("expected no panic, got %v", value)
		}
	}()
	for i := range crc {
		crc[i] = 0xff
	}
	encodeFixedBase62(out[:], crc[:])
}

// BenchmarkFormat measures access key body formatting.
func BenchmarkFormat(b *testing.B) {
	payload := testPayload()
	var body string
	b.ReportAllocs()
	for b.Loop() {
		body = formatAccessKey(payload[:])
	}
	if len(body) != accessKeyBodySize {
		b.Fatalf("expected %d-byte body, got %d", accessKeyBodySize, len(body))
	}
}

// BenchmarkParse measures access key body parsing.
func BenchmarkParse(b *testing.B) {
	payload := testPayload()
	body := formatAccessKey(payload[:])
	var parsed [32]byte
	b.ReportAllocs()
	for b.Loop() {
		if err := parseAccessKey(parsed[:], body); err != nil {
			b.Fatal(err)
		}
	}
	if parsed != payload {
		b.Fatalf("expected payload %x, got %x", payload, parsed)
	}
}

// BenchmarkEncodeFixedBase62 measures fixed-width base62 encoding.
func BenchmarkEncodeFixedBase62(b *testing.B) {
	payload := testPayload()
	var out [accessKeyPayloadBase62Size]byte
	b.ReportAllocs()
	for b.Loop() {
		encodeFixedBase62(out[:], payload[:])
	}
}

// BenchmarkDecodeFixedBase62 measures fixed-width base62 decoding.
func BenchmarkDecodeFixedBase62(b *testing.B) {
	payload := testPayload()
	body := formatAccessKey(payload[:])
	var out [32]byte
	b.ReportAllocs()
	for b.Loop() {
		if err := decodeFixedBase62(out[:], body[:accessKeyPayloadBase62Size]); err != nil {
			b.Fatal(err)
		}
	}
	if out != payload {
		b.Fatalf("expected payload %x, got %x", payload, out)
	}
}

// testPayload returns a non-zero 32-byte payload.
func testPayload() [32]byte {
	var payload [32]byte
	for i := range payload {
		payload[i] = byte(i + 1)
	}
	return payload
}

// maxPayload returns the largest 32-byte payload.
func maxPayload() [32]byte {
	var payload [32]byte
	for i := range payload {
		payload[i] = 0xff
	}
	return payload
}

// otherBase62Char returns a different valid base62 byte.
func otherBase62Char(c byte) byte {
	if c == 'A' {
		return 'B'
	}
	return 'A'
}

func newAccessKeyTestState(t *testing.T) *State {
	t.Helper()
	c := cipher.New(accessKeyTestKMS{})
	t.Cleanup(c.Close)
	return &State{
		cipher: c,
		mu:     new(sync.Mutex),
		metadata: metadata{
			apiKeyPepper: c.Key(accessKeyTestEncryptedPepper),
		},
		accessKeyByHMAC: map[string]*AccessKey{},
	}
}
