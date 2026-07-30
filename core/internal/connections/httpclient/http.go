// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package httpclient provides an HTTP client with OAuth support for
// connections.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/dialer"
	"github.com/krenalis/krenalis/core/internal/state"
)

type noOpHandler struct{}

func (h noOpHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

var noOpHandle = noOpHandler{}

// ErrNoOrganization is the error the requests of a client fail with when the
// organization they are made on behalf of does not exist, because it has been
// deleted or it has never been created.
var ErrNoOrganization = errors.New("organization does not exist")

// HTTP allows creating HTTP clients for connections and enables granting,
// retrieving, and refreshing OAuth access tokens.
type HTTP struct {

	// state is nil if the HTTP client was instantiated without providing a
	// state; in that case, methods related to OAuth cannot be used, as their
	// behavior may be unexpected or cause a panic.
	state *state.State

	// transport is the base transport: the transport of each organization is a
	// clone of it, so that they all share its timeouts and options, and it is
	// used as it is for the requests that are not attributed to an organization.
	transport *http.Transport

	trace io.Writer

	// organizations holds the organizations the requests can be made on behalf
	// of, by ID. An organization is added when it is created and it is removed
	// when it is deleted, so that its transport is discarded with it.
	organizationsMu sync.Mutex               // protects organizations and listening
	organizations   map[string]*organization // by organization ID; protected by organizationsMu

	// listening reports whether the organizations are known, that is whether
	// this HTTP has a state to listen to. Until they are, every organization is
	// considered to exist, because there is no way to tell which ones do.
	listening bool // protected by organizationsMu

	// muxes maps each connector code to the corresponding ServeMux handling its rate limits.
	mu    sync.Mutex                // protect muxes
	muxes map[string]*http.ServeMux // nil if state is nil; protected by mu
}

// organization is an organization the requests of a client are made on behalf
// of.
type organization struct {
	// transport is the transport making the requests of the organization. It is
	// created the first time the organization needs one, and it is only written
	// with organizationsMu held.
	transport *organizationTransport
	// deleted reports whether the organization has been deleted. It is the only
	// field its transport reads after it has been created.
	deleted atomic.Bool
}

// New returns an HTTP instance given the state and the transport to use for
// HTTP connections.
//
// The returned HTTP follows the organizations of the state, so that the
// transport of an organization is discarded when the organization is deleted,
// and the requests made on behalf of an organization that does not exist fail
// with [ErrNoOrganization].
//
// It is possible to provide a nil state; in that case the returned HTTP client
// will be restricted and will not allow invocation of OAuth-related methods, as
// their behavior may be unexpected or may cause a panic. Moreover, it does not
// know which organizations exist, so every organization is considered to exist
// and no transport is ever discarded.
func New(state *state.State, transport *http.Transport) *HTTP {
	h := &HTTP{
		state:         state,
		transport:     transport,
		organizations: map[string]*organization{},
	}
	h.muxes = map[string]*http.ServeMux{}
	if state != nil {
		state.Freeze()
		state.AddListener(h.onCreateOrganization)
		state.AddListener(h.onDeleteOrganization)
		h.organizationsMu.Lock()
		for _, org := range state.Organizations() {
			h.organizations[org.ID] = &organization{}
		}
		h.listening = true
		h.organizationsMu.Unlock()
		state.Unfreeze()
	}
	return h
}

// onCreateOrganization is called when an organization is created. Its transport
// is not created until the organization needs one.
func (h *HTTP) onCreateOrganization(n state.CreateOrganization) {
	h.organizationsMu.Lock()
	if _, ok := h.organizations[n.ID]; !ok {
		h.organizations[n.ID] = &organization{}
	}
	h.organizationsMu.Unlock()
}

// onDeleteOrganization is called when an organization is deleted. It is marked
// as deleted, so that the clients holding its transport stop making requests,
// and its transport is discarded, closing the connections it keeps idle.
func (h *HTTP) onDeleteOrganization(n state.DeleteOrganization) {
	h.organizationsMu.Lock()
	org, ok := h.organizations[n.ID]
	delete(h.organizations, n.ID)
	var transport *organizationTransport
	if ok {
		transport = org.transport
	}
	h.organizationsMu.Unlock()
	if !ok {
		return
	}
	org.deleted.Store(true)
	if transport == nil {
		return
	}
	// The transport of an organization is a clone of the base transport, with a
	// connection pool of its own, so closing its idle connections does not
	// affect the requests of the other organizations. It is not closed when it
	// is the base transport itself, which is shared, as when the Prometheus are
	// disabled.
	if t, ok := transport.base.(*http.Transport); ok && t != h.transport {
		t.CloseIdleConnections()
	}
}

// transportFor returns the transport to make the requests of the organization
// with the given ID with, attributing to it the bytes they send.
//
// The transport is created the first time the organization needs one, and it is
// then reused by all its clients, so that all the connections of an organization
// share the same connection pool.
//
// It panics if organizationID is empty: the clients whose requests are not made
// on behalf of an organization are created with [HTTP.PlainConnectorClient],
// which uses the base transport as it is. If the organization does not exist,
// the returned transport fails every request with [ErrNoOrganization], because
// the clients are created without an error to return.
func (h *HTTP) transportFor(organizationID string) http.RoundTripper {
	if organizationID == "" {
		panic("core/connectors/httpclient: empty organization ID")
	}
	h.organizationsMu.Lock()
	defer h.organizationsMu.Unlock()
	org, ok := h.organizations[organizationID]
	if !ok {
		if h.listening {
			return errorTransport{noOrganizationError(organizationID)}
		}
		org = &organization{}
		h.organizations[organizationID] = org
	}
	if org.transport == nil {
		// The transport of the organization is a clone of the base transport
		// dialing with a dial function of the organization, so that it keeps the
		// timeouts and the options of the base transport and the bytes its
		// connections send are attributed to the organization. When counting is
		// disabled there is nothing to attribute, and the base transport is used
		// as it is, shared with every other organization.
		base := http.RoundTripper(h.transport)
		if dialer.CountingEnabled() {
			t := h.transport.Clone()
			t.DialContext = dialer.DialWith(organizationID)(t.DialContext)
			base = t
		}
		org.transport = &organizationTransport{
			base:         base,
			organization: org,
			id:           organizationID,
		}
	}
	return org.transport
}

// organizationTransport is the transport of an organization: it makes the
// requests with the transport attributing the bytes they send to the
// organization, failing them once the organization has been deleted.
//
// A client keeps the transport it was created with, and it can live long enough
// to make requests after its organization has been deleted, so the transport
// checks that the organization still exists at every request, and not only when
// a connection is established.
type organizationTransport struct {
	base         http.RoundTripper // transport of the organization
	organization *organization     // organization the requests are made on behalf of
	id           string            // ID of the organization, for the error
}

func (t *organizationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.organization.deleted.Load() {
		return nil, noOrganizationError(t.id)
	}
	return t.base.RoundTrip(req)
}

// errorTransport is a transport failing every request with err. It is the
// transport of the clients of an organization that does not exist.
type errorTransport struct {
	err error
}

func (t errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

// noOrganizationError returns the error the requests made on behalf of the
// organization with the given ID fail with when it does not exist.
func noOrganizationError(organizationID string) error {
	return fmt.Errorf("core/connectors/httpclient: %w: %s", ErrNoOrganization, organizationID)
}

// ConnectionClient returns an HTTP client for the provided connection.
// If the connection supports OAuth, the client is capable of retrieving OAuth
// credentials from it. The client's rate limits and retry policy are inherited
// from the connector.
//
// The requests of the client are made on behalf of the organization of the
// connection, and they fail with [ErrNoOrganization] if the organization is
// deleted, even if the client was created before the deletion.
//
// ConnectionClient must be called only once per connection.
func (h *HTTP) ConnectionClient(connection *state.Connection) *Client {
	if h.state == nil {
		panic("core/connectors/httpclient: HTTP.ConnectionClient called while state is nil")
	}
	connector := connection.Connector()
	c := &Client{
		http:       h,
		connector:  connector.Code,
		connection: connection.ID,
		transport:  h.transportFor(connection.Organization().ID),
	}
	c.endpointGroups.mux = h.connectorMux(connector.Code, connector.EndpointGroups)
	c.endpointGroups.byPattern = endpointGroupByPattern(connector.EndpointGroups)
	return c
}

// ConnectorClient returns an HTTP client for the provided connection, with the
// provided OAuth client secret and access token. If the client does not need to
// support OAuth, clientSecret and accessToken can be left empty.
//
// Moreover, if the HTTP has no state and the client does not need to support
// OAuth, clientSecret and accessToken must be left empty; in this case,
// OAuth-related Client methods cannot be invoked, as their behavior may be
// undefined or cause a panic.
//
// A connector, unlike a connection, does not belong to an organization, but the
// requests it sends are made on behalf of one, like when it serves the UI of a
// connection that is being created. organizationID is the ID of that
// organization, and the bytes the returned client sends are attributed to it.
// It panics if it is empty: use [HTTP.PlainConnectorClient] when the connector
// is not used on behalf of an organization. If the organization does not exist,
// or it is deleted later, the requests of the client fail with
// [ErrNoOrganization].
func (h *HTTP) ConnectorClient(connector *state.Connector, organizationID, clientSecret, accessToken string) *Client {
	if h.state == nil && (clientSecret != "" || accessToken != "") {
		panic("when the HTTP state is nil, the clientSecret and accessToken cannot be provided")
	}
	c := &Client{
		http:         h,
		connector:    connector.Code,
		clientSecret: clientSecret,
		accessToken:  accessToken,
		transport:    h.transportFor(organizationID),
	}
	c.endpointGroups.mux = h.connectorMux(connector.Code, connector.EndpointGroups)
	c.endpointGroups.byPattern = endpointGroupByPattern(connector.EndpointGroups)
	return c
}

// PlainConnectorClient returns an HTTP client for the provided connector whose
// requests are not made on behalf of any organization: the bytes they send are
// not counted and they are sent with the base transport, shared with every
// other client that has no organization.
//
// Use it, in place of [HTTP.ConnectorClient], when there is no organization to
// attribute the bytes sent to, as for a connector under test. The returned
// client does not support OAuth.
func (h *HTTP) PlainConnectorClient(connector *state.Connector) *Client {
	c := &Client{
		http:      h,
		connector: connector.Code,
		transport: h.transport,
	}
	c.endpointGroups.mux = h.connectorMux(connector.Code, connector.EndpointGroups)
	c.endpointGroups.byPattern = endpointGroupByPattern(connector.EndpointGroups)
	return c
}

// GrantAuthorization grants an OAuth authorization code and returns the access
// token, the refresh token and the expiration time. redirectionURI is the
// redirection URI, and organizationID is the ID of the organization on behalf
// of which the authorization is granted.
func (h *HTTP) GrantAuthorization(ctx context.Context, connector *state.Connector, organizationID, code, redirectionURI string) (string, string, time.Time, error) {
	client := h.ConnectorClient(connector, organizationID, "", "")
	return client.retrieveOAuthToken(ctx, connector.OAuth, code, redirectionURI, "")
}

// SetTrace sets w as the output destination for tracing HTTP requests and
// responses in HTTP clients.
func (h *HTTP) SetTrace(w io.Writer) {
	h.trace = w
}

// connectorMux returns an http.ServeMux configured with the patterns of the
// connector with the provided name and endpoint groups.
// It panics if a connector's pattern is not valid.
func (h *HTTP) connectorMux(name string, groups []connectors.EndpointGroup) *http.ServeMux {
	h.mu.Lock()
	defer h.mu.Unlock()
	if mux, ok := h.muxes[name]; ok {
		return mux
	}
	defer func() {
		// Handles any panic raised by ServeMux.HandleFunc if an invalid pattern is provided.
		if r := recover(); r != nil {
			msg := r.(error).Error()
			msg = strings.TrimPrefix(msg, "http: ")
			panic(fmt.Errorf("core/connectors/httpclient: connector %s: %s", name, msg))
		}
	}()
	mux := http.NewServeMux()
	if groups == nil {
		mux.Handle("/", noOpHandle)
	} else {
		for _, group := range groups {
			if group.Patterns == nil {
				mux.Handle("/", noOpHandle)
				continue
			}
			for _, pattern := range group.Patterns {
				mux.Handle(pattern, noOpHandle)
			}
		}
	}
	h.muxes[name] = mux
	return mux
}

// endpointGroupByPattern returns a map associating each pattern from the
// provided endpoint groups to an endpointGroup initialized with the group's
// rate limits and retry policy.
func endpointGroupByPattern(groups []connectors.EndpointGroup) map[string]endpointGroup {
	byPattern := map[string]endpointGroup{}
	if groups == nil {
		byPattern["/"] = endpointGroup{
			rateLimiter: newRateLimiter(1, 1, 0),
		}
		return byPattern
	}
	for _, g := range groups {
		eg := endpointGroup{
			requireOAuth: g.RequireOAuth,
			rateLimiter:  newRateLimiter(g.RateLimit.RequestsPerSecond, g.RateLimit.Burst, g.RateLimit.MaxConcurrentRequests),
			retryPolicy:  g.RetryPolicy,
		}
		if g.Patterns == nil {
			byPattern["/"] = eg
			continue
		}
		for _, pattern := range g.Patterns {
			byPattern[pattern] = eg
		}
	}
	return byPattern
}
