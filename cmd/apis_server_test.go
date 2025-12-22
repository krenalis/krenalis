// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	stderrors "errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meergo/meergo/tools/errors"
)

func TestParseID(t *testing.T) {

	tests := []struct {
		s  string
		v  int
		ok bool
	}{
		// valid
		{"1", 1, true},
		{"9", 9, true},
		{"10", 10, true},
		{"2147483647", math.MaxInt32, true},

		// invalid: format
		{"", 0, false},
		{"0", 0, false},
		{"01", 0, false},
		{"000", 0, false},
		{"+1", 0, false},
		{"-1", 0, false},
		{" 1", 0, false},
		{"1 ", 0, false},
		{"1\n", 0, false},
		{"\t1", 0, false},
		{"1\t", 0, false},
		{"1a", 0, false},
		{"a1", 0, false},
		{"3.14", 0, false},

		// invalid: overflow
		{"2147483648", 0, false},
		{"9999999999", 0, false},
		{"18446744073709551616", 0, false},

		// invalid: unicode digits
		{"１２３", 0, false},
	}

	for _, test := range tests {
		got, ok := parseID(test.s)
		if ok != test.ok {
			t.Fatalf("%q: expected %t, got %t", test.s, test.ok, ok)
		}
		if ok {
			if got != test.v {
				t.Fatalf("%q: expected %d, got %d", test.s, test.v, got)
			}
		}
	}
}

func TestWriteSessionCookie(t *testing.T) {

	t.Run("ignores empty cookie string", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		writeSessionCookie(recorder, &http.Cookie{})
		if got := recorder.Header()["Set-Cookie"]; len(got) != 0 {
			t.Fatalf("expected no Set-Cookie header, got %v", got)
		}
	})

	t.Run("adds new session cookie", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Add("Set-Cookie", "other=1")

		writeSessionCookie(recorder, &http.Cookie{
			Name:  sessionCookieName,
			Value: "abc",
		})

		got := recorder.Header()["Set-Cookie"]
		want := []string{"other=1", sessionCookieName + "=abc; Priority=High"}
		if len(got) != len(want) {
			t.Fatalf("expected %d set-cookie values, got %d: %v", len(want), len(got), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected Set-Cookie[%d] to be %q, got %q", i, want[i], got[i])
			}
		}
	})

	t.Run("overwrites existing session cookie", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header()["Set-Cookie"] = []string{
			"other=1",
			sessionCookieName + "=old",
		}

		writeSessionCookie(recorder, &http.Cookie{
			Name:  sessionCookieName,
			Value: "new",
		})

		got := recorder.Header()["Set-Cookie"]
		want := []string{"other=1", sessionCookieName + "=new; Priority=High"}
		if len(got) != len(want) {
			t.Fatalf("expected %d set-cookie values, got %d: %v", len(want), len(got), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected Set-Cookie[%d] to be %q, got %q", i, want[i], got[i])
			}
		}
	})

}

// TestValidateForbiddenBody tests the validateForbiddenBody function.
func TestValidateForbiddenBody(t *testing.T) {

	t.Run("allows empty body with zero content length", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		if err := validateForbiddenBody(req); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("allows empty body with unknown length", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
		req.ContentLength = -1

		if err := validateForbiddenBody(req); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("rejects known-length body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abc"))

		err := validateForbiddenBody(req)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if _, ok := err.(*errors.BadRequestError); !ok {
			t.Fatalf("expected *errors.BadRequestError, got %T", err)
		}
		if err.Error() != "request body not allowed" {
			t.Fatalf("expected request body not allowed, got %q", err.Error())
		}
	})

	t.Run("rejects unknown-length body and keeps payload intact", func(t *testing.T) {
		const payload = "payload"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
		req.ContentLength = -1

		err := validateForbiddenBody(req)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request body not allowed" {
			t.Fatalf("expected request body not allowed, got %q", err.Error())
		}
		body, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			t.Fatalf("expected to read body, got %v", readErr)
		}
		if string(body) != payload {
			t.Fatalf("expected body %q, got %q", payload, string(body))
		}
	})

	t.Run("propagates body read errors", func(t *testing.T) {
		testErr := stderrors.New("boom")
		req := httptest.NewRequest(http.MethodPost, "/", errReader{err: testErr})
		req.ContentLength = -1

		err := validateForbiddenBody(req)
		if !stderrors.Is(err, testErr) {
			t.Fatalf("expected %v, got %v", testErr, err)
		}
	})
}

// TestValidateRequiredBody tests the validateRequiredBody function.
func TestValidateRequiredBody(t *testing.T) {

	t.Run("fails when body is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)

		err := validateRequiredBody(req, false)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request's body is missing" {
			t.Fatalf("expected request's body is missing, got %q", err.Error())
		}
	})

	t.Run("fails when content type is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))

		err := validateRequiredBody(req, false)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request's content type must be 'application/json'" {
			t.Fatalf("expected request's content type must be 'application/json', got %q", err.Error())
		}
	})

	t.Run("fails when content type is not json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "text/plain")

		err := validateRequiredBody(req, false)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request's content type must be 'application/json'" {
			t.Fatalf("expected request's content type must be 'application/json', got %q", err.Error())
		}
	})

	t.Run("fails when content type is not json or plain/text", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/xml")

		err := validateRequiredBody(req, true)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request's content type must be 'application/json'" {
			t.Fatalf("expected request's content type must be 'application/json', got %q", err.Error())
		}
	})

	t.Run("fails when content type has unsupported parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json; charset=utf-8; version=1")

		err := validateRequiredBody(req, false)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request's content type must be 'application/json'" {
			t.Fatalf("expected request's content type must be 'application/json', got %q", err.Error())
		}
	})

	t.Run("fails when charset is not utf8", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json; charset=iso-8859-1")

		err := validateRequiredBody(req, false)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request's content type charset must be 'utf-8'" {
			t.Fatalf("expected request's content type charset must be 'utf-8', got %q", err.Error())
		}
	})

	t.Run("accepts valid json content type case-insensitively", func(t *testing.T) {
		const payload = `{"ok":true}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
		req.Header.Set("Content-Type", "Application/Json; Charset=UTF-8")

		if err := validateRequiredBody(req, false); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("expected to read body, got %v", err)
		}
		if string(body) != payload {
			t.Fatalf("expected body %q, got %q", payload, string(body))
		}
	})

	t.Run("normalizes body", func(t *testing.T) {
		const payload = "{\"text\":\"Cafe\u0301\"}"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		if err := validateRequiredBody(req, false); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("expected to read body, got %v", err)
		}
		if string(body) != "{\"text\":\"Café\"}" {
			t.Fatalf("expected normalized body {\"text\":\"Café\"}, got %q", string(body))
		}
	})

	t.Run("fails when payload exceeds limit", func(t *testing.T) {
		oversized := bytes.Repeat([]byte{'a'}, maxRequestSize+1)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(oversized))
		req.Header.Set("Content-Type", "application/json")

		if err := validateRequiredBody(req, false); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		body, err := io.ReadAll(req.Body)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if err.Error() != "request body too large" {
			t.Fatalf("expected request body too large, got %q", err.Error())
		}
		if len(body) != maxRequestSize {
			t.Fatalf("expected to read %d bytes, got %d", maxRequestSize, len(body))
		}
	})

}

type errReader struct {
	err error
}

func (r errReader) Read(p []byte) (int, error) {
	return 0, r.err
}
