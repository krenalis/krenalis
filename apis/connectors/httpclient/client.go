//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/apis/capture"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/state"
)

// backoffBase is the base for the default exponential backoff.
const backoffBase = 100 * time.Millisecond

var errUnsupportedOAuth = errors.New("OAuth is not supported")

// Client implements the connector.HTTPClient interface.
type Client struct {
	http         *HTTP
	connection   int
	clientSecret string
	accessToken  string
	backoff      map[string]meergo.Backoff
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
	a, ok := connection.Account()
	if !ok {
		return "", fmt.Errorf("connection %d does not have an OAuth account", c.connection)
	}

	if a.AccessToken != "" {
		expired := time.Now().UTC().Add(15 * time.Minute).After(a.ExpiresIn)
		if !expired {
			return a.AccessToken, nil
		}
	}

	accessToken, refreshToken, expiresIn, err := c.http.retrieveOAuthToken(ctx, connector.OAuth, "", "", a.RefreshToken)
	if err != nil {
		return "", err
	}

	n := state.SetAccount{
		ID:           a.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}

	err = c.http.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err = tx.Exec(ctx,
			"UPDATE accounts SET access_token = $1, refresh_token = $2, expires_in = $3 WHERE id = $4",
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

	var body io.Reader
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(b)
		req.Body = io.NopCloser(body)
	}

	for i := 0; ; i++ {

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

		if req.Method != "GET" && req.Method != "HEAD" {
			return res, nil
		}
		if status := res.StatusCode; 200 <= status && status < 300 {
			return res, nil
		}

		wt, err := c.waitTime(res, i)
		if err != nil {
			return res, nil
		}
		select {
		case <-time.After(wt):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}

	}

}

// waitTime calculates the duration to wait before retrying a failed request,
// based on the backoff policy in c.backoff and the response's status code.
// If c.backoff is nil, it checks the Retry-After header for status codes
// 429 (Too Many Requests) and 503 (Service Unavailable); for status code 500
// (Internal Server Error), it applies exponential backoff with an initial
// delay of 100ms. If the response status code does not warrant a retry, it
// returns the meergo.NoRetry error.
func (c *Client) waitTime(res *http.Response, retries int) (time.Duration, error) {
	var primaryBackoff, secondaryBackoff meergo.Backoff
	if c.backoff != nil {
		// Use the client's policy.
		var status string
		if len(res.Status) >= 3 {
			status = res.Status[:3]
		} else {
			status = strconv.Itoa(res.StatusCode)
		}
		for statuses, backoff := range c.backoff {
			if strings.Contains(statuses, status) {
				primaryBackoff = backoff
				break
			}
		}
	} else {
		// Use the default policy.
		switch res.StatusCode {
		case 429:
			primaryBackoff = meergo.RetryAfterBackoff()
			secondaryBackoff = meergo.ExponentialBackoff(backoffBase)
		case 500, 503, 502, 504:
			primaryBackoff = meergo.ExponentialBackoff(backoffBase)
		}
	}
	if primaryBackoff == nil {
		return 0, meergo.NoRetry
	}
	d, err := primaryBackoff(res, retries)
	if err == meergo.NoRetry && secondaryBackoff != nil {
		d, err = secondaryBackoff(res, retries)
	}
	if err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, nil
	}
	return d, nil
}
