//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package server

import (
	"net/http"
	"strconv"

	"chichi/apis"
	"chichi/apis/errors"
)

type connector struct {
	*apisServer
}

// AuthCodeURL returns a URL that directs to the consent page of an OAuth 2.0
// provider
func (connector connector) AuthCodeURL(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := connector.credentials(r); err != nil {
		return nil, err
	}
	c, err := connector.connector(r)
	if err != nil {
		return nil, err
	}
	redirectURI := r.URL.Query().Get("redirecturi")
	return c.AuthCodeURL(redirectURI)
}

func (connector connector) connector(r *http.Request) (*apis.Connector, error) {
	v := r.PathValue("connector")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	return connector.apis.Connector(r.Context(), id)
}
