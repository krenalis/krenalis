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
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/json"
)

type contextKey byte

// RateLimiterPatternContextKey is the key used to propagate the rate limiter
// pattern through the request context. If the request context contains this
// key, the HTTP client checks whether the associated pattern matches the one
// used by the rate limiter for that request. If it doesn't match, the client
// calls the Set function to inform the caller about the actual pattern in use.
//
// This allows the caller to later call the HTTP client's Wait method with the
// correct pattern, determining how long to wait before sending a request to
// avoid being throttled.
const RateLimiterPatternContextKey contextKey = 0

// RateLimiterPatternContextValue is the type of the value associated with the
// RateLimiterPatternContextKey.
type RateLimiterPatternContextValue struct {
	Pattern string               // latest known rate limiter pattern
	Set     func(pattern string) // function to update the pattern
}

// backoffBase is the base for the default exponential backoff.
const backoffBase = 100 * time.Millisecond

// backoffJitterEnabled controls whether jitter is applied in backoff.
var backoffJitterEnabled = true

// netBackoff is the retry strategy applied when a network error occurs.
var netBackoff = meergo.ExponentialStrategy(meergo.NetFailure, 50*time.Millisecond)

var errUnsupportedOAuth = errors.New("OAuth is not supported")

// endpointGroup represents an endpoint group with its rate limiter and retry
// policy.
type endpointGroup struct {
	requireOAuth bool               // require OAuth
	rateLimiter  *rateLimiter       // rate limiter
	retryPolicy  meergo.RetryPolicy // retry policy
}

// Client implements the connector.HTTPClient interface.
type Client struct {
	http           *HTTP
	connector      string // connector name
	connection     int    // connection identifier; it is 0 if the client is not relative to a connection
	clientSecret   string // client secret; only if connection == 0
	accessToken    string // access token; only if connection == 0
	endpointGroups struct {
		mux       *http.ServeMux
		byPattern map[string]endpointGroup // endpoint group by pattern
	}
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

	accessToken, refreshToken, expiresIn, err := c.retrieveOAuthToken(ctx, connector.OAuth, "", "", a.RefreshToken)
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
// It retries the request on network errors or when the client's retry policy
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
	return c.do(req, false)
}

// GetBodyBuffer returns a BodyBuffer, initialised for enc.
func (c *Client) GetBodyBuffer(enc meergo.ContentEncoding) *meergo.BodyBuffer {
	return meergo.GetBodyBuffer(enc, 1024) // TODO(marco): estimate the size of the buffer
}

func (c *Client) do(req *http.Request, isRetriveOAuthToken bool) (*http.Response, error) {

	ctx := req.Context()

	_, pattern := c.endpointGroups.mux.Handler(req)
	if pattern == "" {
		return nil, fmt.Errorf(`connector %s attempted to call the '%s %s%s' endpoint, but there is no endpoint group that matches this request`, c.connector, req.Method, req.Host, req.URL.Path)
	}
	endpointGroup := c.endpointGroups.byPattern[pattern]
	limiter := endpointGroup.rateLimiter
	policy := endpointGroup.retryPolicy

	if !isRetriveOAuthToken {
		// Check if the request context contains a rate limiter pattern value.
		// If the stored pattern differs from the one currently used, update it
		// so the caller can align future requests accordingly.
		if v := ctx.Value(RateLimiterPatternContextKey); v != nil {
			v, ok := v.(RateLimiterPatternContextValue)
			if !ok {
				return nil, errors.New(`context key "RateLimiterPattern" must have a value of type RateLimiterPatternContextValue`)
			}
			if v.Pattern != pattern {
				v.Set(pattern)
			}
		}
	}

	hasBody := req.Body != nil && req.Body != http.NoBody
	retriable := isBodyRetriable(req) && isIdempotent(req)

	retries := 0
	netRetries := false // indicates if the last retry was triggered by a network error.

	// Send the request.
	for {

		// Add Authorization header.
		var accessToken string
		if endpointGroup.requireOAuth && !isRetriveOAuthToken {
			var err error
			accessToken, err = c.AccessToken(ctx)
			if err != nil {
				if err != errUnsupportedOAuth {
					return nil, err
				}
			} else {
				req.Header.Set("Authorization", "Bearer "+accessToken)
			}
		}

		if err := limiter.Wait(ctx); err != nil {
			return nil, err
		}

		// Send the request.
		start := time.Now()
		res, err := c.http.transport.RoundTrip(req)
		duration := time.Since(start)
		if err != nil {
			limiter.OnFailure(duration, meergo.NetFailure, 0)
			if !retriable {
				return res, err
			}
			if !netRetries {
				retries = 0
				netRetries = true
			}
			_, waitTime := netBackoff(nil, retries)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
			retries++
			continue
		}

		if status := res.StatusCode; status == 200 || status == 201 || status == 204 {
			limiter.OnSuccess(duration)
			return res, nil
		}

		reason, waitTime := retryStrategy(policy, res, retries)
		limiter.OnFailure(duration, reason, waitTime)

		if netRetries {
			retries = 0
			netRetries = false
		}

		switch reason {
		case meergo.PermanentFailure:
			return res, nil
		case meergo.NetFailure:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
		case meergo.Unauthorized:
			if isRetriveOAuthToken {
				return res, nil
			}
			if accessToken == "" {
				return res, nil
			}
			// For unauthorized requests to OAuth apps, we assume by default that
			// the request was not consumed, so if the body is retriable, it will be retried.
			if !isBodyRetriable(req) {
				return res, nil
			}
		}

		if hasBody {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

	}
}

// WaitTime estimates how long a Do call would be throttled by the rate limiter
// for the given pattern. Returns zero if the call can proceed immediately.
// Returns an error if there is no rate limiter for the given pattern.
func (c *Client) WaitTime(pattern string) (time.Duration, error) {
	group, ok := c.endpointGroups.byPattern[pattern]
	if !ok {
		return 0, fmt.Errorf("endpoint group with pattern %q does not exist", pattern)
	}
	return group.rateLimiter.WaitTime(), nil
}

// isBodyRetriable reports whether the request body can be retried.
func isBodyRetriable(req *http.Request) bool {
	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return false
	}
	return true
}

// isRetriable reports whether the given HTTP request is idempotent.
func isIdempotent(req *http.Request) bool {
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

// retrieveOAuthToken retrieves an OAuth token and returns the access token,
// refresh token, and expiration time of the access token for the provided
// connector.
//
// To retrieve an authorization code for the first time, both code and
// redirectionURI are required. To refresh the token, only the refreshToken is
// required.
func (c *Client) retrieveOAuthToken(ctx context.Context, auth *state.OAuth, code, redirectionURI, refreshToken string) (string, string, time.Time, error) {

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

	resp, err := c.do(req, true)
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

// retryStrategy determines the failure reason and how long to wait before
// retrying a failed request, based on the provided retry policy and the
// response status code.
//
// If policy is nil, it checks the Retry-After header for status code 429
// (Too Many Requests). For status codes 500 (Internal Server Error) and 503
// (Service Unavailable), it uses exponential backoff starting at 100ms.
//
// If the status code is not eligible for a retry, it returns PermanentFailure.
func retryStrategy(policy meergo.RetryPolicy, res *http.Response, retries int) (meergo.FailureReason, time.Duration) {
	var primaryStrategy, secondaryStrategy meergo.RetryStrategy
	if policy != nil {
		// Use the client's policy.
		var status string
		if len(res.Status) >= 3 {
			status = res.Status[:3]
		} else {
			status = strconv.Itoa(res.StatusCode)
		}
		for statuses, strategy := range policy {
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
			secondaryStrategy = meergo.ExponentialStrategy(meergo.Slowdown, backoffBase)
		case 500, 503, 502, 504:
			primaryStrategy = meergo.ExponentialStrategy(meergo.NetFailure, backoffBase)
		}
	}
	if primaryStrategy == nil {
		if res.StatusCode == 401 {
			return meergo.Unauthorized, 0
		}
		return meergo.PermanentFailure, 0
	}
	reason, wt := primaryStrategy(res, retries)
	if reason == meergo.PermanentFailure && secondaryStrategy != nil {
		reason, wt = secondaryStrategy(res, retries)
	}
	if reason == meergo.PermanentFailure {
		return meergo.PermanentFailure, 0
	}
	if wt <= 0 {
		return reason, 0
	}
	// Add a jitter to introduce variability to the delay.
	if backoffJitterEnabled {
		wt += time.Duration(float64(wt) * rand.Float64() * 0.5)
	}
	return reason, wt
}
