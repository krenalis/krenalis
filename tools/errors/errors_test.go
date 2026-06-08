// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package errors

import (
	"errors"
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
	expected := "{\"error\":{\"code\":\"BadRequest\",\"message\":\"bad: wrong\",\"cause\":\"wrong\"}}\n"
	checkResponse(t, err, http.StatusBadRequest, "BadRequest", expected)

	errNoCause := BadRequest("simple")
	expected = "{\"error\":{\"code\":\"BadRequest\",\"message\":\"simple\"}}\n"
	checkResponse(t, errNoCause, http.StatusBadRequest, "BadRequest", expected)
}

// Test_ForbiddenError checks the Forbidden constructor and its HTTP
// serialization.
func Test_ForbiddenError(t *testing.T) {
	err := Forbidden("forbidden")
	if err.Error() != "forbidden" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	expected := "{\"error\":{\"code\":\"Forbidden\",\"message\":\"forbidden\"}}\n"
	checkResponse(t, err, http.StatusForbidden, "Forbidden", expected)
}

// Test_NotFoundError verifies NotFound errors format and serialize correctly.
func Test_NotFoundError(t *testing.T) {
	err := NotFound("missing")
	if err.Error() != "missing" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	expected := "{\"error\":{\"code\":\"NotFound\",\"message\":\"missing\"}}\n"
	checkResponse(t, err, http.StatusNotFound, "NotFound", expected)
}

// Test_StdWrappers checks that the exported wrappers behave like the ones in
// the standard library and operate correctly with errors created by this
// package.
func Test_StdWrappers(t *testing.T) {
	err := New("hello")
	if err.Error() != "hello" {
		t.Fatalf("expected New error string %q, got %q", "hello", err.Error())
	}

	wrap := fmt.Errorf("wrap: %w", err)
	if got := Unwrap(wrap); got != err {
		t.Fatalf("expected Unwrap to return %v, got %v", err, got)
	}
	if got := Is(wrap, err); !got {
		t.Fatalf("expected Is to return true, got %t", got)
	}
	var target error
	if got := As(wrap, &target); !got {
		t.Fatalf("expected As to return true, got %t", got)
	}
	if target != wrap {
		t.Fatalf("expected As target %v, got %v", wrap, target)
	}

	typedErr := NotFound("missing")
	typedWrap := fmt.Errorf("wrap: %w", typedErr)
	got, ok := AsType[*NotFoundError](typedWrap)
	if got != typedErr || !ok {
		t.Fatalf("expected AsType to return %v, true, got %v, %t", typedErr, got, ok)
	}

	missing, missingOK := AsType[*ForbiddenError](typedWrap)
	if missing != nil || missingOK {
		t.Fatalf("expected AsType to return nil, false, got %v, %t", missing, missingOK)
	}

	// confirm behavior matches std errors functions
	if got := errors.Unwrap(wrap); got != err {
		t.Fatalf("expected stdlib Unwrap to return %v, got %v", err, got)
	}
	if got := errors.Is(wrap, err); !got {
		t.Fatalf("expected stdlib Is to return true, got %t", got)
	}
}

// Test_UnauthorizedError verifies Unauthorized formatting and serialization.
func Test_UnauthorizedError(t *testing.T) {
	err := Unauthorized("no auth")
	if err.Error() != "no auth" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	expected := "{\"error\":{\"code\":\"Unauthorized\",\"message\":\"no auth\"}}\n"
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
	expected := "{\"error\":{\"code\":\"ServiceUnavailable\",\"message\":\"unavail: temp\",\"cause\":\"temp\"}}\n"
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
	expected := "{\"error\":{\"code\":\"SomeCode\",\"message\":\"cannot: bad state\",\"cause\":\"bad state\"}}\n"
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
