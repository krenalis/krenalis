// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package workos

import (
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip dispatches requests to the wrapped transport function.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// sentRequest holds the parts of a request sent to WorkOS that the tests check.
type sentRequest struct {
	method        string
	url           string
	authorization string
	contentType   string
	body          map[string]string
}

// TestCreateOrganization verifies the request that createOrganization sends to
// WorkOS and how it handles the response.
func TestCreateOrganization(t *testing.T) {

	t.Run("creates the WorkOS organization", func(t *testing.T) {

		var sent sentRequest
		wo := newTestWorkOS(t, &sent, http.StatusCreated, `{"id":"org_01JQ","name":"Acme"}`)

		id, err := wo.createOrganization(t.Context(), "Acme", "9RbU4nP8Ly12")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != "org_01JQ" {
			t.Errorf("expected organization ID %q, got %q", "org_01JQ", id)
		}
		if sent.method != http.MethodPost {
			t.Errorf("expected method %s, got %s", http.MethodPost, sent.method)
		}
		if sent.url != "https://api.workos.com/organizations" {
			t.Errorf("expected URL %q, got %q", "https://api.workos.com/organizations", sent.url)
		}
		if sent.authorization != "Bearer sk_test" {
			t.Errorf("expected authorization %q, got %q", "Bearer sk_test", sent.authorization)
		}
		if sent.contentType != "application/json" {
			t.Errorf("expected content type %q, got %q", "application/json", sent.contentType)
		}
		want := map[string]string{"name": "Acme", "external_id": "9RbU4nP8Ly12"}
		if !maps.Equal(sent.body, want) {
			t.Errorf("expected body %v, got %v", want, sent.body)
		}

	})

	t.Run("reports an empty organization ID", func(t *testing.T) {

		var sent sentRequest
		wo := newTestWorkOS(t, &sent, http.StatusCreated, `{"id":""}`)

		id, err := wo.createOrganization(t.Context(), "Acme", "9RbU4nP8Ly12")
		if err != nil {
			if err.Error() != "WorkOS returned an empty organization ID" {
				t.Fatalf("expected error %q, got %q", "WorkOS returned an empty organization ID", err)
			}
			return
		}
		t.Fatalf("expected error, got organization ID %q", id)

	})

	t.Run("reports an unexpected status", func(t *testing.T) {

		var sent sentRequest
		wo := newTestWorkOS(t, &sent, http.StatusUnauthorized, `{"message":"Unauthorized"}`)

		id, err := wo.createOrganization(t.Context(), "Acme", "9RbU4nP8Ly12")
		if err != nil {
			if !strings.Contains(err.Error(), "failed to create the WorkOS organization") {
				t.Fatalf("expected a creation failure, got %q", err)
			}
			return
		}
		t.Fatalf("expected error, got organization ID %q", id)

	})

}

// TestSendInvitation verifies the request that sendInvitation sends to WorkOS
// and how it handles the response.
func TestSendInvitation(t *testing.T) {

	t.Run("invites the admin", func(t *testing.T) {

		var sent sentRequest
		wo := newTestWorkOS(t, &sent, http.StatusCreated, `{"id":"invitation_01JQ"}`)

		err := wo.sendInvitation(t.Context(), "admin@example.com", "org_01JQ")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if sent.method != http.MethodPost {
			t.Errorf("expected method %s, got %s", http.MethodPost, sent.method)
		}
		if sent.url != "https://api.workos.com/user_management/invitations" {
			t.Errorf("expected URL %q, got %q", "https://api.workos.com/user_management/invitations", sent.url)
		}
		if sent.authorization != "Bearer sk_test" {
			t.Errorf("expected authorization %q, got %q", "Bearer sk_test", sent.authorization)
		}
		want := map[string]string{"email": "admin@example.com", "organization_id": "org_01JQ", "role_slug": "admin"}
		if !maps.Equal(sent.body, want) {
			t.Errorf("expected body %v, got %v", want, sent.body)
		}

	})

	t.Run("reports an unexpected status", func(t *testing.T) {

		var sent sentRequest
		wo := newTestWorkOS(t, &sent, http.StatusUnprocessableEntity, `{"message":"Invitation already exists"}`)

		err := wo.sendInvitation(t.Context(), "admin@example.com", "org_01JQ")
		if err != nil {
			if !strings.Contains(err.Error(), "failed to send the WorkOS invitation") {
				t.Fatalf("expected an invitation failure, got %q", err)
			}
			return
		}
		t.Fatal("expected error, got nil")

	})

}

// newTestWorkOS returns a WorkOS that records into sent the single request it
// sends, and answers it with the given status and body.
func newTestWorkOS(t *testing.T, sent *sentRequest, status int, body string) *WorkOS {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("expected no error reading the request body, got %v", err)
		}
		sent.method = r.Method
		sent.url = r.URL.String()
		sent.authorization = r.Header.Get("Authorization")
		sent.contentType = r.Header.Get("Content-Type")
		err = json.Unmarshal(raw, &sent.body)
		if err != nil {
			t.Errorf("expected a JSON request body, got %q", raw)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	return &WorkOS{apiKey: "sk_test", transport: transport}
}
