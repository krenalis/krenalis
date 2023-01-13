//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
)

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

	connector := r.Connector()

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", connector.OAuth.ClientID)
	data.Set("client_secret", connector.OAuth.ClientSecret)
	data.Set("redirect_uri", "https://localhost:9090/admin/oauth/authorize")
	data.Set("refresh_token", r.RefreshToken)

	req, err := http.NewRequest("POST", connector.OAuth.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusBadRequest {
			errData := struct {
				status string
			}{}
			err = json.NewDecoder(res.Body).Decode(&errData)
			if err != nil {
				return "", err
			}
			// TODO(@Andrea): check the status returned by services different
			// from Hubspot.
			if errData.status == "BAD_REFRESH_TOKEN" {
				return "", errors.Unprocessable(InvalidRefreshToken, "OAuth refresh token of connector %d is not valid", connector.ID)
			}
		}
		return "", fmt.Errorf("unexpected status %d returned by connector while trying to get a new access token via refresh token", res.StatusCode)
	}

	oAuth := struct {
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
	}{}
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&oAuth)
	if err != nil {
		return "", err
	}

	// Convert expires_in into a timestamp.
	// TODO(marco): ExpiresIn should be relative to response time?
	expiresIn := time.Now().UTC().Add(time.Duration(oAuth.ExpiresIn) * time.Second)

	n := state.SetResourceNotification{
		ID:           r.ID,
		AccessToken:  oAuth.AccessToken,
		RefreshToken: oAuth.RefreshToken,
		ExpiresIn:    expiresIn,
	}

	err = db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec(
			"UPDATE resources SET access_token = $1, refresh_token = $2, expires_in = $3 WHERE id = $4",
			n.AccessToken, n.RefreshToken, n.ExpiresIn, n.ID)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})
	if err != nil {
		return "", err
	}

	return n.AccessToken, nil
}
