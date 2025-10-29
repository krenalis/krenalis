// Copyright 2025 Open2b. All rights reserved.
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

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/internal/state"
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

	transport http.RoundTripper
	trace     io.Writer

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
func New(state *state.State, transport http.RoundTripper) *HTTP {
	h := &HTTP{
		state:     state,
		transport: transport,
	}
	h.muxes = map[string]*http.ServeMux{}
	return h
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
func (h *HTTP) ConnectorClient(connector *state.Connector, clientSecret, accessToken string) *Client {
	if h.state == nil && (clientSecret != "" || accessToken != "") {
		panic("when the HTTP state is nil, the clientSecret and accessToken cannot be provided")
	}
	c := &Client{
		http:         h,
		connector:    connector.Code,
		clientSecret: clientSecret,
		accessToken:  accessToken,
	}
	c.endpointGroups.mux = h.connectorMux(connector.Code, connector.EndpointGroups)
	c.endpointGroups.byPattern = endpointGroupByPattern(connector.EndpointGroups)
	return c
}

// GrantAuthorization grants an OAuth authorization code and returns the access
// token, the refresh token and the expiration time. redirectionURI is the
// redirection URI.
func (h *HTTP) GrantAuthorization(ctx context.Context, connector *state.Connector, code, redirectionURI string) (string, string, time.Time, error) {
	client := h.ConnectorClient(connector, "", "")
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
func (h *HTTP) connectorMux(name string, groups []meergo.EndpointGroup) *http.ServeMux {
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
func endpointGroupByPattern(groups []meergo.EndpointGroup) map[string]endpointGroup {
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
