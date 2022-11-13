//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
)

// ServerConfig represents the configuration of a server connection.
type ServerConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// ServerConnectionFunc represents functions that create new server
// connections.
type ServerConnectionFunc func(context.Context, *ServerConfig) (ServerConnection, error)

// ServerConnection is the interface implemented by server connections.
type ServerConnection interface {
	Connection
}
