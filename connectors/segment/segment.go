//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package segment provides a connector for Segment.
// (https://segment.com/docs/)
//
// Segment is a trademark of Twilio, Inc.
// This connector is not affiliated with or endorsed by Twilio, Inc.
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
		Code:       "segment",
		Label:      "Segment",
		Categories: meergo.CategoryEventStreaming,
		Icon:       icon,
		Documentation: meergo.ConnectorDocumentation{
			Source: meergo.ConnectorRoleDocumentation{
				Summary:  "Import events and users from Segment",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for Segment.
func New(env *meergo.SDKEnv) (*Segment, error) {
	return &Segment{}, nil
}

type Segment struct{}
