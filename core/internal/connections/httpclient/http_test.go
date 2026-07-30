// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
)

// newHTTP returns an HTTP with no state following the organizations with the
// given IDs, as New does with the ones of a state when the bytes sent are
// counted. Every other organization does not exist.
func newHTTP(t *testing.T, organizationIDs ...string) *HTTP {
	t.Helper()
	h := New(nil, http.DefaultTransport.(*http.Transport))
	h.organizationsMu.Lock()
	h.organizations = map[string]*http.Transport{}
	for _, id := range organizationIDs {
		h.organizations[id] = nil
	}
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
func transportOf(t *testing.T, h *HTTP, organizationID string) (*http.Transport, bool) {
	t.Helper()
	h.organizationsMu.Lock()
	defer h.organizationsMu.Unlock()
	transport := h.organizations[organizationID]
	return transport, transport != nil
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
	if none := h.PlainConnectorClient(&state.Connector{Code: "test"}).transport; none != h.transport {
		t.Fatal("a client with no organization does not make its requests with the base transport")
	}

	// The organization is mandatory: a client whose requests are not made on
	// behalf of one is created with PlainConnectorClient, so an empty
	// organization is a broken call site and it panics.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a client with an empty organization did not panic")
			}
		}()
		client(t, h, "")
	}()
}

func TestTransportWithoutCounting(t *testing.T) {
	// The organizations are not followed, because there is no state and counting
	// is disabled, so there is nothing to attribute to the organization: its
	// requests are made with the base transport, as it is, and no transport is
	// cloned for it.
	h := New(nil, http.DefaultTransport.(*http.Transport))
	url := server(t)
	c := client(t, h, "org-not-counted")
	if err := get(t, c, url); err != nil {
		t.Fatalf("the request of an organization failed: %s", err)
	}
	if c.transport != http.RoundTripper(h.transport) {
		t.Fatal("the requests of the organization are made with a clone, expecting the base transport")
	}
	if _, ok := transportOf(t, h, "org-not-counted"); ok {
		t.Fatal("a transport is created for an organization while the organizations are not followed")
	}
}

func TestTransportUnknownOrganization(t *testing.T) {
	// The organization does not exist, because it has never been created. Its
	// requests are not refused, they are simply made with the base transport,
	// counting nothing, and no transport is created for it.
	h := newHTTP(t, "org-known")
	url := server(t)
	c := client(t, h, "org-unknown")
	if err := get(t, c, url); err != nil {
		t.Fatalf("the request of an organization that does not exist failed: %s", err)
	}
	if c.transport != http.RoundTripper(h.transport) {
		t.Fatal("the requests of an organization that does not exist are not made with the base transport")
	}
	if _, ok := transportOf(t, h, "org-unknown"); ok {
		t.Fatal("a transport is created for an organization that does not exist")
	}
}

func TestTransportCreatedOrganization(t *testing.T) {
	// An organization created after the HTTP exists, so its requests are made
	// with a transport of its own, attributing to it the bytes they send.
	h := newHTTP(t, "org-known")
	url := server(t)
	h.onCreateOrganization(state.CreateOrganization{ID: "org-created"})
	c := client(t, h, "org-created")
	if err := get(t, c, url); err != nil {
		t.Fatalf("the request of a created organization failed: %s", err)
	}
	if c.transport == http.RoundTripper(h.transport) {
		t.Fatal("the requests of a created organization are made with the base transport, expecting its own")
	}
}

func TestTransportDeletedOrganization(t *testing.T) {
	// The transport of a deleted organization is discarded, so that the
	// transports do not accumulate for the whole life of the process.
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

	// A client created before the deletion keeps the transport it was created
	// with, and it still makes its requests with it: the bytes they send go to a
	// counter that is no longer collected.
	if err := get(t, c, url); err != nil {
		t.Fatalf("the request of a client created before the deletion failed: %s", err)
	}

	// A client created after the deletion is one of an organization that does
	// not exist, so its requests are made with the base transport.
	after := client(t, h, "org-deleted")
	if err := get(t, after, url); err != nil {
		t.Fatalf("the request of a client created after the deletion failed: %s", err)
	}
	if after.transport != http.RoundTripper(h.transport) {
		t.Fatal("the requests of a deleted organization are not made with the base transport")
	}

	// Deleting an organization that has no transport, or that does not exist,
	// does nothing.
	h.onCreateOrganization(state.CreateOrganization{ID: "org-no-transport"})
	h.onDeleteOrganization(state.DeleteOrganization{ID: "org-no-transport"})
	h.onDeleteOrganization(state.DeleteOrganization{ID: "org-never-created"})
}
