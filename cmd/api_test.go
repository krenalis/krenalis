// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/cmd/internal/workos"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestOnboard verifies how the onboarding endpoint handles a request when
// WorkOS is disabled, when the honeypot field is filled in, and when the
// request is malformed or carries an invalid admin email address.
func TestOnboard(t *testing.T) {

	// The WorkOS of the enabled server has no Core: a request that reaches the
	// creation of an organization fails the test.
	enabled := api{&apisServer{workOS: &workos.WorkOS{}}}
	disabled := api{&apisServer{}}

	t.Run("reports not found when WorkOS is disabled", func(t *testing.T) {
		body := `{"organizationName":"Acme","adminEmail":"admin@example.com"}`
		_, err := disabled.Onboard(nil, newOnboardingRequest(body))
		if err != nil {
			if _, ok := errors.AsType[*errors.NotFoundError](err); !ok {
				t.Fatalf("expected *errors.NotFoundError, got %T", err)
			}
			return
		}
		t.Fatal("expected error, got nil")
	})

	t.Run("ignores a request that fills in the honeypot", func(t *testing.T) {
		body := `{"organizationName":"Acme","adminEmail":"admin@example.com","website":"https://example.com"}`
		result, err := enabled.Onboard(nil, newOnboardingRequest(body))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result != nil {
			t.Errorf("expected no result, got %v", result)
		}
	})

	t.Run("rejects a request without body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/onboarding", nil)
		_, err := enabled.Onboard(nil, r)
		if err != nil {
			if err.Error() != "request's body is missing" {
				t.Fatalf("expected error %q, got %q", "request's body is missing", err)
			}
			return
		}
		t.Fatal("expected error, got nil")
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		_, err := enabled.Onboard(nil, newOnboardingRequest(`{"organizationName":`))
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected *errors.BadRequestError, got %T", err)
			}
			return
		}
		t.Fatal("expected error, got nil")
	})

	t.Run("rejects an invalid admin email", func(t *testing.T) {
		_, err := enabled.Onboard(nil, newOnboardingRequest(`{"organizationName":"Acme","adminEmail":"admin"}`))
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected *errors.BadRequestError, got %T", err)
			}
			return
		}
		t.Fatal("expected error, got nil")
	})

}

// TestSplitQueryParameters verifies comma-separated query parameter splitting.
func TestSplitQueryParameters(t *testing.T) {
	tests := []struct {
		name      string
		in        []string
		want      []string
		sameSlice bool
	}{
		{
			name: "nil slice",
			in:   nil,
			want: nil,
		},
		{
			name: "empty slice",
			in:   []string{},
			want: nil,
		},
		{
			name:      "single value without comma",
			in:        []string{"foo"},
			want:      []string{"foo"},
			sameSlice: true,
		},
		{
			name:      "multiple values without commas",
			in:        []string{"a", "b", "c"},
			want:      []string{"a", "b", "c"},
			sameSlice: true,
		},
		{
			name: "single value with commas",
			in:   []string{"a,b,c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "mixed plain and comma-separated values",
			in:   []string{"foo", "bar,baz"},
			want: []string{"foo", "bar", "baz"},
		},
		{
			name: "values with spaces around commas",
			in:   []string{" x , y , z "},
			want: []string{"x", "y", "z"},
		},
		{
			name: "values with mixed alphanumeric segments",
			in:   []string{"a1,b2,c3"},
			want: []string{"a1", "b2", "c3"},
		},
		{
			name: "values with empty segments",
			in:   []string{",foo,,bar,"},
			want: []string{"foo", "bar"},
		},
		{
			name: "only empty or whitespace segments",
			in:   []string{", , ,", "\t", "\n", " ", ","},
			want: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := splitQueryParameters(test.in)
			if !slices.Equal(test.want, got) {
				t.Errorf("%v: expected %v, got %v", test.in, test.want, got)
			}
		})
	}
}

// newOnboardingRequest returns a POST request to the onboarding endpoint with
// the given JSON body.
func newOnboardingRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/onboarding", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}
