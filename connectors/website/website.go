//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package website implements the JavaScript connector.
package website

import (
	_ "embed"

	"github.com/meergo/meergo"
)

// Connector icon.
var iconJavaScript = "<svg></svg>"

//go:embed documentation/overview.md
var overview string

func init() {
	websites := []meergo.SDKInfo{
		{
			Name:       "JavaScript",
			Icon:       iconJavaScript,
			Strategies: true,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a website using JavaScript",
					Overview: overview,
				},
			},
		},
	}
	for _, ws := range websites {
		meergo.RegisterSDK(ws, New)
	}
}

// New returns a new Website connector instance.
func New(*meergo.SDKConfig) (*Website, error) {
	return &Website{}, nil
}

type Website struct{}
