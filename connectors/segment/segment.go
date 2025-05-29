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

//go:embed documentation/overview.md
var overview string

func init() {
	meergo.RegisterSDK(meergo.SDKInfo{
		Name: "Segment",
		Icon: icon,
		Documentation: meergo.ConnectorDocumentation{
			Source: meergo.ConnectorRoleDocumentation{
				Summary:  "Import events and users from Segment",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new Segment connector instance.
func New(*meergo.SDKConfig) (*Segment, error) {
	return &Segment{}, nil
}

type Segment struct{}
