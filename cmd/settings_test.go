// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/core/natsopts"
	"github.com/krenalis/krenalis/tools/dotenv"
	"github.com/krenalis/krenalis/tools/validation"

	"github.com/nats-io/nkeys"
)

func TestEnvLoading(t *testing.T) {

	// Load the environment variables from 'test-env-file.env'.
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
		if strings.HasPrefix(key, "KRENALIS_ENV_TEST_") {
			got[key] = value
		}
	}

	// Test the environment variables.
	expected := map[string]any{
		"KRENALIS_ENV_TEST_A": "10",
		"KRENALIS_ENV_TEST_B": "321",
		"KRENALIS_ENV_TEST_C": "  hello  my   friend",
		"KRENALIS_ENV_TEST_D": `"my-quoted-value"`,
		"KRENALIS_ENV_TEST_E": "my-quoted-value",
		"KRENALIS_ENV_TEST_F": "\"my-quoted-value",
		"KRENALIS_ENV_TEST_G": "\"my-quoted-value\"",

		"KRENALIS_ENV_TEST_H":     "3290",
		"KRENALIS_ENV_TEST_I":     "hello\\ world",
		"KRENALIS_ENV_TEST_EMPTY": "",
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
			t.Fatalf("cannot create temp file: %v", err)
		}
		_ = f.Close()
		return f.Name()
	}

	// helper to set a minimal valid baseline env that lets settingsFromEnv succeed.
	setBaseline := func(t *testing.T) {
		t.Helper()
		t.Setenv("KRENALIS_DB_USERNAME", "u")
		t.Setenv("KRENALIS_DB_PASSWORD", "p")
		t.Setenv("KRENALIS_DB_DATABASE", "db")
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
		if sdkURL := "https://cdn.krenalis.com/krenalis.min.js"; s.JavaScriptSDKURL != sdkURL {
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
		if s.HTTP.ExternalEventURL != "http://127.0.0.1:2022/v1/events" {
			t.Errorf("expected ExternalEventURL \"http://127.0.0.1:2022/v1/events\", got %q", s.HTTP.ExternalURL)
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
		if s.InviteMembersViaEmail {
			t.Error("expected InviteMembersViaEmail false, got true")
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

		// Prometheus metrics enabled.
		if s.PrometheusMetricsEnabled {
			t.Error("expected PrometheusMetricsEnabled false, got true")
		}

		// Max queued events per destination.
		if s.MaxQueuedEventsPerDestination != 50000 {
			t.Errorf("expected default MaxQueuedEventsPerDestination 50000, got %d", s.MaxQueuedEventsPerDestination)
		}
	})

	t.Run("termination delay valid and invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_TERMINATION_DELAY", "150ms")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.TerminationDelay != 150*time.Millisecond {
			t.Errorf("expected 150ms, got %s", s.TerminationDelay)
		}

		// invalid.
		setBaseline(t)
		t.Setenv("KRENALIS_TERMINATION_DELAY", "not-a-duration")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid duration, got nil")
		}
		want := "invalid duration value specified for KRENALIS_TERMINATION_DELAY: time: invalid duration \"not-a-duration\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("JavaScript SDK URL invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_JAVASCRIPT_SDK_URL", "://bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid KRENALIS_JAVASCRIPT_SDK_URL, got nil")
		}
		want := "KRENALIS_JAVASCRIPT_SDK_URL must be a valid URL: invalid URL specified for KRENALIS_JAVASCRIPT_SDK_URL: parse \"://bad\": missing protocol scheme"
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
					t.Setenv("KRENALIS_TELEMETRY_LEVEL", in)
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
		t.Setenv("KRENALIS_TELEMETRY_LEVEL", "verbose")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid telemetry level, got nil")
		}
		want := "invalid KRENALIS_TELEMETRY_LEVEL: want one of none, errors, stats, or all"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP host and port validation", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_HOST", "exämple.com")
		t.Setenv("KRENALIS_HTTP_PORT", "8080")
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
		t.Setenv("KRENALIS_HTTP_HOST", "bad host")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid host, got nil")
		}
		want := "KRENALIS_HTTP_HOST must be a valid host: host is not valid"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_HOST", "127.0.0.1")
		t.Setenv("KRENALIS_HTTP_PORT", "0")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid port, got nil")
		}
		want = "KRENALIS_HTTP_PORT must be a valid port: port cannot be 0"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP port non numeric and overflow", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_PORT", "abc")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for non-numeric port, got nil")
		}
		want := "KRENALIS_HTTP_PORT must be a valid port: port is not a positive integer"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_PORT", "70000")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for port >65535, got nil")
		}
		want = "KRENALIS_HTTP_PORT must be a valid port: port must not exceed 65535"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS true requires cert and key", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "true")
		// Missing cert triggers error.
		t.Setenv("KRENALIS_HTTP_TLS_CERT_FILE", "")
		t.Setenv("KRENALIS_HTTP_TLS_KEY_FILE", createTempFile(t, "key-*.pem"))
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is true and cert is missing, got nil")
		}
		want := "KRENALIS_HTTP_TLS_CERT_FILE must be set when KRENALIS_HTTP_TLS_ENABLED is true"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "true")
		t.Setenv("KRENALIS_HTTP_TLS_CERT_FILE", createTempFile(t, "cert-*.pem"))
		// Missing key triggers error.
		t.Setenv("KRENALIS_HTTP_TLS_KEY_FILE", "")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is true and key is missing, got nil")
		}
		want = "KRENALIS_HTTP_TLS_KEY_FILE must be set when KRENALIS_HTTP_TLS_ENABLED is true"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS false with no cert/key is allowed", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "false")
		// No cert/key envs set.
		if _, err := parseEnvSettings(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("TLS false rejects cert file", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "false")
		t.Setenv("KRENALIS_HTTP_TLS_CERT_FILE", "/some/path.pem")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is false and cert file is set, got nil")
		}
		want := "KRENALIS_HTTP_TLS_CERT_FILE must not be set when KRENALIS_HTTP_TLS_ENABLED is false"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS false rejects key file", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "false")
		t.Setenv("KRENALIS_HTTP_TLS_KEY_FILE", "/some/key.pem")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when TLS is false and key file is set, got nil")
		}
		want := "KRENALIS_HTTP_TLS_KEY_FILE must not be set when KRENALIS_HTTP_TLS_ENABLED is false"
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
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "true")
		t.Setenv("KRENALIS_HTTP_TLS_CERT_FILE", nonexistentFile)
		t.Setenv("KRENALIS_HTTP_TLS_KEY_FILE", createTempFile(t, "key-*.pem"))
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for missing cert file, got nil")
		}
		want := fmt.Sprintf("KRENALIS_HTTP_TLS_CERT_FILE points to a non-existent file: %q", nonexistentFile)
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		nonexistentFile = "/no/such/key.pem"
		if runtime.GOOS == "windows" {
			nonexistentFile = `C:\no\such\key.pem`
		}
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "true")
		t.Setenv("KRENALIS_HTTP_TLS_CERT_FILE", createTempFile(t, "cert-*.pem"))
		t.Setenv("KRENALIS_HTTP_TLS_KEY_FILE", nonexistentFile)
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for missing key file, got nil")
		}
		want = fmt.Sprintf("KRENALIS_HTTP_TLS_KEY_FILE points to a non-existent file: %q", nonexistentFile)
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("TLS enabled invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "maybe")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid TLS boolean, got nil")
		}
		want := "KRENALIS_HTTP_TLS_ENABLED must be a boolean: value \"maybe\" is not a valid boolean value (expected true, false or empty string)"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("external URL path or query rejected", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_EXTERNAL_URL", "https://example.com/path")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for external URL with path, got nil")
		}
		want := "invalid URL specified for KRENALIS_HTTP_EXTERNAL_URL: path must be \"/\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_EXTERNAL_URL", "https://example.com/?q=1")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for external URL with query, got nil")
		}
		want = "invalid URL specified for KRENALIS_HTTP_EXTERNAL_URL: query cannot be specified"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("ExternalURL override and event URL override", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_EXTERNAL_URL", "https://example.com/")
		t.Setenv("KRENALIS_HTTP_EXTERNAL_EVENT_URL", "https://example.com/events")
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
		t.Setenv("KRENALIS_HTTP_TLS_ENABLED", "true")
		t.Setenv("KRENALIS_HTTP_TLS_CERT_FILE", createTempFile(t, "cert-*.pem"))
		t.Setenv("KRENALIS_HTTP_TLS_KEY_FILE", createTempFile(t, "key-*.pem"))
		t.Setenv("KRENALIS_HTTP_PORT", "443")

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
		t.Setenv("KRENALIS_HTTP_EXTERNAL_EVENT_URL", "https://example.com/events?q=1")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for event URL with query, got nil")
		}
		want := "invalid URL specified for KRENALIS_HTTP_EXTERNAL_EVENT_URL: query cannot be specified"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP timeouts parsing invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_READ_TIMEOUT", "bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid read timeout, got nil")
		}
		want := "invalid value specified for KRENALIS_HTTP_READ_TIMEOUT: time: invalid duration \"bad\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP write timeout invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_WRITE_TIMEOUT", "bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid write timeout, got nil")
		}
		want := "invalid value specified for KRENALIS_HTTP_WRITE_TIMEOUT: time: invalid duration \"bad\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("HTTP idle timeout invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_HTTP_IDLE_TIMEOUT", "bad")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid idle timeout, got nil")
		}
		want := "invalid value specified for KRENALIS_HTTP_IDLE_TIMEOUT: time: invalid duration \"bad\""
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("db required and validations", func(t *testing.T) {
		// Missing host.
		setBaseline(t)
		t.Setenv("KRENALIS_DB_HOST", "")
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
		err = os.Unsetenv("KRENALIS_DB_USERNAME")
		if err != nil {
			t.Fatalf("expected error for unsetting KRENALIS_DB_USERNAME")
		}
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length, got nil")
		}
		want := "environment variable KRENALIS_DB_USERNAME is missing"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Empty username.
		setBaseline(t)
		t.Setenv("KRENALIS_DB_USERNAME", "")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length, got nil")
		}
		want = "KRENALIS_DB_USERNAME cannot be empty"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Username length.
		setBaseline(t)
		t.Setenv("KRENALIS_DB_USERNAME", strings.Repeat("x", 64))
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length, got nil")
		}
		want = "invalid KRENALIS_DB_USERNAME: length must be 1..63 bytes"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Empty password.
		setBaseline(t)
		t.Setenv("KRENALIS_DB_PASSWORD", "")
		_, err = parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Database length.
		setBaseline(t)
		t.Setenv("KRENALIS_DB_DATABASE", "")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for database length, got nil")
		}
		want = "invalid KRENALIS_DB_DATABASE: length must be 1..63 bytes"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Schema default and override.
		setBaseline(t)
		t.Setenv("KRENALIS_DB_SCHEMA", "custom")
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
		t.Setenv("KRENALIS_DB_HOST", "bad host")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid DB host, got nil")
		}
		want := "KRENALIS_DB_HOST must be a valid host: host is not valid"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
		setBaseline(t)
		t.Setenv("KRENALIS_DB_HOST", "127.0.0.1")
		t.Setenv("KRENALIS_DB_PORT", "65535")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.DB.Port != 65535 {
			t.Errorf("expected 65535, got %d", s.DB.Port)
		}
		tooLong := strings.Repeat("a", 64)
		setBaseline(t)
		t.Setenv("KRENALIS_DB_USERNAME", tooLong)
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for username length >63, got nil")
		}
		setBaseline(t)
		t.Setenv("KRENALIS_DB_DATABASE", tooLong)
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for database length >63, got nil")
		}
		setBaseline(t)
		t.Setenv("KRENALIS_DB_PASSWORD", strings.Repeat("x", 101))
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for password length >100, got nil")
		}
		setBaseline(t)
		t.Setenv("KRENALIS_DB_MAX_CONNECTIONS", "2")
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
		t.Setenv("KRENALIS_DB_MAX_CONNECTIONS", "notint")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for non-integer max connections, got nil")
		}
		want := "KRENALIS_DB_MAX_CONNECTIONS must be an integer"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("KRENALIS_DB_MAX_CONNECTIONS", "1")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for max connections < 2, got nil")
		}
		want = "KRENALIS_DB_MAX_CONNECTIONS must be >= 2, got 1"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("KRENALIS_DB_MAX_CONNECTIONS", "-7")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for max connections < 2, got nil")
		}
		want = "KRENALIS_DB_MAX_CONNECTIONS must be >= 2, got -7"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("KRENALIS_DB_MAX_CONNECTIONS", "64")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.DB.MaxConnections != 64 {
			t.Errorf("expected 64, got %d", s.DB.MaxConnections)
		}
	})

	t.Run("max queued events per destination parsing and bounds", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION", "notint")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for non-integer max queued events per destination, got nil")
		}
		want := "KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION must be an integer"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
		setBaseline(t)
		t.Setenv("KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION", "0")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for max queued events per destination < 1, got nil")
		}
		want = "KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION must be >= 1, got 0"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		setBaseline(t)
		t.Setenv("KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION", "60000")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.MaxQueuedEventsPerDestination != 60000 {
			t.Errorf("expected 60000, got %d", s.MaxQueuedEventsPerDestination)
		}
	})

	t.Run("NATS URL parsing", func(t *testing.T) {
		cases := []struct {
			name    string
			env     string
			want    []string
			wantErr string
		}{
			{
				name: "default when unset",
				env:  "",
				want: []string{"nats://127.0.0.1:4222"},
			},
			{
				name: "single NATS URL",
				env:  "nats://nats.example.com:4222",
				want: []string{"nats://nats.example.com:4222"},
			},
			{
				name: "scheme-less URL accepted",
				env:  "n1:4222",
				want: []string{"n1:4222"},
			},
			{
				name: "multiple non-websocket URLs",
				env:  "nats://n1:4222, n2:4223",
				want: []string{"nats://n1:4222", "n2:4223"},
			},
			{
				name:    "invalid scheme rejected",
				env:     "http://nats.example.com:4222",
				wantErr: "KRENALIS_NATS_URL scheme http is not allowed. Allowed schemes are nats, tls, ws, and wss",
			},
			{
				name:    "invalid URL rejected",
				env:     "nats://[::1",
				wantErr: "KRENALIS_NATS_URL contains an invalid URL: \"nats://[::1\"",
			},
			{
				name:    "mixed websocket and non-websocket rejected",
				env:     "ws://n2:443, n1:4222",
				wantErr: "KRENALIS_NATS_URL contains both websocket and non-websocket URLs",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				setBaseline(t)
				t.Setenv("KRENALIS_NATS_URL", tc.env)
				s, err := parseEnvSettings()
				if tc.wantErr != "" {
					if err == nil {
						t.Fatalf("expected error, got nil")
					}
					if err.Error() != tc.wantErr {
						t.Fatalf("expected %q, got %q", tc.wantErr, err.Error())
					}
					return
				}
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !slices.Equal(s.NATS.Servers, tc.want) {
					t.Fatalf("expected %#v, got %#v", tc.want, s.NATS.Servers)
				}
			})
		}
	})

	t.Run("NATS auth parsing", func(t *testing.T) {
		userKP, err := nkeys.CreateUser()
		if err != nil {
			t.Fatalf("cannot create user NKey: %v", err)
		}
		userSeed, err := userKP.Seed()
		if err != nil {
			t.Fatalf("cannot get user seed: %v", err)
		}
		_, userSeedBytes, err := nkeys.DecodeSeed(userSeed)
		if err != nil {
			t.Fatalf("cannot decode user seed: %v", err)
		}
		accountKP, err := nkeys.CreateAccount()
		if err != nil {
			t.Fatalf("cannot create account NKey: %v", err)
		}
		accountSeed, err := accountKP.Seed()
		if err != nil {
			t.Fatalf("cannot get account seed: %v", err)
		}

		cases := []struct {
			name      string
			envUser   string
			envPass   string
			envToken  string
			envNKey   string
			wantUser  string
			wantPass  string
			wantToken string
			wantNKey  ed25519.PrivateKey
			wantErr   string
		}{
			{
				name:     "user and password set",
				envUser:  "nats-user",
				envPass:  "nats-pass",
				wantUser: "nats-user",
				wantPass: "nats-pass",
			},
			{
				name: "no auth set",
			},
			{
				name:      "multiple auth values set",
				envUser:   "nats-user",
				envPass:   "nats-pass",
				envToken:  "nats-token",
				envNKey:   string(userSeed),
				wantUser:  "nats-user",
				wantPass:  "nats-pass",
				wantToken: "nats-token",
				wantNKey:  ed25519.NewKeyFromSeed(userSeedBytes),
			},
			{
				name:    "password without user rejected",
				envPass: "nats-pass",
				wantErr: "KRENALIS_NATS_USER must be set if KRENALIS_NATS_PASSWORD is provided",
			},
			{
				name:      "token set",
				envToken:  "nats-token",
				wantToken: "nats-token",
			},
			{
				name:     "valid user NKey accepted",
				envNKey:  string(userSeed),
				wantNKey: ed25519.NewKeyFromSeed(userSeedBytes),
			},
			{
				name:    "non-user NKey rejected",
				envNKey: string(accountSeed),
				wantErr: "KRENALIS_NATS_NKEY value is not a user NKey",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				setBaseline(t)
				if tc.envUser != "" {
					t.Setenv("KRENALIS_NATS_USER", tc.envUser)
				}
				if tc.envPass != "" {
					t.Setenv("KRENALIS_NATS_PASSWORD", tc.envPass)
				}
				if tc.envToken != "" {
					t.Setenv("KRENALIS_NATS_TOKEN", tc.envToken)
				}
				if tc.envNKey != "" {
					t.Setenv("KRENALIS_NATS_NKEY", tc.envNKey)
				}
				s, err := parseEnvSettings()
				if tc.wantErr != "" {
					if err == nil {
						t.Fatalf("expected error, got nil")
					}
					if err.Error() != tc.wantErr {
						t.Fatalf("expected %q, got %q", tc.wantErr, err.Error())
					}
					return
				}
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if s.NATS.User != tc.wantUser {
					t.Fatalf("expected %q, got %q", tc.wantUser, s.NATS.User)
				}
				if s.NATS.Password != tc.wantPass {
					t.Fatalf("expected %q, got %q", tc.wantPass, s.NATS.Password)
				}
				if s.NATS.Token != tc.wantToken {
					t.Fatalf("expected %q, got %q", tc.wantToken, s.NATS.Token)
				}
				if len(tc.wantNKey) == 0 {
					if len(s.NATS.NKey) != 0 {
						t.Fatalf("expected empty NKey, got %x", s.NATS.NKey)
					}
				} else if !slices.Equal(s.NATS.NKey, tc.wantNKey) {
					t.Fatalf("expected %x, got %x", tc.wantNKey, s.NATS.NKey)
				}
			})
		}
	})

	t.Run("NATS storage parsing", func(t *testing.T) {
		cases := []struct {
			name    string
			env     string
			want    natsopts.StorageType
			wantErr string
		}{
			{
				name: "default when unset",
				env:  "",
				want: natsopts.FileStorage,
			},
			{
				name: "file storage",
				env:  "file",
				want: natsopts.FileStorage,
			},
			{
				name: "memory storage",
				env:  "memory",
				want: natsopts.MemoryStorage,
			},
			{
				name:    "invalid storage rejected",
				env:     "s3",
				wantErr: "KRENALIS_NATS_STORAGE value \"s3\" is not supported; expected file or memory",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				setBaseline(t)
				if tc.env != "" {
					t.Setenv("KRENALIS_NATS_STORAGE", tc.env)
				}
				s, err := parseEnvSettings()
				if tc.wantErr != "" {
					if err == nil {
						t.Fatalf("expected error, got nil")
					}
					if err.Error() != tc.wantErr {
						t.Fatalf("expected %q, got %q", tc.wantErr, err.Error())
					}
					return
				}
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if s.NATS.Storage != tc.want {
					t.Fatalf("expected %v, got %v", tc.want, s.NATS.Storage)
				}
			})
		}
	})

	t.Run("NATS replicas parsing", func(t *testing.T) {
		cases := []struct {
			name    string
			env     string
			want    int
			wantErr string
		}{
			{
				name: "default when unset",
				env:  "",
				want: 1,
			},
			{
				name: "replicas 1",
				env:  "1",
				want: 1,
			},
			{
				name: "replicas 2",
				env:  "2",
				want: 2,
			},
			{
				name: "replicas 3",
				env:  "3",
				want: 3,
			},
			{
				name: "replicas 4",
				env:  "4",
				want: 4,
			},
			{
				name: "replicas 5",
				env:  "5",
				want: 5,
			},
			{
				name:    "replicas 0 rejected",
				env:     "0",
				wantErr: "KRENALIS_NATS_REPLICAS value \"0\" is not supported; expected 1, 2, 3, 4, or 5",
			},
			{
				name:    "replicas 6 rejected",
				env:     "6",
				wantErr: "KRENALIS_NATS_REPLICAS value \"6\" is not supported; expected 1, 2, 3, 4, or 5",
			},
			{
				name:    "replicas text rejected",
				env:     "two",
				wantErr: "KRENALIS_NATS_REPLICAS value \"two\" is not supported; expected 1, 2, 3, 4, or 5",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				setBaseline(t)
				if tc.env != "" {
					t.Setenv("KRENALIS_NATS_REPLICAS", tc.env)
				}
				s, err := parseEnvSettings()
				if tc.wantErr != "" {
					if err == nil {
						t.Fatalf("expected error, got nil")
					}
					if err.Error() != tc.wantErr {
						t.Fatalf("expected %q, got %q", tc.wantErr, err.Error())
					}
					return
				}
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if s.NATS.Replicas != tc.want {
					t.Fatalf("expected %d, got %d", tc.want, s.NATS.Replicas)
				}
			})
		}
	})

	t.Run("NATS compression parsing", func(t *testing.T) {
		cases := []struct {
			name    string
			env     string
			storage string
			want    natsopts.StoreCompression
			wantErr string
		}{
			{
				name: "default when unset",
				env:  "",
				want: natsopts.NoCompression,
			},
			{
				name:    "compression allowed with file storage",
				env:     "s2",
				storage: "file",
				want:    natsopts.S2Compression,
			},
			{
				name: "s2 compression uppercase",
				env:  "S2",
				want: natsopts.S2Compression,
			},
			{
				name:    "compression with memory storage rejected",
				env:     "s2",
				storage: "memory",
				wantErr: "KRENALIS_NATS_COMPRESSION can be set only when using file storage",
			},
			{
				name:    "invalid compression rejected",
				env:     "gzip",
				wantErr: "KRENALIS_NATS_COMPRESSION value \"gzip\" is not supported; expected s2",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				setBaseline(t)
				if tc.env != "" {
					t.Setenv("KRENALIS_NATS_COMPRESSION", tc.env)
				}
				if tc.storage != "" {
					t.Setenv("KRENALIS_NATS_STORAGE", tc.storage)
				}
				s, err := parseEnvSettings()
				if tc.wantErr != "" {
					if err == nil {
						t.Fatalf("expected error, got nil")
					}
					if err.Error() != tc.wantErr {
						t.Fatalf("expected %q, got %q", tc.wantErr, err.Error())
					}
					return
				}
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if s.NATS.Compression != tc.want {
					t.Fatalf("expected %v, got %v", tc.want, s.NATS.Compression)
				}
			})
		}
	})

	t.Run("boolean flags parsing", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_INVITE_MEMBERS_VIA_EMAIL", "false")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.InviteMembersViaEmail {
			t.Errorf("expected false, got true")
		}

		setBaseline(t)
		t.Setenv("KRENALIS_INVITE_MEMBERS_VIA_EMAIL", "true")
		s, err = parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !s.InviteMembersViaEmail {
			t.Errorf("expected true, got false")
		}

		setBaseline(t)
		t.Setenv("KRENALIS_INVITE_MEMBERS_VIA_EMAIL", "not-bool")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid boolean, got nil")
		}
		want := "KRENALIS_INVITE_MEMBERS_VIA_EMAIL must be a boolean: value \"not-bool\" is not a valid boolean value (expected true, false or empty string)"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

	})

	t.Run("metrics enabled parsing", func(t *testing.T) {

		setBaseline(t)
		t.Setenv("KRENALIS_PROMETHEUS_METRICS_ENABLED", "false")
		s, err := parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.PrometheusMetricsEnabled {
			t.Errorf("expected false, got true")
		}

		setBaseline(t)
		t.Setenv("KRENALIS_PROMETHEUS_METRICS_ENABLED", "true")
		s, err = parseEnvSettings()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !s.PrometheusMetricsEnabled {
			t.Errorf("expected true, got false")
		}

		setBaseline(t)
		t.Setenv("KRENALIS_PROMETHEUS_METRICS_ENABLED", "not-bool")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid boolean, got nil")
		}
		want := "KRENALIS_PROMETHEUS_METRICS_ENABLED must be a boolean: value \"not-bool\" is not a valid boolean value (expected true, false or empty string)"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

	})

	t.Run("SMTP conditional block", func(t *testing.T) {
		// Host set but missing port.
		setBaseline(t)
		t.Setenv("KRENALIS_SMTP_HOST", "smtp.example.com")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error when SMTP host set without port, got nil")
		}
		want := "KRENALIS_SMTP_PORT is required if KRENALIS_SMTP_HOST is set"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Invalid port yields specific error text from code.
		setBaseline(t)
		t.Setenv("KRENALIS_SMTP_HOST", "smtp.example.com")
		t.Setenv("KRENALIS_SMTP_PORT", "0")
		_, err = parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid SMTP port, got nil")
		}
		want = "KRENALIS_SMTP_PORT must be a valid port: port cannot be 0"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}

		// Valid SMTP.
		setBaseline(t)
		t.Setenv("KRENALIS_SMTP_HOST", "smtp.example.com")
		t.Setenv("KRENALIS_SMTP_PORT", "587")
		t.Setenv("KRENALIS_SMTP_USERNAME", "user")
		t.Setenv("KRENALIS_SMTP_PASSWORD", "pass")
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
		t.Setenv("KRENALIS_SMTP_HOST", "bad host")
		t.Setenv("KRENALIS_SMTP_PORT", "25")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid SMTP host, got nil")
		}
		want := "KRENALIS_SMTP_HOST must be a valid host: host is not valid"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("MaxMind DB path set when file exists", func(t *testing.T) {
		setBaseline(t)
		path := createTempFile(t, "GeoIP2-*.mmdb")
		t.Setenv("KRENALIS_MAXMIND_DB_PATH", path)
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
		t.Setenv("KRENALIS_MAXMIND_DB_PATH", nonexistentFile)
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for missing MaxMind db file, got nil")
		}
		want := fmt.Sprintf("KRENALIS_MAXMIND_DB_PATH points to a non-existent file: %q", nonexistentFile)
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("transformers provider invalid", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_TRANSFORMERS_PROVIDER", "unsupported")
		_, err := parseEnvSettings()
		if err == nil {
			t.Fatalf("expected error for invalid transformers provider, got nil")
		}
		want := "invalid KRENALIS_TRANSFORMERS_PROVIDER: want one of local or aws-lambda"
		if err.Error() != want {
			t.Fatalf("expected %q, got %q", want, err)
		}
	})

	t.Run("transformers local-only accepted", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_TRANSFORMERS_PROVIDER", "local")
		t.Setenv("KRENALIS_TRANSFORMERS_LOCAL_NODEJS_EXECUTABLE", "/usr/bin/node")
		if _, err := parseEnvSettings(); err != nil {
			t.Fatalf("expected no error for local-only transformers, got %v", err)
		}
	})

	t.Run("transformers Lambda-only accepted", func(t *testing.T) {
		setBaseline(t)
		t.Setenv("KRENALIS_TRANSFORMERS_PROVIDER", "aws-lambda")
		t.Setenv("KRENALIS_TRANSFORMERS_AWS_LAMBDA_NODEJS_RUNTIME", "nodejs18.x")
		if _, err := parseEnvSettings(); err != nil {
			t.Fatalf("expected no error for Lambda-only transformers, got %v", err)
		}
	})

}

// TestParseEnvURLSuccess verifies valid inputs and normalization behaviors.
func TestParseEnvURLSuccess(t *testing.T) {

	t.Run("empty env returns empty and nil", func(t *testing.T) {
		t.Setenv("KRENALIS_PARSE_ENV_URL_EMPTY", "")
		got, err := parseEnvURL("KRENALIS_PARSE_ENV_URL_EMPTY", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("http without path gets normalized to slash", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_HTTP_NOPATH", "http://example.com")
		got, err := parseEnvURL("KRENALIS_URL_HTTP_NOPATH", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "http://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("https with slash path stays as is", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_HTTPS_SLASH", "https://example.com/")
		got, err := parseEnvURL("KRENALIS_URL_HTTPS_SLASH", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("strip leading zeros in port then default port removal", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_HTTP_LEADING_ZERO_PORT", "http://example.com:00080")
		got, err := parseEnvURL("KRENALIS_URL_HTTP_LEADING_ZERO_PORT", 0)
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
		t.Setenv("KRENALIS_URL_NONDEFAULT_PORT", "https://example.com:8443/")
		got, err := parseEnvURL("KRENALIS_URL_NONDEFAULT_PORT", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com:8443/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("remove default port 443 for https", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_HTTPS_DEFAULT_PORT", "https://example.com:443")
		got, err := parseEnvURL("KRENALIS_URL_HTTPS_DEFAULT_PORT", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("ipv6 literal with port is accepted", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_IPV6_PORT", "http://[::1]:8080")
		got, err := parseEnvURL("KRENALIS_URL_IPV6_PORT", 0)
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
		t.Setenv("KRENALIS_URL_NOPATH_FLAG_BAD", "https://example.com/foo")
		_, err := parseEnvURL("KRENALIS_URL_NOPATH_FLAG_BAD", validation.NoPath)
		wantErr := `invalid URL specified for KRENALIS_URL_NOPATH_FLAG_BAD: path must be "/"`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("noPath allows empty path which normalizes to slash", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_NOPATH_FLAG_OK", "https://example.com")
		got, err := parseEnvURL("KRENALIS_URL_NOPATH_FLAG_OK", validation.NoPath)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "https://example.com/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("noQuery rejects non-empty query", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_NOQUERY_BAD", "https://example.com/?a=b")
		_, err := parseEnvURL("KRENALIS_URL_NOQUERY_BAD", validation.NoQuery)
		wantErr := "invalid URL specified for KRENALIS_URL_NOQUERY_BAD: query cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("noQuery rejects trailing question mark (ForceQuery)", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_NOQUERY_FORCE", "https://example.com/?")
		_, err := parseEnvURL("KRENALIS_URL_NOQUERY_FORCE", validation.NoQuery)
		wantErr := "invalid URL specified for KRENALIS_URL_NOQUERY_FORCE: query cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})
}

// TestParseURLErrors verifies that each distinct error path returns the expected message.
func TestParseURLErrors(t *testing.T) {

	t.Run("leading space", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_SPACE_START", " https://example.com/")
		_, err := parseEnvURL("KRENALIS_URL_SPACE_START", 0)
		wantErr := "invalid URL specified for KRENALIS_URL_SPACE_START: it starts with a space"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("trailing space", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_SPACE_END", "https://example.com/ ")
		_, err := parseEnvURL("KRENALIS_URL_SPACE_END", 0)
		wantErr := "invalid URL specified for KRENALIS_URL_SPACE_END: it ends with a space"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("url.Parse failure", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_PARSE_FAIL", "http://[::1")
		_, err := parseEnvURL("KRENALIS_URL_PARSE_FAIL", 0)
		// We match the exact message format propagated by url.Parse into errInvalidURL.
		wantErr := "invalid URL specified for KRENALIS_URL_PARSE_FAIL: parse \"http://[::1\": missing ']' in host"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("unsupported scheme", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_BAD_SCHEME", "ftp://example.com/")
		_, err := parseEnvURL("KRENALIS_URL_BAD_SCHEME", 0)
		wantErr := `invalid URL specified for KRENALIS_URL_BAD_SCHEME: scheme must be "http" or "https"`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("user info present", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_USERINFO", "http://user:pass@example.com/")
		_, err := parseEnvURL("KRENALIS_URL_USERINFO", 0)
		wantErr := "invalid URL specified for KRENALIS_URL_USERINFO: user and password cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("missing host", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_NO_HOST", "http://")
		_, err := parseEnvURL("KRENALIS_URL_NO_HOST", 0)
		wantErr := "invalid URL specified for KRENALIS_URL_NO_HOST: host must be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: zero", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_PORT_ZERO", "https://example.com:0")
		_, err := parseEnvURL("KRENALIS_URL_PORT_ZERO", 0)
		wantErr := "invalid URL specified for KRENALIS_URL_PORT_ZERO: port cannot be 0"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: above max", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_PORT_BIG", "https://example.com:65536")
		_, err := parseEnvURL("KRENALIS_URL_PORT_BIG", 0)
		wantErr := "invalid URL specified for KRENALIS_URL_PORT_BIG: port must not exceed 65535"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: non numeric", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_PORT_NAN", "https://example.com:abc")
		_, err := parseEnvURL("KRENALIS_URL_PORT_NAN", 0)
		wantErr := `invalid URL specified for KRENALIS_URL_PORT_NAN: parse "https://example.com:abc": invalid port ":abc" after host`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("fragment present", func(t *testing.T) {
		t.Setenv("KRENALIS_URL_FRAGMENT", "https://example.com/#frag")
		_, err := parseEnvURL("KRENALIS_URL_FRAGMENT", 0)
		wantErr := "invalid URL specified for KRENALIS_URL_FRAGMENT: fragment cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})
}
