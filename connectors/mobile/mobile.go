//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

// Package mobile implements the Android and Apple connectors.
package mobile

import (
	_ "embed"

	"github.com/meergo/meergo"
)

// Connector icon.
var iconAndroid = "<svg></svg>"

// Connector icon.
var iconApple = "<svg></svg>"

//go:embed documentation/android/overview.md
var androidOverview string

//go:embed documentation/apple/overview.md
var appleOverview string

func init() {
	mobiles := []meergo.SDKInfo{
		{
			Name:       "Android",
			Icon:       iconAndroid,
			Strategies: true,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from an Android mobile device",
					Overview: androidOverview,
				},
			},
		},
		{
			Name:       "Apple",
			Icon:       iconApple,
			Strategies: true,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from an Apple mobile device",
					Overview: appleOverview,
				},
			},
		},
	}
	for _, mobile := range mobiles {
		meergo.RegisterSDK(mobile, New)
	}
}

// New returns a new Mobile connector instance.
func New(*meergo.SDKConfig) (*Mobile, error) {
	return &Mobile{}, nil
}

type Mobile struct{}
