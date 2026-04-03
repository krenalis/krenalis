// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test_httpLogger_Write checks that httpLogger.Write logs expected messages and
// skips unwanted ones.
func Test_httpLogger_Write(t *testing.T) {

	handler := &captureHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(orig)

	l := &httpLogger{}

	tests := []struct {
		name    string
		input   []byte
		wantMsg string
	}{
		{"empty", []byte{}, ""},
		{"tls without newline", append(tlsHandshakeMsg, []byte("1.2.3.4:1: EOF")...), ""},
		{"tls with newline", append(append([]byte{}, tlsHandshakeMsg...), []byte("1.2.3.4:1: EOF\n")...), ""},
		{"trim newline", []byte("hello world\n"), "hello world"},
		{"plain", []byte("simple log"), "simple log"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler.reset()
			n, err := l.Write(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if n != len(tt.input) {
				t.Fatalf("expected %d bytes written, got %d", len(tt.input), n)
			}
			got := handler.Messages()
			if tt.wantMsg == "" {
				if len(got) != 0 {
					t.Errorf("expected no log message, got %v", got)
				}
			} else {
				if len(got) != 1 {
					t.Errorf("expected one log message, got %v", got)
					return
				}
				if got[0] != tt.wantMsg {
					t.Errorf("expected %q, got %q", tt.wantMsg, got[0])
				}
			}
		})
	}

}

// Test_verifyCertificate checks that verifyCertificate maps certificate
// validation failures to the expected error messages.
func Test_verifyCertificate(t *testing.T) {

	t.Run("nil leaf is ignored", func(t *testing.T) {
		err := verifyCertificate(tls.Certificate{}, "example.com", nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("hostname mismatch", func(t *testing.T) {
		cert, roots := newTestTLSCertificate(t, testTLSCertificateOptions{
			dnsNames:  []string{"example.com"},
			notBefore: time.Now().Add(-time.Hour),
			notAfter:  time.Now().Add(time.Hour),
		})

		err := verifyCertificate(cert, "wrong.example.com", roots)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		want := `server TLS certificate is not valid for the hostname "wrong.example.com"`
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("unknown authority", func(t *testing.T) {
		cert, _ := newTestTLSCertificate(t, testTLSCertificateOptions{
			dnsNames:  []string{"example.com"},
			notBefore: time.Now().Add(-time.Hour),
			notAfter:  time.Now().Add(time.Hour),
		})

		err := verifyCertificate(cert, "example.com", x509.NewCertPool())
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		want := "server TLS certificate is not trusted by system CA"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("expired certificate", func(t *testing.T) {
		cert, roots := newTestTLSCertificate(t, testTLSCertificateOptions{
			dnsNames:  []string{"example.com"},
			notBefore: time.Now().Add(-2 * time.Hour),
			notAfter:  time.Now().Add(-time.Hour),
		})

		err := verifyCertificate(cert, "example.com", roots)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		want := "server TLS certificate has expired"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("malformed intermediate", func(t *testing.T) {
		cert, roots := newTestTLSCertificate(t, testTLSCertificateOptions{
			dnsNames:          []string{"example.com"},
			notBefore:         time.Now().Add(-time.Hour),
			notAfter:          time.Now().Add(time.Hour),
			intermediateBytes: [][]byte{[]byte("not-a-certificate")},
		})

		err := verifyCertificate(cert, "example.com", roots)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse intermediate certificate") {
			t.Fatalf("expected intermediate parse error, got %q", err)
		}
	})
}

type testTLSCertificateOptions struct {
	dnsNames          []string
	notBefore         time.Time
	notAfter          time.Time
	intermediateBytes [][]byte
}

func newTestTLSCertificate(t *testing.T, opts testTLSCertificateOptions) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("cannot generate root key: %v", err)
	}

	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root CA"},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, rootKey.Public(), rootKey)
	if err != nil {
		t.Fatalf("cannot create root certificate: %v", err)
	}

	rootCert, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatalf("cannot parse root certificate: %v", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("cannot generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: opts.dnsNames[0]},
		DNSNames:              append([]string(nil), opts.dnsNames...),
		NotBefore:             opts.notBefore,
		NotAfter:              opts.notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, rootCert, key.Public(), rootKey)
	if err != nil {
		t.Fatalf("cannot create certificate: %v", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("cannot parse certificate: %v", err)
	}

	cert := tls.Certificate{
		Certificate: append([][]byte{der}, opts.intermediateBytes...),
		PrivateKey:  key,
		Leaf:        leaf,
	}

	roots := x509.NewCertPool()
	roots.AddCert(rootCert)

	return cert, roots
}

// captureHandler is a slog.Handler that records log messages and their levels.
type captureHandler struct {
	mu   sync.Mutex
	msgs []slog.Record
}

func (h *captureHandler) Enabled(ctx context.Context, level slog.Level) bool { return true }

func (h *captureHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Make a copy of the record, since slog.Record may be reused internally.
	rec := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	h.msgs = append(h.msgs, rec)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(name string) slog.Handler       { return h }

func (h *captureHandler) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.msgs = h.msgs[:0]
}

// Messages returns a copy of the recorded messages.
func (h *captureHandler) Messages() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	msgs := make([]string, len(h.msgs))
	for i, rec := range h.msgs {
		msgs[i] = rec.Message
	}
	return msgs
}
