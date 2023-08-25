//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package website implements the Website connector.
package website

// This package is the Website connector.

import (
	_ "embed"

	"chichi/connector"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterWebsite(connector.Website{
		Name:              "Website",
		SourceDescription: "collect events, and import users and groups from a website",
		Icon:              icon,
	}, open)
}

// open opens a Website connection and returns it.
func open(*connector.WebsiteConfig) (*connection, error) {
	return &connection{}, nil
}

type connection struct{}
