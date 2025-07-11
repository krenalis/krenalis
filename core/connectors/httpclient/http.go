//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package httpclient provides an HTTP client with OAuth support for
// connections.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/json"
)

func noOpHandler(http.ResponseWriter, *http.Request) {}

// HTTP allows creating HTTP clients for connections and enables granting,
// retrieving, and refreshing OAuth access tokens.
type HTTP struct {

	// state is nil if the HTTP client was instantiated without providing a
	// state; in that case, methods related to OAuth cannot be used, as their
	// behavior may be unexpected or cause a panic.
	state *state.State

	transport http.RoundTripper
	trace     io.Writer

	// muxes maps each connector name to the corresponding ServeMux handling its rate limits.
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
		connector:  connector.Name,
		connection: connection.ID,
	}
	c.endpointGroups.mux = h.connectorMux(connector.Name, connector.EndpointGroups)
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
		connector:    connector.Name,
		clientSecret: clientSecret,
		accessToken:  accessToken,
	}
	c.endpointGroups.mux = h.connectorMux(connector.Name, connector.EndpointGroups)
	c.endpointGroups.byPattern = endpointGroupByPattern(connector.EndpointGroups)
	return c
}

// GrantAuthorization grants an OAuth authorization code and returns the access
// token, the refresh token and the expiration time. redirectionURI is the
// redirection URI.
func (h *HTTP) GrantAuthorization(ctx context.Context, auth *state.OAuth, code, redirectionURI string) (string, string, time.Time, error) {
	return h.retrieveOAuthToken(ctx, auth, code, redirectionURI, "")
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
		mux.HandleFunc("/", noOpHandler)
	} else {
		for _, group := range groups {
			for _, pattern := range group.Patterns {
				mux.HandleFunc(pattern, noOpHandler)
			}
		}
	}
	h.muxes[name] = mux
	return mux
}

// retrieveOAuthToken retrieves an OAuth token and returns the access token,
// refresh token, and expiration time of the access token for the provided
// connector.
//
// To retrieve an authorization code for the first time, both code and
// redirectionURI are required. To refresh the token, only the refreshToken is
// required.
func (h *HTTP) retrieveOAuthToken(ctx context.Context, auth *state.OAuth, code, redirectionURI, refreshToken string) (string, string, time.Time, error) {

	v := url.Values{
		"client_id":     {auth.ClientID},
		"client_secret": {auth.ClientSecret},
	}
	if code == "" {
		v.Set("grant_type", "refresh_token")
		v.Set("refresh_token", refreshToken)
	} else {
		v.Set("grant_type", "authorization_code")
		v.Set("code", code)
		v.Set("redirect_uri", redirectionURI)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", auth.TokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return "", "", time.Time{}, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("cannot retrieve the refresh and access tokens from %s: %s", auth.TokenURL, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return "", "", time.Time{}, fmt.Errorf("cannot retrieve the refresh and access tokens from %s: server responded with status %d", auth.TokenURL, resp.StatusCode)
	}

	tokens := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"` // TODO(carlo): validate the value
		ExpiresIn    *int   `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}{}
	err = json.Decode(resp.Body, &tokens)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("cannot decode response from %s: %s", auth.TokenURL, err)
	}

	// TODO(carlo): compute the token type to use

	var expiration time.Time
	if date := resp.Header.Get("date"); date != "" {
		expiration, _ = time.Parse(time.RFC1123, date)
	}
	expiration = expiration.UTC()
	if now := time.Now().UTC(); expiration.IsZero() || expiration.After(now.Add(time.Hour)) {
		expiration = now
	}
	expiresIn := auth.ExpiresIn
	if expiresIn <= 0 {
		if tokens.ExpiresIn == nil {
			return "", "", time.Time{}, fmt.Errorf("the OAuth provider for %s did not return expires_in", auth.TokenURL)
		}
		s := *tokens.ExpiresIn
		if s < 1 {
			return "", "", time.Time{}, fmt.Errorf("the OAuth provider for %s returned an invalid expires_in = %v", auth.TokenURL, tokens.ExpiresIn)
		}
		expiresIn = int32(s)
		if s > math.MaxInt32 {
			expiresIn = math.MaxInt32
		}
	}
	expiration = expiration.Add(time.Duration(expiresIn) * time.Second)

	return tokens.AccessToken, tokens.RefreshToken, expiration, nil
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
			rateLimiter: newRateLimiter(g.RateLimit.RequestsPerSecond, g.RateLimit.Burst, g.RateLimit.MaxConcurrentRequests),
			retryPolicy: g.RetryPolicy,
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
