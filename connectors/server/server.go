//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package server implements the .Net, Go, Java, Node.js, and Python connectors.
package server

import (
	_ "embed"

	"github.com/meergo/meergo"
)

// Connector icon.
var iconDotNet = "<svg></svg>"

// Connector icon.
var iconGo = "<svg></svg>"

// Connector icon.
var iconJava = "<svg></svg>"

// Connector icon.
var iconNode = "<svg></svg>"

// Connector icon.
var iconPython = "<svg></svg>"

func init() {
	servers := []meergo.ServerInfo{
		{
			Name:              ".NET",
			SourceDescription: "Import events and users from a server using .NET",
			Icon:              iconDotNet,
		},
		{
			Name:              "Go",
			SourceDescription: "Import events and users from a server using Go",
			Icon:              iconGo,
		},
		{
			Name:              "Java",
			SourceDescription: "Import events and users from a server using Java",
			Icon:              iconJava,
		},
		{
			Name:              "Node.js",
			SourceDescription: "Import events and users from a server using Node.js",
			Icon:              iconNode,
		},
		{
			Name:              "Python",
			SourceDescription: "Import events and users from a server using Python",
			Icon:              iconPython,
		},
	}
	for _, srv := range servers {
		meergo.RegisterServer(srv, New)
	}
}

// New returns a new Server connector instance.
func New(*meergo.ServerConfig) (*Server, error) {
	return &Server{}, nil
}

type Server struct{}
