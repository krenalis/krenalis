//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package server implements the Server connector.
package server

import (
	_ "embed"

	"chichi/connector"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterServer(connector.Server{
		Name:              "Server",
		SourceDescription: "collect events, and import users and groups from a server",
		Icon:              icon,
	}, open)
}

// open opens a Server connection and returns it.
func open(*connector.ServerConfig) (*connection, error) {
	return &connection{}, nil
}

type connection struct{}
