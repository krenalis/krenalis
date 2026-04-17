// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	kmssdk "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

const testKeyID = "test-key"

// TestNewEmptyKeyID rejects empty AWS KMS key identifiers.
func TestNewEmptyKeyID(t *testing.T) {
	kms, err := New(context.Background(), "")
	if err == nil {
		t.Fatal("expected error from New with empty key ID, got nil")
	}
	if err.Error() != "kms/aws: empty key ID" {
		t.Fatalf("expected empty key ID error, got %v", err)
	}
	if kms != nil {
		t.Fatalf("expected nil Kms from New with empty key ID, got %#v", kms)
	}
}

// TestNewSuccessWithEnv builds a client from minimal AWS environment settings.
func TestNewSuccessWithEnv(t *testing.T) {

	configDir := t.TempDir()
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(configDir, "credentials"))
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(configDir, "config"))

	kms, err := New(context.Background(), testKeyID)
	if err != nil {
		t.Fatalf("expected nil error from New, got %v", err)
	}
	if kms == nil {
		t.Fatal("expected non-nil Kms from New, got nil")
	}
	if kms.client == nil {
		t.Fatal("expected New to initialize the AWS client, got nil client")
	}
	if kms.keyID != testKeyID {
		t.Fatalf("expected keyID %q from New, got %q", testKeyID, kms.keyID)
	}

}

// TestGenerateDataKeyInvalidLength rejects unsupported data key sizes locally.
func TestGenerateDataKeyInvalidLength(t *testing.T) {
	var calls atomic.Int32
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, _, err := kms.GenerateDataKey(context.Background(), 63); err == nil {
		t.Fatal("expected error from GenerateDataKey(63), got nil")
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected GenerateDataKey(63) to avoid HTTP calls, got %d call(s)", got)
	}
}

// TestGenerateDataKeySuccess verifies successful key generation for supported
// sizes.
func TestGenerateDataKeySuccess(t *testing.T) {

	for _, tc := range []struct {
		name       string
		keyLen     int
		ciphertext []byte
	}{
		{name: "32-byte key", keyLen: 32, ciphertext: []byte("encrypted-32")},
		{name: "64-byte key", keyLen: 64, ciphertext: []byte("encrypted-64")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plaintext := bytesOfLength(tc.keyLen)

			kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
				assertGenerateDataKeyRequest(t, r, tc.keyLen)
				writeJSONResponse(t, w, http.StatusOK, map[string]string{
					"Plaintext":      base64.StdEncoding.EncodeToString(plaintext),
					"CiphertextBlob": base64.StdEncoding.EncodeToString(tc.ciphertext),
				})
			})

			dataKey, encryptedDataKey, err := kms.GenerateDataKey(context.Background(), tc.keyLen)
			if err != nil {
				t.Fatalf("expected nil error from GenerateDataKey(%d), got %v", tc.keyLen, err)
			}
			if string(dataKey) != string(plaintext) {
				t.Fatalf("expected plaintext %q from GenerateDataKey(%d), got %q", plaintext, tc.keyLen, dataKey)
			}
			if string(encryptedDataKey) != string(tc.ciphertext) {
				t.Fatalf("expected ciphertext %q from GenerateDataKey(%d), got %q", tc.ciphertext, tc.keyLen, encryptedDataKey)
			}
		})
	}

}

// TestGenerateDataKeyServiceError wraps service failures with the package
// prefix.
func TestGenerateDataKeyServiceError(t *testing.T) {
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertGenerateDataKeyRequest(t, r, 32)
		writeServiceError(t, w, http.StatusInternalServerError, "DependencyTimeoutException", "boom")
	})
	_, _, err := kms.GenerateDataKey(context.Background(), 32)
	assertWrappedErrorContains(t, err, "boom")
}

// TestGenerateDataKeyUnexpectedPlaintextLength rejects unexpected plaintext
// sizes.
func TestGenerateDataKeyUnexpectedPlaintextLength(t *testing.T) {
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertGenerateDataKeyRequest(t, r, 32)
		writeJSONResponse(t, w, http.StatusOK, map[string]string{
			"Plaintext":      base64.StdEncoding.EncodeToString(bytesOfLength(31)),
			"CiphertextBlob": base64.StdEncoding.EncodeToString([]byte("encrypted")),
		})
	})
	if _, _, err := kms.GenerateDataKey(context.Background(), 32); err == nil {
		t.Fatal("expected error from GenerateDataKey(32), got nil")
	} else if err.Error() != "kms/aws: unexpected plaintext data key length" {
		t.Fatalf("expected unexpected plaintext length error, got %v", err)
	}
}

// TestGenerateDataKeyEmptyCiphertext rejects empty encrypted data keys.
func TestGenerateDataKeyEmptyCiphertext(t *testing.T) {
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertGenerateDataKeyRequest(t, r, 32)

		writeJSONResponse(t, w, http.StatusOK, map[string]string{
			"Plaintext":      base64.StdEncoding.EncodeToString(bytesOfLength(32)),
			"CiphertextBlob": "",
		})
	})
	if _, _, err := kms.GenerateDataKey(context.Background(), 32); err == nil {
		t.Fatal("expected error from GenerateDataKey(32), got nil")
	} else if err.Error() != "kms/aws: empty encrypted data key" {
		t.Fatalf("expected empty encrypted key error, got %v", err)
	}
}

// TestGenerateDataKeyInvalidBase64Plaintext wraps deserialization failures.
func TestGenerateDataKeyInvalidBase64Plaintext(t *testing.T) {
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertGenerateDataKeyRequest(t, r, 32)

		writeJSONResponse(t, w, http.StatusOK, map[string]string{
			"Plaintext":      "!!!",
			"CiphertextBlob": base64.StdEncoding.EncodeToString([]byte("encrypted")),
		})
	})
	_, _, err := kms.GenerateDataKey(context.Background(), 32)
	assertWrappedErrorContains(t, err, "failed to base64 decode PlaintextType")
}

// TestGenerateDataKeyWithoutPlaintextInvalidLength rejects unsupported data key
// sizes locally.
func TestGenerateDataKeyWithoutPlaintextInvalidLength(t *testing.T) {
	var calls atomic.Int32
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, err := kms.GenerateDataKeyWithoutPlaintext(context.Background(), 63); err == nil {
		t.Fatal("expected error from GenerateDataKeyWithoutPlaintext(63), got nil")
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected GenerateDataKeyWithoutPlaintext(63) to avoid HTTP calls, got %d call(s)", got)
	}
}

// TestGenerateDataKeyWithoutPlaintextSuccess verifies the dedicated AWS
// operation is used.
func TestGenerateDataKeyWithoutPlaintextSuccess(t *testing.T) {
	for _, tc := range []struct {
		name       string
		keyLen     int
		ciphertext []byte
	}{
		{name: "32-byte key", keyLen: 32, ciphertext: []byte("encrypted-32")},
		{name: "64-byte key", keyLen: 64, ciphertext: []byte("encrypted-64")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
				assertGenerateDataKeyWithoutPlaintextRequest(t, r, tc.keyLen)
				writeJSONResponse(t, w, http.StatusOK, map[string]string{
					"CiphertextBlob": base64.StdEncoding.EncodeToString(tc.ciphertext),
				})
			})

			encryptedDataKey, err := kms.GenerateDataKeyWithoutPlaintext(context.Background(), tc.keyLen)
			if err != nil {
				t.Fatalf("expected nil error from GenerateDataKeyWithoutPlaintext(%d), got %v", tc.keyLen, err)
			}
			if string(encryptedDataKey) != string(tc.ciphertext) {
				t.Fatalf("expected ciphertext %q from GenerateDataKeyWithoutPlaintext(%d), got %q", tc.ciphertext, tc.keyLen, encryptedDataKey)
			}
		})
	}
}

// TestGenerateDataKeyWithoutPlaintextServiceError wraps service failures with
// the package prefix.
func TestGenerateDataKeyWithoutPlaintextServiceError(t *testing.T) {
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertGenerateDataKeyWithoutPlaintextRequest(t, r, 32)
		writeServiceError(t, w, http.StatusInternalServerError, "DependencyTimeoutException", "boom")
	})
	_, err := kms.GenerateDataKeyWithoutPlaintext(context.Background(), 32)
	assertWrappedErrorContains(t, err, "boom")
}

// TestGenerateDataKeyWithoutPlaintextEmptyCiphertext rejects empty encrypted
// data keys.
func TestGenerateDataKeyWithoutPlaintextEmptyCiphertext(t *testing.T) {
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertGenerateDataKeyWithoutPlaintextRequest(t, r, 32)
		writeJSONResponse(t, w, http.StatusOK, map[string]string{
			"CiphertextBlob": "",
		})
	})
	if _, err := kms.GenerateDataKeyWithoutPlaintext(context.Background(), 32); err == nil {
		t.Fatal("expected error from GenerateDataKeyWithoutPlaintext(32), got nil")
	} else if err.Error() != "kms/aws: empty encrypted data key" {
		t.Fatalf("expected empty encrypted key error, got %v", err)
	}
}

// TestGenerateDataKeyWithoutPlaintextInvalidBase64Ciphertext wraps
// deserialization failures.
func TestGenerateDataKeyWithoutPlaintextInvalidBase64Ciphertext(t *testing.T) {
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertGenerateDataKeyWithoutPlaintextRequest(t, r, 32)
		writeJSONResponse(t, w, http.StatusOK, map[string]string{
			"CiphertextBlob": "!!!",
		})
	})
	_, err := kms.GenerateDataKeyWithoutPlaintext(context.Background(), 32)
	assertWrappedErrorContains(t, err, "failed to base64 decode CiphertextType")
}

// TestDecryptEmptyEncryptedDataKey rejects empty ciphertext locally.
func TestDecryptEmptyEncryptedDataKey(t *testing.T) {
	var calls atomic.Int32
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, err := kms.DecryptDataKey(context.Background(), nil); err == nil {
		t.Fatal("expected error from DecryptDataKey(nil), got nil")
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("expected DecryptDataKey(nil) to avoid HTTP calls, got %d call(s)", got)
	}
}

// TestDecryptSuccess verifies successful decryption for supported plaintext
// sizes.
func TestDecryptSuccess(t *testing.T) {
	for _, tc := range []struct {
		name       string
		ciphertext []byte
		plaintext  []byte
	}{
		{name: "32-byte key", ciphertext: []byte("ciphertext-32"), plaintext: bytesOfLength(32)},
		{name: "64-byte key", ciphertext: []byte("ciphertext-64"), plaintext: bytesOfLength(64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
				assertDecryptRequest(t, r, tc.ciphertext)
				writeJSONResponse(t, w, http.StatusOK, map[string]string{
					"Plaintext": base64.StdEncoding.EncodeToString(tc.plaintext),
				})
			})
			dataKey, err := kms.DecryptDataKey(context.Background(), tc.ciphertext)
			if err != nil {
				t.Fatalf("expected nil error from DecryptDataKey, got %v", err)
			}
			if string(dataKey) != string(tc.plaintext) {
				t.Fatalf("expected plaintext %q from DecryptDataKey, got %q", tc.plaintext, dataKey)
			}
		})
	}
}

// TestDecryptServiceError wraps service failures with the package prefix.
func TestDecryptServiceError(t *testing.T) {
	ciphertext := []byte("ciphertext")
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertDecryptRequest(t, r, ciphertext)
		writeServiceError(t, w, http.StatusInternalServerError, "DependencyTimeoutException", "boom")
	})
	_, err := kms.DecryptDataKey(context.Background(), ciphertext)
	assertWrappedErrorContains(t, err, "boom")
}

// TestDecryptUnexpectedPlaintextLength rejects unexpected plaintext sizes.
func TestDecryptUnexpectedPlaintextLength(t *testing.T) {
	ciphertext := []byte("ciphertext")
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertDecryptRequest(t, r, ciphertext)
		writeJSONResponse(t, w, http.StatusOK, map[string]string{
			"Plaintext": base64.StdEncoding.EncodeToString(bytesOfLength(31)),
		})
	})
	if _, err := kms.DecryptDataKey(context.Background(), ciphertext); err == nil {
		t.Fatal("expected error from DecryptDataKey, got nil")
	} else if err.Error() != "kms/aws: unexpected plaintext data key length" {
		t.Fatalf("expected unexpected plaintext length error, got %v", err)
	}
}

// TestDecryptInvalidBase64Plaintext wraps deserialization failures.
func TestDecryptInvalidBase64Plaintext(t *testing.T) {
	ciphertext := []byte("ciphertext")
	kms := newTestKms(t, func(w http.ResponseWriter, r *http.Request) {
		assertDecryptRequest(t, r, ciphertext)
		writeJSONResponse(t, w, http.StatusOK, map[string]string{
			"Plaintext": "!!!",
		})
	})
	_, err := kms.DecryptDataKey(context.Background(), ciphertext)
	assertWrappedErrorContains(t, err, "failed to base64 decode PlaintextType")
}

// newTestKms builds a Kms backed by an in-process fake HTTP transport.
func newTestKms(t *testing.T, handler http.HandlerFunc) *Kms {
	t.Helper()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			recorder := newResponseRecorder()
			handler.ServeHTTP(recorder, r)
			return recorder.response(), nil
		}),
	}

	client := kmssdk.New(kmssdk.Options{
		BaseEndpoint:     awsv2.String("https://kms.test"),
		Credentials:      credentials.NewStaticCredentialsProvider("test-access-key", "test-secret-key", ""),
		HTTPClient:       httpClient,
		Region:           "eu-west-1",
		RetryMaxAttempts: 1,
	})

	return &Kms{
		client: client,
		keyID:  testKeyID,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip dispatches requests to the wrapped transport function.
func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

// responseRecorder implements the subset of ResponseWriter used by these tests.
type responseRecorder struct {
	header http.Header
	body   []byte
	status int
}

// newResponseRecorder returns a minimal ResponseWriter for fake transports.
func newResponseRecorder() *responseRecorder {
	return &responseRecorder{
		header: make(http.Header),
		status: http.StatusOK,
	}
}

// Header returns the response headers.
func (r *responseRecorder) Header() http.Header {
	return r.header
}

// Write appends response bytes to the recorder buffer.
func (r *responseRecorder) Write(body []byte) (int, error) {
	r.body = append(r.body, body...)
	return len(body), nil
}

// WriteHeader records the response status code.
func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
}

// response converts the recorded output into an HTTP response.
func (r *responseRecorder) response() *http.Response {
	return &http.Response{
		Body:       io.NopCloser(bytes.NewReader(r.body)),
		Header:     r.header.Clone(),
		StatusCode: r.status,
	}
}

// assertGenerateDataKeyRequest validates the GenerateDataKey request shape.
func assertGenerateDataKeyRequest(t *testing.T, r *http.Request, keyLen int) {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Fatalf("expected POST request, got %s", r.Method)
	}
	if got := r.Header.Get("Content-Type"); got != "application/x-amz-json-1.1" {
		t.Fatalf("expected Content-Type %q, got %q", "application/x-amz-json-1.1", got)
	}
	if got := r.Header.Get("X-Amz-Target"); got != "TrentService.GenerateDataKey" {
		t.Fatalf("expected X-Amz-Target %q, got %q", "TrentService.GenerateDataKey", got)
	}

	body := decodeRequestBody(t, r)
	assertStringField(t, body["KeyId"], testKeyID)
	switch keyLen {
	case 32:
		assertStringField(t, body["KeySpec"], string(types.DataKeySpecAes256))
		if _, ok := body["NumberOfBytes"]; ok {
			t.Fatalf("expected request body to omit %q, got %v", "NumberOfBytes", body)
		}
	case 64:
		assertNumberField(t, body["NumberOfBytes"], 64)
		if _, ok := body["KeySpec"]; ok {
			t.Fatalf("expected request body to omit %q, got %v", "KeySpec", body)
		}
	default:
		t.Fatalf("unexpected key length %d in test helper", keyLen)
	}

}

// assertDecryptRequest validates the Decrypt request shape.
func assertDecryptRequest(t *testing.T, r *http.Request, ciphertext []byte) {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Fatalf("expected POST request, got %s", r.Method)
	}
	if got := r.Header.Get("Content-Type"); got != "application/x-amz-json-1.1" {
		t.Fatalf("expected Content-Type %q, got %q", "application/x-amz-json-1.1", got)
	}
	if got := r.Header.Get("X-Amz-Target"); got != "TrentService.Decrypt" {
		t.Fatalf("expected X-Amz-Target %q, got %q", "TrentService.Decrypt", got)
	}

	body := decodeRequestBody(t, r)
	assertStringField(t, body["KeyId"], testKeyID)
	assertStringField(t, body["CiphertextBlob"], base64.StdEncoding.EncodeToString(ciphertext))

}

// decodeRequestBody decodes a JSON request body into a map.
func decodeRequestBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	defer r.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("expected nil error while decoding request body, got %v", err)
	}
	return body
}

// writeJSONResponse writes a JSON response with the given status.
func writeJSONResponse(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("expected nil error while encoding response body, got %v", err)
	}
}

// assertGenerateDataKeyWithoutPlaintextRequest validates the
// GenerateDataKeyWithoutPlaintext request shape.
func assertGenerateDataKeyWithoutPlaintextRequest(t *testing.T, r *http.Request, keyLen int) {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Fatalf("expected POST request, got %s", r.Method)
	}
	if got := r.Header.Get("Content-Type"); got != "application/x-amz-json-1.1" {
		t.Fatalf("expected Content-Type %q, got %q", "application/x-amz-json-1.1", got)
	}
	if got := r.Header.Get("X-Amz-Target"); got != "TrentService.GenerateDataKeyWithoutPlaintext" {
		t.Fatalf("expected X-Amz-Target %q, got %q", "TrentService.GenerateDataKeyWithoutPlaintext", got)
	}

	body := decodeRequestBody(t, r)
	assertStringField(t, body["KeyId"], testKeyID)
	switch keyLen {
	case 32:
		assertStringField(t, body["KeySpec"], string(types.DataKeySpecAes256))
		if _, ok := body["NumberOfBytes"]; ok {
			t.Fatalf("expected request body to omit %q, got %v", "NumberOfBytes", body)
		}
	case 64:
		assertNumberField(t, body["NumberOfBytes"], 64)
		if _, ok := body["KeySpec"]; ok {
			t.Fatalf("expected request body to omit %q, got %v", "KeySpec", body)
		}
	default:
		t.Fatalf("unsupported keyLen %d in test helper", keyLen)
	}
}

// writeServiceError writes an AWS-style JSON error response.
func writeServiceError(t *testing.T, w http.ResponseWriter, status int, errorType, message string) {
	t.Helper()
	w.Header().Set("X-Amzn-ErrorType", errorType)
	writeJSONResponse(t, w, status, map[string]string{"message": message})
}

// assertWrappedErrorContains checks for the package prefix and a message fragment.
func assertWrappedErrorContains(t *testing.T, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "kms/aws: ") {
		t.Fatalf("expected error to start with %q, got %v", "kms/aws: ", err)
	}
	if !strings.Contains(err.Error(), substring) {
		t.Fatalf("expected error to contain %q, got %v", substring, err)
	}
}

// assertStringField checks a JSON string field value.
func assertStringField(t *testing.T, value any, expected string) {
	t.Helper()
	got, ok := value.(string)
	if !ok {
		t.Fatalf("expected string field %q, got %T (%v)", expected, value, value)
	}
	if got != expected {
		t.Fatalf("expected string field %q, got %q", expected, got)
	}
}

// assertNumberField checks a JSON numeric field value.
func assertNumberField(t *testing.T, value any, expected float64) {
	t.Helper()
	got, ok := value.(float64)
	if !ok {
		t.Fatalf("expected numeric field %v, got %T (%v)", expected, value, value)
	}
	if got != expected {
		t.Fatalf("expected numeric field %v, got %v", expected, got)
	}
}

// bytesOfLength returns deterministic bytes for the requested length.
func bytesOfLength(length int) []byte {
	data := make([]byte, length)
	for i := range data {
		data[i] = byte(i)
	}
	return data
}
