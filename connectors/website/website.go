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

	"chichi/connector"
)

// Connector icon.
var iconJavaScript = "<svg></svg>"

func init() {
	websites := []connector.Website{
		{
			Name:              "JavaScript",
			SourceDescription: "collect events, and import users and groups from a website using JavaScript",
			Icon:              iconJavaScript,
		},
	}
	for _, ws := range websites {
		connector.RegisterWebsite(ws, new)
	}
}

// new returns a new Website connection.
func new(*connector.WebsiteConfig) (*connection, error) {
	return &connection{}, nil
}

type connection struct{}
