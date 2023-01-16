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

// Server represents a server connector.
type Server struct {
	Name string
	Icon string // icon in SVG format
	Open OpenServerFunc
}

// ServerConfig represents the configuration of a server connection.
type ServerConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenServerFunc represents functions that open server connections.
type OpenServerFunc func(context.Context, *ServerConfig) (ServerConnection, error)

// ServerConnection is the interface implemented by server connections.
type ServerConnection interface {
	Connection
}
