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

//go:embed documentation/dotnet/overview.md
var dotnetOverview string

//go:embed documentation/go/overview.md
var goOverview string

//go:embed documentation/java/overview.md
var javaOverview string

//go:embed documentation/node/overview.md
var nodeOverview string

//go:embed documentation/python/overview.md
var pythonOverview string

func init() {
	servers := []meergo.ServerInfo{
		{
			Name: ".NET",
			Icon: iconDotNet,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a server using .NET",
					Overview: dotnetOverview,
				},
			},
		},
		{
			Name: "Go",
			Icon: iconGo,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a server using Go",
					Overview: goOverview,
				},
			},
		},
		{
			Name: "Java",
			Icon: iconJava,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a server using Java",
					Overview: javaOverview,
				},
			},
		},
		{
			Name: "Node.js",
			Icon: iconNode,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a server using Node.js",
					Overview: nodeOverview,
				},
			},
		},
		{
			Name: "Python",
			Icon: iconPython,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a server using Python",
					Overview: pythonOverview,
				},
			},
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
