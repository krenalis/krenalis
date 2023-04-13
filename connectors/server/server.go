//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package server

// This package is the Server connector.

import (
	"context"
	_ "embed"

	"chichi/connector"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the ServerConnection interface.
var _ connector.ServerConnection = &connection{}

func init() {
	connector.RegisterServer(connector.Server{
		Name:              "Server",
		SourceDescription: "receive events from a server",
		Icon:              icon,
		Open:              open,
	})
}

type connection struct{}

// open opens a Server connection and returns it.
func open(context.Context, *connector.ServerConfig) (connector.ServerConnection, error) {
	return &connection{}, nil
}
