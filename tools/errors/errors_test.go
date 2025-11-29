// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package errors

import (
	stdErrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test_BadRequestError ensures BadRequest returns an error wrapping the cause
// and that WriteTo produces the correct HTTP response.
func Test_BadRequestError(t *testing.T) {
	cause := New("wrong")
	err := BadRequest("bad: %w", cause)
	if err.Error() != "bad: wrong" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if err.Unwrap() != cause {
		t.Fatalf("expected unwrap %v, got %v", cause, err.Unwrap())
	}
	expected := "{\"error\":{\"code\":\"BadRequest\",\"message\":\"bad: wrong\",\"cause\":\"wrong\"}}"
	checkResponse(t, err, http.StatusBadRequest, "BadRequest", expected)

	errNoCause := BadRequest("simple")
	expected = "{\"error\":{\"code\":\"BadRequest\",\"message\":\"simple\"}}"
	checkResponse(t, errNoCause, http.StatusBadRequest, "BadRequest", expected)
}

// Test_ForbiddenError checks the Forbidden constructor and its HTTP
// serialization.
func Test_ForbiddenError(t *testing.T) {
	err := Forbidden("forbidden")
	if err.Error() != "forbidden" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	expected := "{\"error\":{\"code\":\"Forbidden\",\"message\":\"forbidden\"}}"
	checkResponse(t, err, http.StatusForbidden, "Forbidden", expected)
}

// Test_NotFoundError verifies NotFound errors format and serialize correctly.
func Test_NotFoundError(t *testing.T) {
	err := NotFound("missing")
	if err.Error() != "missing" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	expected := "{\"error\":{\"code\":\"NotFound\",\"message\":\"missing\"}}"
	checkResponse(t, err, http.StatusNotFound, "NotFound", expected)
}

// Test_StdWrappers checks that the exported wrappers behave like the ones in
// the standard library and operate correctly with errors created by this
// package.
func Test_StdWrappers(t *testing.T) {
	err := New("hello")
	if err.Error() != "hello" {
		t.Fatalf("unexpected New error string: %q", err.Error())
	}

	wrap := fmt.Errorf("wrap: %w", err)
	if Unwrap(wrap) != err {
		t.Fatalf("Unwrap failed")
	}
	if !Is(wrap, err) {
		t.Fatalf("Is failed")
	}
	var target error
	if !As(wrap, &target) {
		t.Fatalf("As failed")
	}
	if target != wrap {
		t.Fatalf("As returned %v, expected %v", target, wrap)
	}

	// confirm behavior matches std errors functions
	if stdErrors.Unwrap(wrap) != err || !stdErrors.Is(wrap, err) {
		t.Fatalf("stdlib mismatch")
	}
}

// Test_UnauthorizedError verifies Unauthorized formatting and serialization.
func Test_UnauthorizedError(t *testing.T) {
	err := Unauthorized("no auth")
	if err.Error() != "no auth" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	expected := "{\"error\":{\"code\":\"Unauthorized\",\"message\":\"no auth\"}}"
	checkResponse(t, err, http.StatusUnauthorized, "Unauthorized", expected)
}

// Test_UnavailableError checks Unavailable wrapping and response behaviour.
func Test_UnavailableError(t *testing.T) {
	cause := New("temp")
	err := Unavailable("unavail: %w", cause)
	if err.Error() != "unavail: temp" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if err.Unwrap() != cause {
		t.Fatalf("expected unwrap %v, got %v", cause, err.Unwrap())
	}
	expected := "{\"error\":{\"code\":\"ServiceUnavailable\",\"message\":\"unavail: temp\",\"cause\":\"temp\"}}"
	checkResponse(t, err, http.StatusServiceUnavailable, "ServiceUnavailable", expected)
}

// Test_UnprocessableError ensures Unprocessable includes the error code and
// wrapped cause in the HTTP response.
func Test_UnprocessableError(t *testing.T) {
	cause := New("bad state")
	err := Unprocessable("SomeCode", "cannot: %w", cause)
	if err.Error() != "cannot: bad state" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if err.Unwrap() != cause {
		t.Fatalf("expected unwrap %v, got %v", cause, err.Unwrap())
	}
	expected := "{\"error\":{\"code\":\"SomeCode\",\"message\":\"cannot: bad state\",\"cause\":\"bad state\"}}"
	checkResponse(t, err, http.StatusUnprocessableEntity, "SomeCode", expected)
}

func checkResponse(t *testing.T, err ResponseWriterTo, status int, code Code, body string) {
	t.Helper()
	rr := httptest.NewRecorder()
	if werr := err.WriteTo(rr); werr != nil {
		t.Fatalf("WriteTo error: %v", werr)
	}
	if rr.Code != status {
		t.Fatalf("expected status %d, got %d", status, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", ct)
	}
	if cl := rr.Header().Get("Content-Length"); cl != fmt.Sprint(len(body)) {
		t.Fatalf("unexpected content length: %s", cl)
	}
	if got := rr.Body.String(); got != body {
		t.Fatalf("expected body %q, got %q", body, got)
	}
}
