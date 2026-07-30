// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package httpclient provides an HTTP client with OAuth support for
// connections.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/dialer"
	"github.com/krenalis/krenalis/core/internal/state"
)

type noOpHandler struct{}

func (h noOpHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

var noOpHandle = noOpHandler{}

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
	trace     io.Writer

	// organizations holds the transport of each organization the requests can be
	// made on behalf of, by ID. The transport is created the first time the
	// organization needs one, so it is nil until then. An organization is added
	// when it is created and it is removed when it is deleted, so that its
	// transport is discarded with it.
	//
	// An organization that does not exist has no entry here: making requests on
	// its behalf is not an error, there is simply nothing to attribute the bytes
	// they send to, and they are made with the base transport.
	//
	// It is nil when the organizations are not known, that is when there is no
	// state to follow or the bytes sent are not counted at all, so that no
	// transport is cloned for anyone.
	organizationsMu sync.Mutex                 // protects organizations
	organizations   map[string]*http.Transport // by organization ID; protected by organizationsMu

	// muxes maps each connector code to the corresponding ServeMux handling its rate limits.
	mu    sync.Mutex                // protect muxes
	muxes map[string]*http.ServeMux // nil if state is nil; protected by mu
}

// New returns an HTTP instance given the state and the transport to use for
// HTTP connections.
//
// It is possible to provide a nil state; in that case the returned HTTP client
// will be restricted and will not allow invocation of OAuth-related methods, as
// their behavior may be unexpected or may cause a panic.
func New(state *state.State, transport *http.Transport) *HTTP {
	h := &HTTP{
		state:     state,
		transport: transport,
	}
	h.muxes = map[string]*http.ServeMux{}
	if state != nil {
		state.Freeze()
		state.AddListener(h.onCreateOrganization)
		state.AddListener(h.onDeleteOrganization)
		h.organizationsMu.Lock()
		h.organizations = map[string]*http.Transport{}
		for _, org := range state.Organizations() {
			h.organizations[org.ID] = nil
		}
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
		h.organizations[n.ID] = nil
	}
	h.organizationsMu.Unlock()
}

// onDeleteOrganization is called when an organization is deleted. Its transport
// is discarded, closing the connections it keeps idle, so that the transports do
// not accumulate for the whole life of the process.
//
// The clients created before the deletion keep the transport they were created
// with, and they can still make requests with it, as the connections a deleted
// organization has already dialed can still be written to: the bytes they send
// are added to a counter that is no longer collected, see [dialer.EnableCounting].
func (h *HTTP) onDeleteOrganization(n state.DeleteOrganization) {
	h.organizationsMu.Lock()
	transport := h.organizations[n.ID]
	delete(h.organizations, n.ID)
	h.organizationsMu.Unlock()
	// The transport is nil when the organization does not exist, or when it has
	// never needed one, and there is then nothing to discard. It is a clone of
	// the base transport, with a connection pool of its own, so closing its idle
	// connections does not affect the requests of the other organizations.
	if transport != nil {
		transport.CloseIdleConnections()
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
// which uses the base transport as it is. The base transport is returned, as it
// is, when the organization does not exist, so that the requests are made all
// the same, counting nothing.
func (h *HTTP) transportFor(organization string) http.RoundTripper {
	if organization == "" {
		panic("core/connectors/httpclient: empty organization ID")
	}
	h.organizationsMu.Lock()
	defer h.organizationsMu.Unlock()
	transport, ok := h.organizations[organization]
	if !ok {
		// The organization does not exist, or the organizations are not known:
		// there is nothing to attribute the bytes sent to, and the requests are
		// made with the base transport.
		return h.transport
	}
	if transport == nil {
		// The transport of the organization is a clone of the base transport
		// dialing with a dial function of the organization, so that it keeps the
		// timeouts and the options of the base transport and the bytes its
		// connections send are attributed to the organization.
		transport = h.transport.Clone()
		transport.DialContext = dialer.DialWith(organization)(transport.DialContext)
		h.organizations[organization] = transport
	}
	return transport
}

// ConnectionClient returns an HTTP client for the provided connection.
// If the connection supports OAuth, the client is capable of retrieving OAuth
// credentials from it. The client's rate limits and retry policy are inherited
// from the connector.
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
// the requests are made all the same and the bytes they send are counted for no
// one.
func (h *HTTP) ConnectorClient(connector *state.Connector, organization, clientSecret, accessToken string) *Client {
	if h.state == nil && (clientSecret != "" || accessToken != "") {
		panic("when the HTTP state is nil, the clientSecret and accessToken cannot be provided")
	}
	c := &Client{
		http:         h,
		connector:    connector.Code,
		clientSecret: clientSecret,
		accessToken:  accessToken,
		transport:    h.transportFor(organization),
	}
	c.endpointGroups.mux = h.connectorMux(connector.Code, connector.EndpointGroups)
	c.endpointGroups.byPattern = endpointGroupByPattern(connector.EndpointGroups)
	return c
}

// PlainConnectorClient returns an HTTP client for the provided connector whose
// requests are not made on behalf of any organization. This is useful in test
// scenarios.
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
// redirection URI, and organization is the ID of the organization on behalf
// of which the authorization is granted.
func (h *HTTP) GrantAuthorization(ctx context.Context, connector *state.Connector, organization, code, redirectionURI string) (string, string, time.Time, error) {
	client := h.ConnectorClient(connector, organization, "", "")
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
