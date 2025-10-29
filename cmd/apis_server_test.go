// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
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
