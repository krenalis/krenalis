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

	"github.com/open2b/chichi"
)

// Connector icon.
var iconJavaScript = "<svg></svg>"

func init() {
	websites := []chichi.WebsiteInfo{
		{
			Name:              "JavaScript",
			SourceDescription: "import events, users and groups from a website using JavaScript",
			Icon:              iconJavaScript,
		},
	}
	for _, ws := range websites {
		chichi.RegisterWebsite(ws, New)
	}
}

// New returns a new Website connector instance.
func New(*chichi.WebsiteConfig) (*Website, error) {
	return &Website{}, nil
}

type Website struct{}
