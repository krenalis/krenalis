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

//go:embed documentation/overview.md
var overview string

func init() {
	meergo.RegisterSDK(meergo.SDKInfo{
		Name:       "RudderStack",
		Categories: meergo.CategoryAnalytics,
		Icon:       icon,
		Documentation: meergo.ConnectorDocumentation{
			Source: meergo.ConnectorRoleDocumentation{
				Summary:  "Import events and users from RudderStack",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new RudderStack connector instance.
func New(env *meergo.SDKEnv) (*RudderStack, error) {
	return &RudderStack{}, nil
}

type RudderStack struct{}
