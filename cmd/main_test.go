//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package cmd

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestEnvLoading(t *testing.T) {

	// Load the environment variables form 'test-env-file.env'.
	err := godotenv.Overload("testdata/test-env-file.env")
	if err != nil {
		t.Fatal(err)
	}

	// Determine the got key-values from the environment variables loaded by
	// godotenv.
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

		// TODO(Gianluca): this is caused by a bug in the parsing library.
		// See https://github.com/meergo/meergo/issues/1655 and
		// https://github.com/joho/godotenv/issues/226.
		// "MEERGO_ENV_TEST_G": "\"my-quoted-value\"",

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

// TestParseURLSuccess verifies valid inputs and normalization behaviors.
func TestParseURLSuccess(t *testing.T) {

	t.Run("empty env returns empty and nil", func(t *testing.T) {
		t.Setenv("MEERGO_PARSE_URL_EMPTY", "")
		got, err := parseURL("MEERGO_PARSE_URL_EMPTY", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("http without path gets normalized to slash", func(t *testing.T) {
		t.Setenv("MEERGO_URL_HTTP_NOPATH", "http://example.com")
		got, err := parseURL("MEERGO_URL_HTTP_NOPATH", 0)
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
		got, err := parseURL("MEERGO_URL_HTTPS_SLASH", 0)
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
		got, err := parseURL("MEERGO_URL_HTTP_LEADING_ZERO_PORT", 0)
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
		got, err := parseURL("MEERGO_URL_NONDEFAULT_PORT", 0)
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
		got, err := parseURL("MEERGO_URL_HTTPS_DEFAULT_PORT", 0)
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
		got, err := parseURL("MEERGO_URL_IPV6_PORT", 0)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := "http://[::1]:8080/"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})
}

// TestParseURLFlags verifies behavior controlled by validation flags.
func TestParseURLFlags(t *testing.T) {

	t.Run("noPath rejects non-root path", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NOPATH_FLAG_BAD", "https://example.com/foo")
		_, err := parseURL("MEERGO_URL_NOPATH_FLAG_BAD", noPath)
		wantErr := `invalid URL specified for MEERGO_URL_NOPATH_FLAG_BAD: path must be "/"`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("noPath allows empty path which normalizes to slash", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NOPATH_FLAG_OK", "https://example.com")
		got, err := parseURL("MEERGO_URL_NOPATH_FLAG_OK", noPath)
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
		_, err := parseURL("MEERGO_URL_NOQUERY_BAD", noQuery)
		wantErr := "invalid URL specified for MEERGO_URL_NOQUERY_BAD: query cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("noQuery rejects trailing question mark (ForceQuery)", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NOQUERY_FORCE", "https://example.com/?")
		_, err := parseURL("MEERGO_URL_NOQUERY_FORCE", noQuery)
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
		_, err := parseURL("MEERGO_URL_SPACE_START", 0)
		wantErr := "invalid URL specified for MEERGO_URL_SPACE_START: it starts with a space"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("trailing space", func(t *testing.T) {
		t.Setenv("MEERGO_URL_SPACE_END", "https://example.com/ ")
		_, err := parseURL("MEERGO_URL_SPACE_END", 0)
		wantErr := "invalid URL specified for MEERGO_URL_SPACE_END: it ends with a space"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("url.Parse failure", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PARSE_FAIL", "http://[::1")
		_, err := parseURL("MEERGO_URL_PARSE_FAIL", 0)
		// We match the exact message format propagated by url.Parse into errInvalidURL.
		wantErr := "invalid URL specified for MEERGO_URL_PARSE_FAIL: parse \"http://[::1\": missing ']' in host"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("unsupported scheme", func(t *testing.T) {
		t.Setenv("MEERGO_URL_BAD_SCHEME", "ftp://example.com/")
		_, err := parseURL("MEERGO_URL_BAD_SCHEME", 0)
		wantErr := `invalid URL specified for MEERGO_URL_BAD_SCHEME: scheme must be "http" or "https"`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("user info present", func(t *testing.T) {
		t.Setenv("MEERGO_URL_USERINFO", "http://user:pass@example.com/")
		_, err := parseURL("MEERGO_URL_USERINFO", 0)
		wantErr := "invalid URL specified for MEERGO_URL_USERINFO: user and password cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("missing host", func(t *testing.T) {
		t.Setenv("MEERGO_URL_NO_HOST", "http://")
		_, err := parseURL("MEERGO_URL_NO_HOST", 0)
		wantErr := "invalid URL specified for MEERGO_URL_NO_HOST: host must be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: zero", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PORT_ZERO", "https://example.com:0")
		_, err := parseURL("MEERGO_URL_PORT_ZERO", 0)
		wantErr := "invalid URL specified for MEERGO_URL_PORT_ZERO: port must be in range [1,65535]"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: above max", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PORT_BIG", "https://example.com:65536")
		_, err := parseURL("MEERGO_URL_PORT_BIG", 0)
		wantErr := "invalid URL specified for MEERGO_URL_PORT_BIG: port must be in range [1,65535]"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("invalid port: non numeric", func(t *testing.T) {
		t.Setenv("MEERGO_URL_PORT_NAN", "https://example.com:abc")
		_, err := parseURL("MEERGO_URL_PORT_NAN", 0)
		wantErr := `invalid URL specified for MEERGO_URL_PORT_NAN: parse "https://example.com:abc": invalid port ":abc" after host`
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})

	t.Run("fragment present", func(t *testing.T) {
		t.Setenv("MEERGO_URL_FRAGMENT", "https://example.com/#frag")
		_, err := parseURL("MEERGO_URL_FRAGMENT", 0)
		wantErr := "invalid URL specified for MEERGO_URL_FRAGMENT: fragment cannot be specified"
		if err == nil || err.Error() != wantErr {
			t.Fatalf("expected %q, got %v", wantErr, err)
		}
	})
}
