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
var icon = "<svg></svg>"

// Make sure it implements the WebsiteConnection interface.
var _ connector.WebsiteConnection = &connection{}

func init() {
	connector.RegisterWebsite(connector.Website{
		Name: "Website",
		Icon: icon,
		Open: open,
	})
}

type connection struct{}

// open opens a Website connection and returns it.
func open(context.Context, *connector.WebsiteConfig) (connector.WebsiteConnection, error) {
	return &connection{}, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {
	return nil, nil, ui.ErrEventNotExist
}
