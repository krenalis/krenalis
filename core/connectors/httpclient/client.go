//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package httpclient

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/state"
)

// backoffBase is the base for the default exponential backoff.
const backoffBase = 100 * time.Millisecond

// backoffJitterEnabled controls whether jitter is applied in backoff.
var backoffJitterEnabled = true

// netBackoff is the backoff strategy applied when a network error occurs.
var netBackoff = meergo.ExponentialStrategy(50 * time.Millisecond)

var errUnsupportedOAuth = errors.New("OAuth is not supported")

// Client implements the connector.HTTPClient interface.
type Client struct {
	http          *HTTP
	connection    int
	clientSecret  string
	accessToken   string
	backoffPolicy meergo.BackoffPolicy
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
		Workspace:    connection.Workspace().ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}

	err = c.http.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err = tx.Exec(ctx,
			"UPDATE accounts SET access_token = $1, refresh_token = $2, expires_in = $3 WHERE id = $4",
			n.AccessToken, n.RefreshToken, n.ExpiresIn, n.ID)
		if err != nil {
			return nil, err
		}
		return n, nil
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

// Do sends an HTTP request and returns the corresponding HTTP response.
//
// If the client supports OAuth, it adds the Authorization header automatically.
//
// It retries the request on network errors or when the client's backoff policy
// applies. A request is retried only if it is idempotent
// (see http.Transport for details), which is defined as:
//
//   - method is GET, HEAD, OPTIONS, or TRACE and Request.GetBody is set, or
//   - Request.Header contains an Idempotency-Key or X-Idempotency-Key key.
//
// An empty header value is considered idempotent but is not sent.
//
// It always closes the request body, even if an error occurs.
// It does not follow redirects.
func (c *Client) Do(req *http.Request) (*http.Response, error) {

	ctx := req.Context()

	retriable := isRetriable(req)

	retries := 0
	netRetries := false // indicates if the last retry was triggered by a network error.

	for {

		// Set the Authorization header if OAuth is supported.
		accessToken, err := c.AccessToken(req.Context())
		if err != nil {
			if err != errUnsupportedOAuth {
				return nil, err
			}
		} else {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}

		// Send the request.
		res, err := c.http.transport.RoundTrip(req)
		if err != nil {
			if retriable {
				// Wait before retrying.
				if !netRetries {
					retries = 0
					netRetries = true
				}
				wt, _ := netBackoff(nil, retries)
				select {
				case <-time.After(wt):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				retries++
				continue
			}
			return nil, err
		}

		if !retriable {
			return res, nil
		}
		if status := res.StatusCode; status == 200 || status == 201 {
			return res, nil
		}

		// Wait before retrying.
		if netRetries {
			retries = 0
			netRetries = false
		}
		wt, err := c.waitTime(res, retries)
		if err != nil {
			return res, nil
		}
		// Drain and close the response body.
		closed := make(chan struct{})
		go func() {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			close(closed)
		}()
		// Wait.
		select {
		case <-time.After(wt):
		case <-ctx.Done():
			_ = res.Body.Close()
			return nil, ctx.Err()
		}

		// Wait for the response body to close.
		select {
		case <-closed:
		case <-ctx.Done():
			_ = res.Body.Close()
			return nil, ctx.Err()
		}

		// Restore the request's body.
		if req.Body != nil && req.Body != http.NoBody {
			req = req.Clone(ctx)
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

		retries++

	}

}

// waitTime calculates the duration to wait before retrying a failed request,
// based on the backoff policy in c.backoffPolicy and the response's status
// code. If c.backoffPolicy is nil, it checks the Retry-After header for status
// code 429 (Too Many Requests); for status codes 500 (Internal Server Error)
// and 503 (Service Unavailable), it applies exponential backoff with an initial
// delay of 100ms. If the response status code does not warrant a retry, it
// returns the meergo.NoRetry error.
func (c *Client) waitTime(res *http.Response, retries int) (time.Duration, error) {
	var primaryStrategy, secondaryStrategy meergo.BackoffStrategy
	if c.backoffPolicy != nil {
		// Use the client's policy.
		var status string
		if len(res.Status) >= 3 {
			status = res.Status[:3]
		} else {
			status = strconv.Itoa(res.StatusCode)
		}
		for statuses, strategy := range c.backoffPolicy {
			if strings.Contains(statuses, status) {
				primaryStrategy = strategy
				break
			}
		}
	} else {
		// Use the default policy.
		switch res.StatusCode {
		case 429:
			primaryStrategy = meergo.RetryAfterStrategy()
			secondaryStrategy = meergo.ExponentialStrategy(backoffBase)
		case 500, 503, 502, 504:
			primaryStrategy = meergo.ExponentialStrategy(backoffBase)
		}
	}
	if primaryStrategy == nil {
		return 0, meergo.NoRetry
	}
	d, err := primaryStrategy(res, retries)
	if err == meergo.NoRetry && secondaryStrategy != nil {
		d, err = secondaryStrategy(res, retries)
	}
	if err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, nil
	}
	// Add a jitter to introduce variability to the delay.
	if backoffJitterEnabled {
		d += time.Duration(float64(d) * rand.Float64() * 0.5)
	}
	return d, nil
}

// isRetriable reports whether the given HTTP request is retriable.
func isRetriable(req *http.Request) bool {
	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return false
	}
	switch req.Method {
	case "", "GET", "HEAD", "OPTIONS", "TRACE":
		return true
	}
	if _, ok := req.Header["Idempotency-Key"]; ok {
		return true
	}
	if _, ok := req.Header["X-Idempotency-Key"]; ok {
		return true
	}
	return false
}
