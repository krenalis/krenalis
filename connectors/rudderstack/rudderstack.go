//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package rudderstack provides a connector for RudderStack.
// (https://www.rudderstack.com/docs/)
//
// RudderStack is a trademark of RudderStack, Inc.
// This connector is not affiliated with or endorsed by RudderStack, Inc.
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
		Categories: meergo.CategoryEventStreaming,
		Icon:       icon,
		Documentation: meergo.ConnectorDocumentation{
			Source: meergo.ConnectorRoleDocumentation{
				Summary:  "Import events and users from RudderStack",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for RudderStack.
func New(env *meergo.SDKEnv) (*RudderStack, error) {
	return &RudderStack{}, nil
}

type RudderStack struct{}
