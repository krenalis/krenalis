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
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/json"
)

// HTTP allows creating HTTP clients for connections and enables granting,
// retrieving, and refreshing OAuth access tokens.
type HTTP struct {
	state     *state.State
	transport http.RoundTripper
	trace     io.Writer
}

// New returns an HTTP instance given the state and the transport to use for
// HTTP connections.
func New(state *state.State, transport http.RoundTripper) *HTTP {
	return &HTTP{
		state:     state,
		transport: transport,
	}
}

// Client returns an HTTP client with the provided OAuth client secret and
// access token. If the client does not need to support OAuth, clientSecret
// and accessToken can be left empty.
//
// backoffPolicy is the backoff policy. If it is nil, the client will use a
// default policy.
func (h *HTTP) Client(clientSecret, accessToken string, backoffPolicy meergo.BackoffPolicy) *Client {
	return &Client{
		http:          h,
		clientSecret:  clientSecret,
		accessToken:   accessToken,
		backoffPolicy: backoffPolicy,
	}
}

// ConnectionClient returns an HTTP client capable of retrieving OAuth
// credentials from the provided connection if it supports OAuth. The client's
// backoff policy is the connector's policy.
func (h *HTTP) ConnectionClient(connection int) *Client {
	c, _ := h.state.Connection(connection)
	return &Client{
		http:          h,
		connection:    connection,
		backoffPolicy: c.Connector().BackoffPolicy,
	}
}

// GrantAuthorization grants an OAuth authorization code and returns the access
// token, the refresh token and the expiration time. redirectionURI is the
// redirection URI.
func (h *HTTP) GrantAuthorization(ctx context.Context, auth *state.OAuth, code, redirectionURI string) (string, string, time.Time, error) {
	return h.retrieveOAuthToken(ctx, auth, code, redirectionURI, "")
}

// SetTrace sets w as the output destination for tracing HTTP request and
// responses in HTTP clients.
func (h *HTTP) SetTrace(w io.Writer) {
	h.trace = w
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
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return "", "", time.Time{}, err
	}

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
			return "", "", time.Time{}, fmt.Errorf("the OAuth provider for %s did not returned expires_in", auth.TokenURL)
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
