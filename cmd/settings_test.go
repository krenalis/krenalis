// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/dotenv"
)

func TestEnvLoading(t *testing.T) {

	// Load the environment variables form 'test-env-file.env'.
	err := dotenv.Load("testdata/test-env-file.env")
	if err != nil {
		t.Fatal(err)
	}

	// Determine the got key-values from the environment variables.
	got := map[string]any{}
	for _, env := range os.Environ() {
		key, value, ok := strings.Cut(env, "=")
		if !ok {
			t.Fatalf("unexpected: %q", env)
		}
		if strings.HasPrefix(key, "MEERGO_ENV_TEST_") {
			got[key] = value
		}
	}

	// Test the environment variables.
	expected := map[string]any{
		"MEERGO_ENV_TEST_A": "10",
		"MEERGO_ENV_TEST_B": "321",
		"MEERGO_ENV_TEST_C": "  hello  my   friend",
		"MEERGO_ENV_TEST_D": `"my-quoted-value"`,
		"MEERGO_ENV_TEST_E": "my-quoted-value",
		"MEERGO_ENV_TEST_F": "\"my-quoted-value",
		"MEERGO_ENV_TEST_G": "\"my-quoted-value\"",

		"MEERGO_ENV_TEST_H":     "3290",
		"MEERGO_ENV_TEST_I":     "hello\\ world",
		"MEERGO_ENV_TEST_EMPTY": "",
	}
	if !reflect.DeepEqual(got, expected) {
		for expectedK, expectedV := range expected {
			gotV, ok := got[expectedK]
			if !ok {
				t.Fatalf("env var %s expected but not read from env", expectedK)
			}
			if gotV != expectedV {
				t.Fatalf("invalid value for env var %s: expected %q, got %q", expectedK, expectedV, gotV)
			}
		}
		for gotK := range got {
			if _, ok := expected[gotK]; !ok {
				t.Fatalf("got var %s, but was not expected", gotK)
			}
		}
		t.Fatalf("expected and got env variables do not match")
	}
}

// TestParseSettings tests parseSettings across normal and edge cases.
func TestParseSettings(t *testing.T) {

	// Each subtest isolates environment using t.Setenv and validates both success
	// paths and expected failures.

	// helper to create a temporary file and return its path.
	createTempFile := func(t *testing.T, prefix string) string {
		t.Helper()
		f, err := os.CreateTemp(t.TempDir(), prefix)
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		_ = f.Close()
		return f.Name()
	}

	// helper to set a minimal valid baseline env that lets settingsFromEnv succeed.
	setBaseline := func(t *testing.T) {
		t.Helper()
		t.Setenv("MEERGO_DB_USERNAME", "u")
		t.Setenv("MEERGO_DB_PASSWORD", "p")
		t.Setenv("MEERGO_DB_DATABASE", "db")
	}

	t.Run("minimal baseline with defaults", func(t *testing.T) {
		setBaseline(t)

		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// General.
		if s.TerminationDelay != 0 {
			t.Errorf("expected default 0, got %s", s.TerminationDelay)
		}
		if sdkURL := "https://cdn.meergo.com/meergo.min.js"; s.JavaScriptSDKURL != sdkURL {
			t.Errorf("expected default %q, got %q", sdkURL, s.JavaScriptSDKURL)
		}
		if s.SentryTelemetryLevel != core.TelemetryLevelAll {
			t.Errorf("expected telemetry 'all', got %q", s.SentryTelemetryLevel)
		}

		// HTTP server.
		if s.HTTP.Host != "127.0.0.1" {
			t.Errorf("expected default HTTP host 127.0.0.1, got %q", s.HTTP.Host)
		}
		if s.HTTP.Port != 2022 {
			t.Errorf("expected default HTTP port 2022, got %d", s.HTTP.Port)
		}
		if s.HTTP.TLS.Enabled {
			t.Errorf("expected TLS disabled, got enabled")
		}
		if s.HTTP.TLS.CertFile != "" {
			t.Errorf("expected empty TLS cert file, got %q", s.HTTP.TLS.CertFile)
		}
		if s.HTTP.TLS.KeyFile != "" {
			t.Errorf("expected empty TLS key file, got %q", s.HTTP.TLS.KeyFile)
		}
		if s.HTTP.ExternalURL != "http://127.0.0.1:2022/" {
			t.Errorf("expected ExternalURL \"http://127.0.0.1:2022/\", got %q", s.HTTP.ExternalURL)
		}
		if s.HTTP.ExternalEventURL != "http://127.0.0.1:2022/api/v1/events" {
			t.Errorf("expected ExternalEventURL \"http://127.0.0.1:2022/api/v1/events\", got %q", s.HTTP.ExternalURL)
		}
		if s.HTTP.ReadHeaderTimeout != 2*time.Second {
			t.Errorf("expected default ReadHeaderTimeout 2s, got %s", s.HTTP.ReadHeaderTimeout)
		}
		if s.HTTP.ReadTimeout != 5*time.Second {
			t.Errorf("expected default ReadTimeout 5s, got %s", s.HTTP.ReadTimeout)
		}
		if s.HTTP.WriteTimeout != 30*time.Second {
			t.Errorf("expected default WriteTimeout 30s, got %s", s.HTTP.WriteTimeout)
		}
		if s.HTTP.IdleTimeout != 120*time.Second {
			t.Errorf("expected default IdleTimeout 120s, got %s", s.HTTP.IdleTimeout)
		}

		// Database.
		if s.DB.Port != 5432 {
			t.Errorf("expected default DB port 5432, got %d", s.DB.Port)
		}
		if s.DB.Schema != "public" {
			t.Errorf("expected default DB schema 'public', got %q", s.DB.Schema)
		}
		if s.DB.MaxConnections != 8 {
			t.Errorf("expected default DB max connections 8, got %d", s.DB.MaxConnections)
		}

		// Member emails.
		if !s.MemberEmailVerificationRequired {
			t.Error("expected MemberEmailVerificationRequired true, got false")
		}
		if s.MemberEmailFrom != "" {
			t.Errorf("expected MemberEmailFrom empty, got %q", s.MemberEmailFrom)
		}

		// SMTP.
		if s.SMTP.Host != "" {
			t.Errorf("expected SMTP Host empty, got %q", s.SMTP.Host)
		}
		if s.SMTP.Port != 0 {
			t.Errorf("expected SMTP Port not set, got %d", s.SMTP.Port)
		}
		if s.SMTP.Username != "" {
			t.Errorf("expected SMTP Username empty, got %q", s.SMTP.Username)
		}
		if s.SMTP.Password != "" {
			t.Errorf("expected SMTP Password empty, got %q", s.SMTP.Password)
		}

		// MaxMind.
		if s.MaxMindDBPath != "" {
			t.Errorf("expected s.MaxMindDBPath empty, got %q", s.MaxMindDBPath)
		}

	})

	t.Run("termination delay valid and invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_TERMINATION_DELAY", "150ms")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.TerminationDelay != 150*time.Millisecond {
			t.Errorf("expected 150ms, got %s", s.TerminationDelay)
		}

		// invalid.
		setBaseline(t)
		t.Setenv("MEERGO_TERMINATION_DELAY", "not-a-duration")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid duration, got nil")
		}
		want := "invalid duration value specified for MEERGO_TERMINATION_DELAY: time: invalid duration \"not-a-duration\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("JavaScript SDK URL invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_JAVASCRIPT_SDK_URL", "://bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid MEERGO_JAVASCRIPT_SDK_URL, got nil")
		}
		want := "MEERGO_JAVASCRIPT_SDK_URL must be a valid URL: invalid URL specified for MEERGO_JAVASCRIPT_SDK_URL: parse \"://bad\": missing protocol scheme"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("telemetry level matrix", func(t *testing.T) {
		cases := map[string]core.TelemetryLevel{
			"":       core.TelemetryLevelAll,
			"all":    core.TelemetryLevelAll,
			"none":   core.TelemetryLevelNone,
			"errors": core.TelemetryLevelErrors,
			"stats":  core.TelemetryLevelStats,
		}
		for in, want := range cases {
			t.Run("level="+in, func(t *testing.T) {
				setBaseline(t)
				if in != "" {
					t.Setenv("MEERGO_TELEMETRY_LEVEL", in)
				}
				s, err := parseEnvSettings()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if s.SentryTelemetryLevel != want {
					t.Errorf("expected %q, got %q", want, s.SentryTelemetryLevel)
				}
			})
		}

		setBaseline(t)
		t.Setenv("MEERGO_TELEMETRY_LEVEL", "verbose")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid telemetry level, got nil")
		}
		want := "invalid MEERGO_TELEMETRY_LEVEL: want one of none, errors, stats, or all"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP host and port validation", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_HOST", "exämple.com")
		t.Setenv("MEERGO_HTTP_PORT", "8080")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.HTTP.Host != "exämple.com" {
			t.Errorf("expected host exämple.com, got %q", s.HTTP.Host)
		}
		if s.HTTP.Port != 8080 {
			t.Errorf("expected port 8080, got %d", s.HTTP.Port)
		}

		setBaseline(t)
		t.Setenv("MEERGO_HTTP_HOST", "bad host")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid host, got nil")
		}
		want := "MEERGO_HTTP_HOST must be a valid host: host is not valid"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("MEERGO_HTTP_HOST", "127.0.0.1")
		t.Setenv("MEERGO_HTTP_PORT", "0")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid port, got nil")
		}
		want = "MEERGO_HTTP_PORT must be a valid port: port cannot be 0"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP port non numeric and overflow", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_PORT", "abc")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for non-numeric port, got nil")
		}
		want := "MEERGO_HTTP_PORT must be a valid port: port is not a positive integer"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_PORT", "70000")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for port >65535, got nil")
		}
		want = "MEERGO_HTTP_PORT must be a valid port: port must not exceed 65535"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS true requires cert and key", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "true")
		// Missing cert triggers error.
		t.Setenv("MEERGO_HTTP_TLS_CERT_FILE", "")
		t.Setenv("MEERGO_HTTP_TLS_KEY_FILE", createTempFile(t, "key-*.pem"))
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is true and cert is missing, got nil")
		}
		want := "MEERGO_HTTP_TLS_CERT_FILE must be set when MEERGO_HTTP_TLS_ENABLED is true"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "true")
		t.Setenv("MEERGO_HTTP_TLS_CERT_FILE", createTempFile(t, "cert-*.pem"))
		// Missing key triggers error.
		t.Setenv("MEERGO_HTTP_TLS_KEY_FILE", "")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is true and key is missing, got nil")
		}
		want = "MEERGO_HTTP_TLS_KEY_FILE must be set when MEERGO_HTTP_TLS_ENABLED is true"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS false with no cert/key is allowed", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "false")
		// No cert/key envs set.
		if _, err := parseEnvSettings(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("TLS false rejects cert file", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "false")
		t.Setenv("MEERGO_HTTP_TLS_CERT_FILE", "/some/path.pem")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is false and cert file is set, got nil")
		}
		want := "MEERGO_HTTP_TLS_CERT_FILE must not be set when MEERGO_HTTP_TLS_ENABLED is false"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS false rejects key file", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "false")
		t.Setenv("MEERGO_HTTP_TLS_KEY_FILE", "/some/key.pem")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is false and key file is set, got nil")
		}
		want := "MEERGO_HTTP_TLS_KEY_FILE must not be set when MEERGO_HTTP_TLS_ENABLED is false"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS file paths must exist", func(t *testing.T) {
		setBaseline(t)
		nonexistentFile := "/no/such/cert.pem"
		if runtime.GOOS == "windows" {
			nonexistentFile = `C:\no\such\cert.pem`
		}
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "true")
		t.Setenv("MEERGO_HTTP_TLS_CERT_FILE", nonexistentFile)
		t.Setenv("MEERGO_HTTP_TLS_KEY_FILE", createTempFile(t, "key-*.pem"))
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for missing cert file, got nil")
		}
		want := fmt.Sprintf("MEERGO_HTTP_TLS_CERT_FILE points to a non-existent file: %q", nonexistentFile)
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		nonexistentFile = "/no/such/key.pem"
		if runtime.GOOS == "windows" {
			nonexistentFile = `C:\no\such\key.pem`
		}
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "true")
		t.Setenv("MEERGO_HTTP_TLS_CERT_FILE", createTempFile(t, "cert-*.pem"))
		t.Setenv("MEERGO_HTTP_TLS_KEY_FILE", nonexistentFile)
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for missing key file, got nil")
		}
		want = fmt.Sprintf("MEERGO_HTTP_TLS_KEY_FILE points to a non-existent file: %q", nonexistentFile)
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS enabled invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "maybe")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid TLS boolean, got nil")
		}
		want := "MEERGO_HTTP_TLS_ENABLED must be a boolean: value \"maybe\" is not a valid boolean value (expected true, false or empty string)"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("external URL path or query rejected", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_EXTERNAL_URL", "https://example.com/path")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for external URL with path, got nil")
		}
		want := "invalid URL specified for MEERGO_HTTP_EXTERNAL_URL: path must be \"/\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_EXTERNAL_URL", "https://example.com/?q=1")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for external URL with query, got nil")
		}
		want = "invalid URL specified for MEERGO_HTTP_EXTERNAL_URL: query cannot be specified"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("ExternalURL override and event URL override", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_EXTERNAL_URL", "https://example.com/")
		t.Setenv("MEERGO_HTTP_EXTERNAL_EVENT_URL", "https://example.com/events")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.HTTP.ExternalURL != "https://example.com/" {
			t.Errorf("expected \"https://example.com/\", got %q", s.HTTP.ExternalURL)
		}
		if s.HTTP.ExternalEventURL != "https://example.com/events" {
			t.Errorf("expected \"https://example.com/events\", got %q", s.HTTP.ExternalEventURL)
		}
	})

	t.Run("external URL omits default https port", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_TLS_ENABLED", "true")
		t.Setenv("MEERGO_HTTP_TLS_CERT_FILE", createTempFile(t, "cert-*.pem"))
		t.Setenv("MEERGO_HTTP_TLS_KEY_FILE", createTempFile(t, "key-*.pem"))
		t.Setenv("MEERGO_HTTP_PORT", "443")

		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.HTTP.ExternalURL != "https://127.0.0.1/" {
			t.Errorf("expected https://127.0.0.1/, got %q", s.HTTP.ExternalURL)
		}
	})

	t.Run("external event URL query rejected", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_EXTERNAL_EVENT_URL", "https://example.com/events?q=1")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for event URL with query, got nil")
		}
		want := "invalid URL specified for MEERGO_HTTP_EXTERNAL_EVENT_URL: query cannot be specified"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP timeouts parsing invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_READ_TIMEOUT", "bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid read timeout, got nil")
		}
		want := "invalid value specified for MEERGO_HTTP_READ_TIMEOUT: time: invalid duration \"bad\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP write timeout invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_WRITE_TIMEOUT", "bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid write timeout, got nil")
		}
		want := "invalid value specified for MEERGO_HTTP_WRITE_TIMEOUT: time: invalid duration \"bad\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP idle timeout invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_HTTP_IDLE_TIMEOUT", "bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid idle timeout, got nil")
		}
		want := "invalid value specified for MEERGO_HTTP_IDLE_TIMEOUT: time: invalid duration \"bad\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("db required and validations", func(t *testing.T) {
		// Missing host.
		setBaseline(t)
		t.Setenv("MEERGO_DB_HOST", "")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.DB.Host != "127.0.0.1" {
			t.Errorf("expected host 127.0.0.1, got %q", s.DB.Host)
		}

		// Missing username.
		setBaseline(t)
		// Remove the variable since it's part of the baseline.
		err = os.Unsetenv("MEERGO_DB_USERNAME")
		if err != nil {
			t.Fatalf("expected error for unsetting MEERGO_DB_USERNAME")
		}
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length, got nil")
		}
		want := "environment variable MEERGO_DB_USERNAME is missing"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Empty username.
		setBaseline(t)
		t.Setenv("MEERGO_DB_USERNAME", "")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length, got nil")
		}
		want = "MEERGO_DB_USERNAME cannot be empty"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Username length.
		setBaseline(t)
		t.Setenv("MEERGO_DB_USERNAME", strings.Repeat("x", 64))
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length, got nil")
		}
		want = "invalid MEERGO_DB_USERNAME: length must be 1..63 bytes"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Password length.
		setBaseline(t)
		t.Setenv("MEERGO_DB_PASSWORD", "")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for password length, got nil")
		}
		want = "invalid MEERGO_DB_PASSWORD: length must be 1..100 characters"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Database length.
		setBaseline(t)
		t.Setenv("MEERGO_DB_DATABASE", "")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for database length, got nil")
		}
		want = "invalid MEERGO_DB_DATABASE: length must be 1..63 bytes"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Schema default and override.
		setBaseline(t)
		t.Setenv("MEERGO_DB_SCHEMA", "custom")
		s, err = parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.DB.Schema != "custom" {
			t.Errorf("expected schema custom, got %q", s.DB.Schema)
		}
	})

	t.Run("db host invalid and boundary values", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_DB_HOST", "bad host")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid DB host, got nil")
		}
		want := "MEERGO_DB_HOST must be a valid host: host is not valid"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
		setBaseline(t)
		t.Setenv("MEERGO_DB_HOST", "127.0.0.1")
		t.Setenv("MEERGO_DB_PORT", "65535")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.DB.Port != 65535 {
			t.Errorf("expected 65535, got %d", s.DB.Port)
		}
		tooLong := strings.Repeat("a", 64)
		setBaseline(t)
		t.Setenv("MEERGO_DB_USERNAME", tooLong)
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length >63, got nil")
		}
		setBaseline(t)
		t.Setenv("MEERGO_DB_DATABASE", tooLong)
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for database length >63, got nil")
		}
		setBaseline(t)
		t.Setenv("MEERGO_DB_PASSWORD", strings.Repeat("x", 101))
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for password length >100, got nil")
		}
		setBaseline(t)
		t.Setenv("MEERGO_DB_MAX_CONNECTIONS", "2")
		s, err = parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.DB.MaxConnections != 2 {
			t.Errorf("expected 2, got %d", s.DB.MaxConnections)
		}
	})

	t.Run("db max connections parsing and bounds", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_DB_MAX_CONNECTIONS", "notint")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for non-integer max connections, got nil")
		}
		want := "MEERGO_DB_MAX_CONNECTIONS must be an integer"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("MEERGO_DB_MAX_CONNECTIONS", "1")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for max connections < 2, got nil")
		}
		want = "MEERGO_DB_MAX_CONNECTIONS must be >= 2, got 1"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("MEERGO_DB_MAX_CONNECTIONS", "-7")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for max connections < 2, got nil")
		}
		want = "MEERGO_DB_MAX_CONNECTIONS must be >= 2, got -7"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("MEERGO_DB_MAX_CONNECTIONS", "64")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.DB.MaxConnections != 64 {
			t.Errorf("expected 64, got %d", s.DB.MaxConnections)
		}
	})

	t.Run("boolean flags parsing", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_MEMBER_EMAIL_VERIFICATION_REQUIRED", "false")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.MemberEmailVerificationRequired {
			t.Errorf("expected false, got true")
		}

		setBaseline(t)
		t.Setenv("MEERGO_MEMBER_EMAIL_VERIFICATION_REQUIRED", "true")
		s, err = parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !s.MemberEmailVerificationRequired {
			t.Errorf("expected true, got false")
		}

		setBaseline(t)
		t.Setenv("MEERGO_MEMBER_EMAIL_VERIFICATION_REQUIRED", "not-bool")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid boolean, got nil")
		}
		want := "MEERGO_MEMBER_EMAIL_VERIFICATION_REQUIRED must be a boolean: value \"not-bool\" is not a valid boolean value (expected true, false or empty string)"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("SMTP conditional block", func(t *testing.T) {
		// Host set but missing port.
		setBaseline(t)
		t.Setenv("MEERGO_SMTP_HOST", "smtp.example.com")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when SMTP host set without port, got nil")
		}
		want := "MEERGO_SMTP_PORT is required if MEERGO_SMTP_HOST is set"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Invalid port yields specific error text from code.
		setBaseline(t)
		t.Setenv("MEERGO_SMTP_HOST", "smtp.example.com")
		t.Setenv("MEERGO_SMTP_PORT", "0")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid SMTP port, got nil")
		}
		want = "MEERGO_SMTP_PORT must be a valid port: port cannot be 0"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Valid SMTP.
		setBaseline(t)
		t.Setenv("MEERGO_SMTP_HOST", "smtp.example.com")
		t.Setenv("MEERGO_SMTP_PORT", "587")
		t.Setenv("MEERGO_SMTP_USERNAME", "user")
		t.Setenv("MEERGO_SMTP_PASSWORD", "pass")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.SMTP.Host != "smtp.example.com" || s.SMTP.Port != 587 {
			t.Errorf("expected SMTP host smtp.example.com and port 587, got %s:%d", s.SMTP.Host, s.SMTP.Port)
		}
		if s.SMTP.Username != "user" || s.SMTP.Password != "pass" {
			t.Errorf("expected SMTP username \"user\" and password \"pass\", got %q and %q", s.SMTP.Username, s.SMTP.Password)
		}

	})

	t.Run("smtp host invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_SMTP_HOST", "bad host")
		t.Setenv("MEERGO_SMTP_PORT", "25")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid SMTP host, got nil")
		}
		want := "MEERGO_SMTP_HOST must be a valid host: host is not valid"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("MaxMind DB path set when file exists", func(t *testing.T) {
		setBaseline(t)
		path := createTempFile(t, "GeoIP2-*.mmdb")
		t.Setenv("MEERGO_MAXMIND_DB_PATH", path)
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if clean := filepath.Clean(s.MaxMindDBPath); s.MaxMindDBPath != clean {
			t.Errorf("expected clean path %q, got %q", clean, s.MaxMindDBPath)
		}
		if s.MaxMindDBPath != filepath.Clean(path) {
			t.Errorf("expected %q, got %q", path, s.MaxMindDBPath)
		}
	})

	t.Run("maxmind path missing", func(t *testing.T) {
		nonexistentFile := "/no/such.mmdb"
		if runtime.GOOS == "windows" {
			nonexistentFile = `C:\no\such.mmdb`
		}
		setBaseline(t)
		t.Setenv("MEERGO_MAXMIND_DB_PATH", nonexistentFile)
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for missing MaxMind db file, got nil")
		}
		want := fmt.Sprintf("MEERGO_MAXMIND_DB_PATH points to a non-existent file: %q", nonexistentFile)
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("transformers conflict between local and Lambda", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_TRANSFORMERS_LOCAL_NODE_EXECUTABLE", "/usr/bin/node")
		t.Setenv("MEERGO_TRANSFORMERS_AWS_LAMBDA_NODE_RUNTIME", "nodejs18.x")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for conflicting transformers, got nil")
		}
		want := "invalid configuration: cannot set both Lambda and local transformers"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("transformers local-only accepted", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_TRANSFORMERS_LOCAL_NODE_EXECUTABLE", "/usr/bin/node")
		if _, err := parseEnvSettings(); err != nil {
			t.Fatalf("expected no error for local-only transformers, got %v", err)
		}
	})

	t.Run("transformers Lambda-only accepted", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("MEERGO_TRANSFORMERS_AWS_LAMBDA_NODE_RUNTIME", "nodejs18.x")
		if _, err := parseEnvSettings(); err != nil {
			t.Fatalf("expected no error for Lambda-only transformers, got %v", err)
		}
	})

	t.Run("OAuth HubSpot and Mailchimp combinations", func(t *testing.T) {
		// HubSpot ID without secret -> error.
		setBaseline(t)
		t.Setenv("MEERGO_OAUTH_HUBSPOT_CLIENT_ID", "id")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for HubSpot ID without secret, got nil")
		}
		want := "MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET is required when MEERGO_OAUTH_HUBSPOT_CLIENT_ID is set"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// HubSpot valid, Mailchimp missing secret -> error.
		setBaseline(t)
		t.Setenv("MEERGO_OAUTH_HUBSPOT_CLIENT_ID", "id")
		t.Setenv("MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET", "sec")
		t.Setenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_ID", "mcid")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for Mailchimp ID without secret, got nil")
		}
		want = "MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET is required when MEERGO_OAUTH_MAILCHIMP_CLIENT_ID is set"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Mailchimp secret without id -> error. Ensure ID is unset.
		setBaseline(t)
		t.Setenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_ID", "")
		t.Setenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET", "sec")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for Mailchimp secret without id, got nil")
		}
		want = "MEERGO_OAUTH_MAILCHIMP_CLIENT_ID is required when MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET is set"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Both valid.
		setBaseline(t)
		t.Setenv("MEERGO_OAUTH_HUBSPOT_CLIENT_ID", "id")
		t.Setenv("MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET", "sec")
		t.Setenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_ID", "mcid")
		t.Setenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET", "msec")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.OAuthCredentials == nil || s.OAuthCredentials["hubspot"] == nil || s.OAuthCredentials["mailchimp"] == nil {
			t.Fatalf("expected both OAuth connectors present, got %#v", s.OAuthCredentials)
		}
	})
}

// TestParseEnvURLSuccess verifies valid inputs and normalization behaviors.
func TestParseEnvURLSuccess(t *testing.T) {

	t.Run("empty env returns empty and nil", func(t *testing.T) {
		t.Setenv("MEERGO_PARSE_ENV_URL_EMPTY", "")
		got, err := parseEnvURL("MEERGO_PARSE_ENV_URL_EMPTY", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("http without path gets normalized to slash", func(t *testing.T) {
		t.Setenv("MEERGO_URL_HTTP_NOPATH", "http://example.com")
		got, err := parseEnvURL("MEERGO_URL_HTTP_NOPATH", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "http://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("https with slash path stays as is", func(t *testing.T) {
		t.Setenv("MEERGO_URL_HTTPS_SLASH", "https://example.com/")
		got, err := parseEnvURL("MEERGO_URL_HTTPS_SLASH", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("strip leading zeros in port then default port removal", func(t *testing.T) {
		t.Setenv("MEERGO_URL_HTTP_LEADING_ZERO_PORT", "http://example.com:00080")
		got, err := parseEnvURL("MEERGO_URL_HTTP_LEADING_ZERO_PORT", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		// Port 00080 -> 80, and 80 is default for http, so it is removed.
		want := "http://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("keep non-default valid port", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NONDEFAULT_PORT", "https://example.com:8443/")
		got, err := parseEnvURL("MEERGO_URL_NONDEFAULT_PORT", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com:8443/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("remove default port 443 for https", func(t *testing.T) {
		t.Setenv("MEERGO_URL_HTTPS_DEFAULT_PORT", "https://example.com:443")
		got, err := parseEnvURL("MEERGO_URL_HTTPS_DEFAULT_PORT", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("ipv6 literal with port is accepted", func(t *testing.T) {
		t.Setenv("MEERGO_URL_IPV6_PORT", "http://[::1]:8080")
		got, err := parseEnvURL("MEERGO_URL_IPV6_PORT", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "http://[::1]:8080/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})
}

// TestParseEnvURLFlags verifies behavior controlled by validation flags.
func TestParseEnvURLFlags(t *testing.T) {

	t.Run("noPath rejects non-root path", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NOPATH_FLAG_BAD", "https://example.com/foo")
		_, err := parseEnvURL("MEERGO_URL_NOPATH_FLAG_BAD", noPath)
		wantErr := `invalid URL specified for MEERGO_URL_NOPATH_FLAG_BAD: path must be "/"`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("noPath allows empty path which normalizes to slash", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NOPATH_FLAG_OK", "https://example.com")
		got, err := parseEnvURL("MEERGO_URL_NOPATH_FLAG_OK", noPath)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("noQuery rejects non-empty query", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NOQUERY_BAD", "https://example.com/?a=b")
		_, err := parseEnvURL("MEERGO_URL_NOQUERY_BAD", noQuery)
		wantErr := "invalid URL specified for MEERGO_URL_NOQUERY_BAD: query cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("noQuery rejects trailing question mark (ForceQuery)", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NOQUERY_FORCE", "https://example.com/?")
		_, err := parseEnvURL("MEERGO_URL_NOQUERY_FORCE", noQuery)
		wantErr := "invalid URL specified for MEERGO_URL_NOQUERY_FORCE: query cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})
}

// TestParseURLErrors verifies that each distinct error path returns the expected message.
func TestParseURLErrors(t *testing.T) {

	t.Run("leading space", func(t *testing.T) {
		t.Setenv("MEERGO_URL_SPACE_START", " https://example.com/")
		_, err := parseEnvURL("MEERGO_URL_SPACE_START", 0)
		wantErr := "invalid URL specified for MEERGO_URL_SPACE_START: it starts with a space"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("trailing space", func(t *testing.T) {
		t.Setenv("MEERGO_URL_SPACE_END", "https://example.com/ ")
		_, err := parseEnvURL("MEERGO_URL_SPACE_END", 0)
		wantErr := "invalid URL specified for MEERGO_URL_SPACE_END: it ends with a space"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("url.Parse failure", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PARSE_FAIL", "http://[::1")
		_, err := parseEnvURL("MEERGO_URL_PARSE_FAIL", 0)
		// We match the exact message format propagated by url.Parse into errInvalidURL.
		wantErr := "invalid URL specified for MEERGO_URL_PARSE_FAIL: parse \"http://[::1\": missing ']' in host"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("unsupported scheme", func(t *testing.T) {
		t.Setenv("MEERGO_URL_BAD_SCHEME", "ftp://example.com/")
		_, err := parseEnvURL("MEERGO_URL_BAD_SCHEME", 0)
		wantErr := `invalid URL specified for MEERGO_URL_BAD_SCHEME: scheme must be "http" or "https"`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("user info present", func(t *testing.T) {
		t.Setenv("MEERGO_URL_USERINFO", "http://user:pass@example.com/")
		_, err := parseEnvURL("MEERGO_URL_USERINFO", 0)
		wantErr := "invalid URL specified for MEERGO_URL_USERINFO: user and password cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("missing host", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NO_HOST", "http://")
		_, err := parseEnvURL("MEERGO_URL_NO_HOST", 0)
		wantErr := "invalid URL specified for MEERGO_URL_NO_HOST: host must be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: zero", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PORT_ZERO", "https://example.com:0")
		_, err := parseEnvURL("MEERGO_URL_PORT_ZERO", 0)
		wantErr := "invalid URL specified for MEERGO_URL_PORT_ZERO: port cannot be 0"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: above max", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PORT_BIG", "https://example.com:65536")
		_, err := parseEnvURL("MEERGO_URL_PORT_BIG", 0)
		wantErr := "invalid URL specified for MEERGO_URL_PORT_BIG: port must not exceed 65535"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: non numeric", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PORT_NAN", "https://example.com:abc")
		_, err := parseEnvURL("MEERGO_URL_PORT_NAN", 0)
		wantErr := `invalid URL specified for MEERGO_URL_PORT_NAN: parse "https://example.com:abc": invalid port ":abc" after host`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("fragment present", func(t *testing.T) {
		t.Setenv("MEERGO_URL_FRAGMENT", "https://example.com/#frag")
		_, err := parseEnvURL("MEERGO_URL_FRAGMENT", 0)
		wantErr := "invalid URL specified for MEERGO_URL_FRAGMENT: fragment cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})
}
