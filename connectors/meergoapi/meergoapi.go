//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package meergoapi provides a connector for Meergo API.
package meergoapi

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
		Name:       "Meergo API",
		Categories: meergo.CategorySDKAndAPI,
		Icon:       icon,
		Documentation: meergo.ConnectorDocumentation{
			Source: meergo.ConnectorRoleDocumentation{
				Summary:  "Import events and users by calling the Meergo APIs from your application",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for Meergo API.
func New(env *meergo.SDKEnv) (*MeergoAPI, error) {
	return &MeergoAPI{}, nil
}

type MeergoAPI struct{}
