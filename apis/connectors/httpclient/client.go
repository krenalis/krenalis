//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package httpclient

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"time"

	"chichi/apis/capture"
	"chichi/apis/errors"
	"chichi/apis/state"
)

var errUnsupportedOAuth = errors.New("OAuth is not supported")

// Client implements the connector.HTTPClient interface.
type Client struct {
	http         *HTTP
	connection   int
	clientSecret string
	accessToken  string
}

// AccessToken returns an OAuth access token.
func (c *Client) AccessToken(ctx context.Context) (string, error) {

	if c.connection == 0 {
		if c.accessToken == "" {
			return "", errUnsupportedOAuth
		}
		return c.accessToken, nil
	}

	connection, ok := c.http.state.Connection(c.connection)
	if !ok {
		return "", fmt.Errorf("connection %d does not exist anymore", c.connection)
	}
	connector := connection.Connector()
	if connector.OAuth == nil {
		return "", errUnsupportedOAuth
	}
	r, ok := connection.Resource()
	if !ok {
		return "", fmt.Errorf("connection %d does not have a resource", c.connection)
	}

	if r.AccessToken != "" {
		expired := time.Now().UTC().Add(15 * time.Minute).After(r.ExpiresIn)
		if !expired {
			return r.AccessToken, nil
		}
	}

	accessToken, refreshToken, expiresIn, err := c.http.retrieveOAuthToken(ctx, connector.OAuth, "", "", r.RefreshToken)
	if err != nil {
		return "", err
	}

	n := state.SetResource{
		ID:           r.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}

	err = c.http.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err = tx.Exec(ctx,
			"UPDATE resources SET access_token = $1, refresh_token = $2, expires_in = $3 WHERE id = $4",
			n.AccessToken, n.RefreshToken, n.ExpiresIn, n.ID)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return "", err
	}

	return n.AccessToken, nil
}

// ClientSecret returns the OAuth client secret of the HTTP client.
func (c *Client) ClientSecret() (string, error) {
	if c.connection == 0 {
		if c.clientSecret == "" {
			return "", errUnsupportedOAuth
		}
		return c.clientSecret, nil
	}
	connection, ok := c.http.state.Connection(c.connection)
	if !ok {
		return "", fmt.Errorf("connection %d does not exist anymore", c.connection)
	}
	connector := connection.Connector()
	if connector.OAuth == nil {
		return "", errUnsupportedOAuth
	}
	return connector.OAuth.ClientSecret, nil
}

// Do sends an HTTP request with an Authorization header if required. It
// returns the response and ensures that the request body is closed, even in
// the case of errors. Redirects are not followed.
func (c *Client) Do(req *http.Request) (*http.Response, error) {

	// Close the body before returning.
	bodyClosed := false
	if req.Body != nil {
		defer func() {
			if !bodyClosed {
				_ = req.Body.Close()
			}
		}()
	}

	// Set the Authorization header if OAuth is supported.
	accessToken, err := c.AccessToken(req.Context())
	if err != nil {
		if err != errUnsupportedOAuth {
			return nil, err
		}
	} else {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	// Trace the request.
	var dump *bufio.Writer
	if c.http.trace != nil {
		dump = bufio.NewWriter(c.http.trace)
		_, _ = dump.WriteString("\nRequest:\n------\n")
		capture.Request(req, dump, true, true)
	}

	// Sent the request.
	bodyClosed = true
	res, err := c.http.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Trace the response.
	if c.http.trace != nil {
		dump.Reset(c.http.trace)
		_, _ = dump.WriteString("\n\n\nResponse:\n------\n")
		capture.Response(res, dump, true, true)
	}

	return res, nil
}
