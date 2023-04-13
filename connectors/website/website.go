//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package website

// This package is the Website connector.

import (
	"context"
	_ "embed"

	"chichi/connector"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterWebsite(connector.Website{
		Name:              "Website",
		SourceDescription: "receive events from a website",
		Icon:              icon,
	}, open)
}

type connection struct{}

// open opens a Website connection and returns it.
func open(context.Context, *connector.WebsiteConfig) (*connection, error) {
	return &connection{}, nil
}
