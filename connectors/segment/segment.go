//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package segment

import (
	"context"
	"net/http"

	"chichi/connectors"
)

type Connector struct {
	ClientSecret string
	Context      context.Context
}

func init() {
	connectors.RegisterConnector("Segment", (*Connector)(nil))
}

// Groups returns the groups starting from the given cursor.
func (c *Connector) Groups(account, cursor string) error {
	return nil
}

// Properties returns all user and group properties.
func (c *Connector) Properties(account string) ([]connectors.Property, []connectors.Property, error) {
	return nil, nil, nil
}

// ServeWebhook serves a webhook request.
func (c *Connector) ServeWebhook(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// SetUsers sets the given users.
func (c *Connector) SetUsers(token string, users []connectors.User) error {
	return nil
}

// Users returns the users starting from the given cursor.
func (c *Connector) Users(account, cursor string) error {
	return nil
}
