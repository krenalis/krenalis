//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package rudderstack implements the RudderStack connector.
package rudderstack

import (
	_ "embed"

	"github.com/meergo/meergo"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterServer(meergo.ServerInfo{
		Name:              "RudderStack",
		SourceDescription: "Import events and users from RudderStack",
		Icon:              icon,
	}, New)
}

// New returns a new RudderStack connector instance.
func New(*meergo.ServerConfig) (*RudderStack, error) {
	return &RudderStack{}, nil
}

type RudderStack struct{}
