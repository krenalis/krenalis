//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package website

// This package is the Website connector.

import (
	"context"
	_ "embed"

	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon []byte

// Make sure it implements the WebsiteConnection interface.
var _ connector.WebsiteConnection = &connection{}

func init() {
	connector.RegisterWebsite("Website", New)
}

type connection struct{}

// New returns a new website connection.
func New(context.Context, *connector.WebsiteConfig) (connector.WebsiteConnection, error) {
	return &connection{}, nil
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "Website",
		Type: connector.WebsiteType,
		Icon: icon,
	}
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, error) {
	return nil, ui.ErrEventNotExist
}
