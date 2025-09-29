//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
