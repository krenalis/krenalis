// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/tools/errors"
)

type connector struct {
	*apisServer
}

// AuthURL returns a URL that directs to the consent page of an OAuth 2.0
// provider.
func (connector connector) AuthURL(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := connector.authenticateRequest(r); err != nil {
		return nil, err
	}
	q := r.URL.Query()
	c, err := connector.core.Connector(q.Get("connector"))
	if err != nil {
		return nil, err
	}
	var role core.Role
	switch q.Get("role") {
	case "Source":
		role = core.Source
	case "Destination":
		role = core.Destination
	default:
		return nil, errors.BadRequest("unexpected connection role '%s'", role)
	}
	redirectURI := q.Get("redirectURI")
	authURL, err := c.AuthURL(role, redirectURI)
	if err != nil {
		return nil, err
	}
	return map[string]any{"authUrl": authURL}, nil
}
