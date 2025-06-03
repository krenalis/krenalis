//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package server implements the .Net, Android, Go, Java, JavaScript, Node.js,
// and Python connectors.
package server

import (
	_ "embed"

	"github.com/meergo/meergo"
)

// Connector icon.
var iconDotNet = "<svg></svg>"

// Connector icon.
var iconAndroid = "<svg></svg>"

// Connector icon.
var iconGo = "<svg></svg>"

// Connector icon.
var iconJava = "<svg></svg>"

// Connector icon.
var iconJavaScript = "<svg></svg>"

// Connector icon.
var iconNode = "<svg></svg>"

// Connector icon.
var iconPython = "<svg></svg>"

//go:embed documentation/dotnet/overview.md
var dotnetOverview string

//go:embed documentation/android/overview.md
var androidOverview string

//go:embed documentation/go/overview.md
var goOverview string

//go:embed documentation/java/overview.md
var javaOverview string

//go:embed documentation/javascript/overview.md
var javaScriptOverview string

//go:embed documentation/node/overview.md
var nodeOverview string

//go:embed documentation/python/overview.md
var pythonOverview string

func init() {
	servers := []meergo.SDKInfo{
		{
			Name:       ".NET",
			Categories: meergo.CategorySDK,
			Icon:       iconDotNet,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using .NET",
					Overview: dotnetOverview,
				},
			},
		},
		{
			Name:       "Android",
			Categories: meergo.CategorySDK | meergo.CategoryMobile,
			Icon:       iconAndroid,
			Strategies: true,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from an Android mobile device",
					Overview: androidOverview,
				},
			},
		},
		{
			Name:       "Go",
			Categories: meergo.CategorySDK,
			Icon:       iconGo,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Go",
					Overview: goOverview,
				},
			},
		},
		{
			Name:       "Java",
			Categories: meergo.CategorySDK,
			Icon:       iconJava,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Java",
					Overview: javaOverview,
				},
			},
		},
		{
			Name:       "JavaScript",
			Categories: meergo.CategorySDK | meergo.CategoryWebsite,
			Icon:       iconJavaScript,
			Strategies: true,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a website using JavaScript",
					Overview: javaScriptOverview,
				},
			},
		},
		{
			Name:       "Node.js",
			Categories: meergo.CategorySDK,
			Icon:       iconNode,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Node.js",
					Overview: nodeOverview,
				},
			},
		},
		{
			Name:       "Python",
			Categories: meergo.CategorySDK,
			Icon:       iconPython,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Python",
					Overview: pythonOverview,
				},
			},
		},
	}
	for _, srv := range servers {
		meergo.RegisterSDK(srv, New)
	}
}

// New returns a new SDK connector instance.
func New(*meergo.SDKConfig) (*SDK, error) {
	return &SDK{}, nil
}

type SDK struct{}
