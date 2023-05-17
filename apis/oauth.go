//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

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

// How OAuth works:
//
// 1) The UI calls the (*Connector).AuthCodeURL method with the redirect URL. This
//    method returns the URL to which the user should be redirected to grant authorization.
// 2) The UI redirects the user to the returned URL.
// 3) The user authorizes the application.
// 4) The provider redirects the user to the specified redirect URL.
// 5) If no error occurs, the UI receives the authorization code from the provider and
//    calls the (*Workspace).OAuthToken method. In return, it receives a string
//    that identifies the authorized resource.
// 6) The UI displays the connector settings interface.
// 7) The UI calls the (*Workspace).AddConnection method to add the new connection,
//    passing the string of the authorized resource as one of the arguments.

// freshAccessToken returns a fresh OAuth access token for the resource. If it
// has no access token, or it is expired, freshAccessToken fetches a fresh
// access token and returns it.
func freshAccessToken(db *postgres.DB, r *state.Resource) (string, error) {

	if r.AccessToken != "" {
		expired := time.Now().UTC().Add(15 * time.Minute).After(r.ExpiresIn)
		if !expired {
			return r.AccessToken, nil
		}
	}

	ctx := context.Background()
	accessToken, refreshToken, expiresIn, err := retrieveOAuthToken(ctx, r.Connector(), "", "", r.RefreshToken)
	if err != nil {
		return "", err
	}

	n := state.SetResourceNotification{
		ID:           r.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}

	err = db.Transaction(ctx, func(tx *postgres.Tx) error {
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

// retrieveOAuthToken retrieves an OAuth token and returns the access token,
// refresh token, and expiration time of the access token for the given
// connector.
//
// To retrieve an authorization code for the first time, both authorizationCode
// and redirectURI are required. To refresh the token, only the refreshToken
// is required.
func retrieveOAuthToken(ctx context.Context, connector *state.Connector, authorizationCode, redirectURI, refreshToken string) (string, string, time.Time, error) {

	v := url.Values{
		"client_id":     {connector.OAuth.ClientID},
		"client_secret": {connector.OAuth.ClientSecret},
	}
	if authorizationCode == "" {
		v.Set("grant_type", "refresh_token")
		v.Set("refresh_token", refreshToken)
	} else {
		v.Set("grant_type", "authorization_code")
		v.Set("code", authorizationCode)
		v.Set("redirect_uri", redirectURI)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", connector.OAuth.TokenURL, strings.NewReader(v.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return "", "", time.Time{}, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("cannot retrieve the refresh and access tokens from connector %d: %s", connector.ID, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return "", "", time.Time{}, fmt.Errorf("cannot retrieve the refresh and access tokens from connector %d: server responded with status %d", connector.ID, resp.StatusCode)
	}

	tokens := struct {
		AccessToken  string       `json:"access_token"`
		TokenType    string       `json:"token_type"` // TODO(carlo): validate the value
		ExpiresIn    *json.Number `json:"expires_in"`
		RefreshToken string       `json:"refresh_token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("cannot decode response from OAuth server of connector %d: %s", connector.ID, err)
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
	expiresIn := connector.OAuth.ExpiresIn
	if expiresIn == 0 {
		if tokens.ExpiresIn == nil {
			return "", "", time.Time{}, fmt.Errorf("the OAuth provider for connector %d did not returned expires_in", connector.ID)
		}
		s, _ := tokens.ExpiresIn.Int64()
		if s < 1 {
			return "", "", time.Time{}, fmt.Errorf("the OAuth provider for connector %d returned an invalid expires_in = %q", connector.ID, tokens.ExpiresIn)
		}
		if s > math.MaxInt32 {
			s = math.MaxInt32
		}
		expiresIn = time.Duration(s) * time.Second
	}

	return tokens.AccessToken, tokens.RefreshToken, expiration.Add(expiresIn), nil
}
