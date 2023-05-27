//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package oauth provides support to grant, retrieve, and refresh access tokens.
//
// How OAuth works:
//
//  1. The UI calls the (*Connector).AuthCodeURL method with the redirect URL. This
//     method returns the URL to which the user should be redirected to grant authorization.
//  2. The UI redirects the user to the returned URL.
//  3. The user authorizes the application.
//  4. The provider redirects the user to the specified redirect URL.
//  5. If no error occurs, the UI receives the authorization code from the provider and
//     calls the (*Workspace).OAuthToken method. In return, it receives a string
//     that identifies the authorized resource.
//  6. The UI displays the connector settings interface.
//  7. The UI calls the (*Workspace).AddConnection method to add the new connection,
//     passing the string of the authorized resource as one of the arguments.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/state"
)

// OAuth implements the methods to grant, retrieve, and refresh access tokens.
type OAuth struct {
	db *postgres.DB
}

// New returns a new *OAuth value.
func New(db *postgres.DB) *OAuth {
	return &OAuth{db}
}

// AccessToken returns an access token for the resource r. If r does not have an
// access token or the existing token is expiring, it generates a new access
// token and returns it.
func (oauth *OAuth) AccessToken(ctx context.Context, r *state.Resource) (string, error) {

	if r.AccessToken != "" {
		expired := time.Now().UTC().Add(15 * time.Minute).After(r.ExpiresIn)
		if !expired {
			return r.AccessToken, nil
		}
	}

	c := r.Connector()
	accessToken, refreshToken, expiresIn, err := oauth.retrieveOAuthToken(ctx, c.OAuth, "", "", r.RefreshToken)
	if err != nil {
		return "", err
	}

	n := state.SetResourceNotification{
		ID:           r.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}

	err = oauth.db.Transaction(ctx, func(tx *postgres.Tx) error {
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

// AuthCodeURL returns a URL that directs to the consent page of the provider.
// This page requests explicit permissions for the required scopes. After that,
// the provider redirects to the URL specified by redirectURI.
// After acquiring the authorization code, call GrantAuthorizationCode.
func (oauth *OAuth) AuthCodeURL(auth *state.OAuth, redirectURI string) (string, error) {
	var b strings.Builder
	b.WriteString(auth.AuthURL)
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {auth.ClientID},
		"redirect_uri":  {redirectURI},
		"state":         {"state"},
	}
	if len(auth.Scopes) > 0 {
		v.Set("scope", strings.Join(auth.Scopes, " "))
	}
	if strings.Contains(auth.AuthURL, "?") {
		b.WriteByte('&')
	} else {
		b.WriteByte('?')
	}
	b.WriteString(v.Encode())
	return b.String(), nil
}

// GrantAuthorizationCode grants an authorization code and returns the access
// token, the refresh token and the expiration time. redirectURI is the redirect
// URL previously passed to AuthCodeURL.
func (oauth *OAuth) GrantAuthorizationCode(ctx context.Context, auth *state.OAuth, authorizationCode, redirectURI string) (string, string, time.Time, error) {
	return oauth.retrieveOAuthToken(ctx, auth, authorizationCode, redirectURI, "")
}

// retrieveOAuthToken retrieves an OAuth token and returns the access token,
// refresh token, and expiration time of the access token for the given
// connector.
//
// To retrieve an authorization code for the first time, both authorizationCode
// and redirectURI are required. To refresh the token, only the refreshToken
// is required.
func (oauth *OAuth) retrieveOAuthToken(ctx context.Context, auth *state.OAuth, authorizationCode, redirectURI, refreshToken string) (string, string, time.Time, error) {

	v := url.Values{
		"client_id":     {auth.ClientID},
		"client_secret": {auth.ClientSecret},
	}
	if authorizationCode == "" {
		v.Set("grant_type", "refresh_token")
		v.Set("refresh_token", refreshToken)
	} else {
		v.Set("grant_type", "authorization_code")
		v.Set("code", authorizationCode)
		v.Set("redirect_uri", redirectURI)
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
		AccessToken  string       `json:"access_token"`
		TokenType    string       `json:"token_type"` // TODO(carlo): validate the value
		ExpiresIn    *json.Number `json:"expires_in"`
		RefreshToken string       `json:"refresh_token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&tokens)
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
		s, _ := tokens.ExpiresIn.Int64()
		if s < 1 {
			return "", "", time.Time{}, fmt.Errorf("the OAuth provider for %s returned an invalid expires_in = %q", auth.TokenURL, tokens.ExpiresIn)
		}
		expiresIn = int32(s)
		if s > math.MaxInt32 {
			expiresIn = math.MaxInt32
		}
	}
	expiration = expiration.Add(time.Duration(expiresIn) * time.Second)

	return tokens.AccessToken, tokens.RefreshToken, expiration, nil
}
