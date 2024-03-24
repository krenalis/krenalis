//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package server implements the .Net, Go, Java, Node.js, PHP and Python
// connectors.
package server

import (
	_ "embed"

	"chichi"
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
var iconPHP = "<svg></svg>"

// Connector icon.
var iconPython = "<svg></svg>"

func init() {
	servers := []chichi.ServerInfo{
		{
			Name:              ".NET",
			SourceDescription: "import events, users and groups from a server using .NET",
			Icon:              iconDotNet,
		},
		{
			Name:              "Go",
			SourceDescription: "import events, users and groups from a server using Go",
			Icon:              iconGo,
		},
		{
			Name:              "Java",
			SourceDescription: "import events, users and groups from a server using Java",
			Icon:              iconJava,
		},
		{
			Name:              "Node.js",
			SourceDescription: "import events, users and groups from a server using Node.js",
			Icon:              iconNode,
		},
		{
			Name:              "PHP",
			SourceDescription: "import events, users and groups from a server using PHP",
			Icon:              iconPHP,
		},
		{
			Name:              "Python",
			SourceDescription: "import events, users and groups from a server using Python",
			Icon:              iconPython,
		},
	}
	for _, srv := range servers {
		chichi.RegisterServer(srv, New)
	}
}

// New returns a new Server connector instance.
func New(*chichi.ServerConfig) (*Server, error) {
	return &Server{}, nil
}

type Server struct{}
