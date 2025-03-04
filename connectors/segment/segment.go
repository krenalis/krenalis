//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package segment implements the Segment connector.
package segment

import (
	_ "embed"

	"github.com/meergo/meergo"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterServer(meergo.ServerInfo{
		Name:              "Segment",
		SourceDescription: "Import events and users from Segment",
		Icon:              icon,
	}, New)
}

// New returns a new Segment connector instance.
func New(*meergo.ServerConfig) (*Segment, error) {
	return &Segment{}, nil
}

type Segment struct{}
