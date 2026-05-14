// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"

	"golang.org/x/crypto/ssh"
)

const testHostPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICM4A2h7XwqPj7jY6n4xWm5mWw0kD9Nh9NP7VDaRao7I test@example.com"
const testRSAHostPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7ER2IzZQHHB5wK6IsXF5vYzC5ICsTrhzWnEnTqRx+d6AqNh+NirjQ8k6tHgGx5v/UtgTiLT2Z3NVvNKY3QBD5VOWAV07MiHZHBsCcK3w6Er4CFSin7raXWcAK/WVhaISx5E6qLuVP84nQW+CIHYHcuQ3StM3y3rbSGTxEEQW0lXavrvwrAyiPFbClmxBgrPYq5HRjMFnndShS6nSMkZgkPdD64sv/9T+2lGusxTS5KFtK7EP9giQmj4sM8PKW/c/c8mBsRcd0DlPrWIeV1spDUg0pvGyZsSo2F0zvCp/+5MRAI0SH7vGXRNYAUhGqz7uqoUuP4m2P97dhU3BhE1sX test@example.com"

type testSettingsStore struct {
	settings json.Value
}

// newTestSettingsStore returns a test settings store.
func newTestSettingsStore(t *testing.T, settings any) *testSettingsStore {
	t.Helper()
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("cannot marshal test settings: %s", err)
	}
	return &testSettingsStore{settings: data}
}

// Load unmarshals the stored settings into dst.
func (s *testSettingsStore) Load(ctx context.Context, dst any) error {
	return json.Unmarshal(s.settings, dst)
}

// Store replaces the stored settings with src.
func (s *testSettingsStore) Store(ctx context.Context, src any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	s.settings = data
	return nil
}

// TestHostKeyValidatorMismatch tests host key mismatch detection.
func TestHostKeyValidatorMismatch(t *testing.T) {
	expected, err := parseHostPublicKey(testHostPublicKey)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	other, err := parseHostPublicKey("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHK9aX4A6wqTiSEPAxS0x1QF6P6T4L+6vB8m0uY7JYyV test@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	err = hostKeyValidator(expected)("", nil, other)
	if !errors.Is(err, errHostKeyMismatch) {
		t.Fatalf("expected error matching errHostKeyMismatch, got %v", err)
	}
}

// TestNewSSHClientConfigPinnedHostKey tests SSH client configuration for a
// pinned host public key.
func TestNewSSHClientConfigPinnedHostKey(t *testing.T) {
	sshConfig, err := newSSHClientConfig(&innerSettings{
		Username:      "username",
		Password:      "password",
		HostPublicKey: testHostPublicKey,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(sshConfig.HostKeyAlgorithms) != 1 {
		t.Fatalf("expected HostKeyAlgorithms length 1, got %d", len(sshConfig.HostKeyAlgorithms))
	}
	if sshConfig.HostKeyAlgorithms[0] != ssh.KeyAlgoED25519 {
		t.Fatalf("expected HostKeyAlgorithms[0] %q, got %q", ssh.KeyAlgoED25519, sshConfig.HostKeyAlgorithms[0])
	}
}

// TestNewSSHClientConfigPinnedRSAHostKey tests SSH client configuration for a
// pinned RSA host public key.
func TestNewSSHClientConfigPinnedRSAHostKey(t *testing.T) {
	sshConfig, err := newSSHClientConfig(&innerSettings{
		Username:      "username",
		Password:      "password",
		HostPublicKey: testRSAHostPublicKey,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	expected := []string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA}
	if len(sshConfig.HostKeyAlgorithms) != len(expected) {
		t.Fatalf("expected HostKeyAlgorithms length %d, got %d", len(expected), len(sshConfig.HostKeyAlgorithms))
	}
	for i, algo := range expected {
		if sshConfig.HostKeyAlgorithms[i] != algo {
			t.Fatalf("expected HostKeyAlgorithms[%d] %q, got %q", i, algo, sshConfig.HostKeyAlgorithms[i])
		}
	}
}

// TestOpenClientRejectsInvalidHostPublicKey tests that openClient rejects host
// public keys that do not satisfy the connector validation rules.
func TestOpenClientRejectsInvalidHostPublicKey(t *testing.T) {
	_, err := openClient(context.Background(), &innerSettings{
		Host:          "example.com",
		Port:          22,
		Username:      "username",
		Password:      "password",
		HostPublicKey: `command="echo hi" ` + testHostPublicKey,
	})
	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	if !strings.Contains(err.Error(), "public key options are not allowed") {
		t.Fatalf("expected error containing %q, got %q", "public key options are not allowed", err.Error())
	}
}

// TestParseAndValidateHostPublicKey tests host public key parsing and
// validation.
func TestParseAndValidateHostPublicKey(t *testing.T) {
	if _, err := parseHostPublicKey(testHostPublicKey); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestPathConvert tests path conversion to SFTP URLs.
func TestPathConvert(t *testing.T) {
	sf := &SFTP{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{Host: "example.com", Port: 22})}}
	tests := []testconnector.AbsolutePathTest{
		{Name: "/a", Expected: "sftp://example.com:22/a"},
		{Name: "/a/b", Expected: "sftp://example.com:22/a/b"},
		{Name: "a", Expected: "sftp://example.com:22/a"},
		{Name: "/\x00", Expected: "sftp://example.com:22/%00"},
	}
	err := testconnector.TestAbsolutePath(sf, tests)
	if err != nil {
		t.Errorf("SFTP connector: %s", err)
	}
}

// TestSaveSettingsHostPublicKeyValidation tests host public key validation in
// saveSettings.
func TestSaveSettingsHostPublicKeyValidation(t *testing.T) {
	sf := &SFTP{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{})}}
	settings, err := json.Marshal(innerSettings{
		Host:          "example.com",
		Port:          22,
		Username:      "username",
		Password:      "password",
		HostPublicKey: "not a public key",
	})
	if err != nil {
		t.Fatalf("cannot marshal settings: %s", err)
	}
	err = sf.saveSettings(context.Background(), settings, connectors.Source, true)
	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	if err.Error() != "server public key must be a valid OpenSSH public key" {
		t.Fatalf("expected error %q, got %q", "server public key must be a valid OpenSSH public key", err.Error())
	}
}

// TestValidateHostPublicKey tests host public key validation.
func TestValidateHostPublicKey(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		if err := validateHostPublicKey(testHostPublicKey); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if err := validateHostPublicKey("not a public key"); err == nil {
			t.Fatal("expected non-nil error, got nil")
		}
	})

	t.Run("multiline", func(t *testing.T) {
		if err := validateHostPublicKey(testHostPublicKey + "\n# comment"); err == nil {
			t.Fatal("expected non-nil error, got nil")
		}
	})
}
