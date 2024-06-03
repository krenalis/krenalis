//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"net/http"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/apis/errors"
)

type connector struct {
	*apisServer
}

// AuthCodeURL returns a URL that directs to the consent page of an OAuth 2.0
// provider.
func (connector connector) AuthCodeURL(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := connector.credentials(r); err != nil {
		return nil, err
	}
	c, err := connector.connector(r)
	if err != nil {
		return nil, err
	}
	var role apis.Role
	switch r.URL.Query().Get("role") {
	case "Source":
		role = apis.Source
	case "Destination":
		role = apis.Destination
	default:
		return nil, errors.BadRequest("unexpected connection role '%s'", role)
	}
	redirectURI := r.URL.Query().Get("redirecturi")
	authCodeURL, err := c.AuthCodeURL(role, redirectURI)
	if err != nil {
		return nil, err
	}
	return map[string]any{"url": authCodeURL}, nil
}

func (connector connector) connector(r *http.Request) (*apis.Connector, error) {
	return connector.apis.Connector(r.Context(), r.PathValue("connector"))
}
