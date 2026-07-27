// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
)

// newHTTP returns an HTTP with no state, and makes the organizations with the
// given IDs the existing ones, as New does with the ones of a state.
func newHTTP(t *testing.T, organizationIDs ...string) *HTTP {
	t.Helper()
	h := New(nil, http.DefaultTransport.(*http.Transport))
	h.organizationsMu.Lock()
	for _, id := range organizationIDs {
		h.organizations[id] = &organization{}
	}
	h.listening = true
	h.organizationsMu.Unlock()
	return h
}

// client returns a client of the organization with the given ID, making its
// requests with the transport of the organization.
func client(t *testing.T, h *HTTP, organizationID string) *Client {
	t.Helper()
	return h.ConnectorClient(&state.Connector{Code: "test"}, organizationID, "", "")
}

// get sends a GET request to url with c, and returns the error it fails with.
// It closes the body of the response, if there is one.
func get(t *testing.T, c *Client, url string) error {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Do(req)
	if res != nil {
		_ = res.Body.Close()
	}
	return err
}

// server starts a server that responds with 200 to every request, and returns
// its URL. The server is closed when the test ends.
func server(t *testing.T) string {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(s.Close)
	return s.URL
}

// transportOf returns the transport of the organization with the given ID, and
// reports whether the organization has one at all.
func transportOf(t *testing.T, h *HTTP, organizationID string) (*organizationTransport, bool) {
	t.Helper()
	h.organizationsMu.Lock()
	defer h.organizationsMu.Unlock()
	org, ok := h.organizations[organizationID]
	if !ok {
		return nil, false
	}
	return org.transport, org.transport != nil
}

func TestTransportCreatedWhenNeeded(t *testing.T) {
	// The transport of an organization is not created when the organization is
	// created, but only when the organization needs one.
	h := newHTTP(t)
	h.onCreateOrganization(state.CreateOrganization{ID: "org-lazy"})
	if _, ok := transportOf(t, h, "org-lazy"); ok {
		t.Fatal("the created organization already has a transport, expecting none")
	}

	// Every client of the organization makes its requests with the same
	// transport, so that they all share the same connection pool.
	transport := client(t, h, "org-lazy").transport
	created, ok := transportOf(t, h, "org-lazy")
	if !ok {
		t.Fatal("the organization has no transport, expecting one")
	}
	if transport != created {
		t.Fatal("the client does not make its requests with the transport of its organization")
	}
	if other := client(t, h, "org-lazy").transport; other != transport {
		t.Fatal("two clients of the same organization have different transports")
	}

	// The transport of an organization is not shared with the other
	// organizations, nor with the requests that are not made on behalf of one.
	h.onCreateOrganization(state.CreateOrganization{ID: "org-other"})
	if other := client(t, h, "org-other").transport; other == transport {
		t.Fatal("two organizations share the same transport")
	}
	if none := client(t, h, "").transport; none != h.transport {
		t.Fatal("a client with no organization does not make its requests with the base transport")
	}
}

func TestTransportWithoutCounting(t *testing.T) {
	// Counting is disabled, because dialer.EnableCounting is not called, so
	// there is nothing to attribute to the organization and its requests are
	// made with the base transport as it is.
	h := New(nil, http.DefaultTransport.(*http.Transport))
	url := server(t)
	c := client(t, h, "org-not-counted")
	if err := get(t, c, url); err != nil {
		t.Fatalf("the request of an organization failed: %s", err)
	}
	transport, ok := transportOf(t, h, "org-not-counted")
	if !ok {
		t.Fatal("the organization has no transport, expecting one")
	}
	if transport.base != http.RoundTripper(h.transport) {
		t.Fatal("the transport of the organization is a clone, expecting the base transport")
	}

	// Deleting the organization discards its transport without closing the idle
	// connections of the base transport, which is shared.
	h.onDeleteOrganization(state.DeleteOrganization{ID: "org-not-counted"})
	if err := get(t, client(t, h, "org-other"), url); err != nil {
		t.Fatalf("the request of another organization failed: %s", err)
	}
}

func TestTransportUnknownOrganization(t *testing.T) {
	// The organization does not exist, because it has never been created, so its
	// requests fail and no transport is created for it.
	h := newHTTP(t, "org-known")
	url := server(t)
	err := get(t, client(t, h, "org-unknown"), url)
	if !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("the request returned the error %v, expecting ErrNoOrganization", err)
	}
	if _, ok := transportOf(t, h, "org-unknown"); ok {
		t.Fatal("a transport is created for an organization that does not exist")
	}
}

func TestTransportCreatedOrganization(t *testing.T) {
	// An organization created after the HTTP exists, so its requests are sent.
	h := newHTTP(t, "org-known")
	url := server(t)
	h.onCreateOrganization(state.CreateOrganization{ID: "org-created"})
	if err := get(t, client(t, h, "org-created"), url); err != nil {
		t.Fatalf("the request of a created organization failed: %s", err)
	}
}

func TestTransportDeletedOrganization(t *testing.T) {
	// The transport of a deleted organization is discarded, and the clients
	// created before the deletion, which keep it, no longer send requests.
	h := newHTTP(t, "org-deleted")
	url := server(t)
	c := client(t, h, "org-deleted")
	if err := get(t, c, url); err != nil {
		t.Fatalf("the request of an existing organization failed: %s", err)
	}

	h.onDeleteOrganization(state.DeleteOrganization{ID: "org-deleted"})

	h.organizationsMu.Lock()
	_, kept := h.organizations["org-deleted"]
	h.organizationsMu.Unlock()
	if kept {
		t.Fatal("the deleted organization is still kept")
	}
	err := get(t, c, url)
	if !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("the request returned the error %v, expecting ErrNoOrganization", err)
	}

	// A client created after the deletion does not send requests either.
	err = get(t, client(t, h, "org-deleted"), url)
	if !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("the request returned the error %v, expecting ErrNoOrganization", err)
	}

	// Deleting an organization that has no transport, or that does not exist,
	// does nothing.
	h.onCreateOrganization(state.CreateOrganization{ID: "org-no-transport"})
	h.onDeleteOrganization(state.DeleteOrganization{ID: "org-no-transport"})
	h.onDeleteOrganization(state.DeleteOrganization{ID: "org-never-created"})
}

func TestTransportWithoutListening(t *testing.T) {
	// The HTTP has no state, so the organizations are not known and every one of
	// them is considered to exist.
	h := New(nil, http.DefaultTransport.(*http.Transport))
	url := server(t)
	if err := get(t, client(t, h, "org-not-listening"), url); err != nil {
		t.Fatalf("the request of an unknown organization failed: %s", err)
	}
	if _, ok := transportOf(t, h, "org-not-listening"); !ok {
		t.Fatal("the organization has no transport, expecting one")
	}
}
