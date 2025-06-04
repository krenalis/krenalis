//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package meergoapi implements the meergoapi connector.
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
		Categories: meergo.CategoryAPI,
		Icon:       icon,
		Documentation: meergo.ConnectorDocumentation{
			Source: meergo.ConnectorRoleDocumentation{
				Summary:  "Import events and users by calling the Meergo APIs from your application",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new Meergo API connector instance.
func New(*meergo.SDKConfig) (*MeergoAPI, error) {
	return &MeergoAPI{}, nil
}

type MeergoAPI struct{}
