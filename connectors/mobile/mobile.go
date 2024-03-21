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

	"chichi"
)

// Connector icon.
var iconAndroid = "<svg></svg>"

// Connector icon.
var iconApple = "<svg></svg>"

func init() {
	mobiles := []chichi.Mobile{
		{
			Name:              "Android",
			SourceDescription: "collect events, and import users and groups from an Android mobile device",
			Icon:              iconAndroid,
		},
		{
			Name:              "Apple",
			SourceDescription: "collect events, and import users and groups from an Apple mobile device",
			Icon:              iconApple,
		},
	}
	for _, srv := range mobiles {
		chichi.RegisterMobile(srv, New)
	}
}

// New returns a new Mobile connection.
func New(*chichi.MobileConfig) (*Mobile, error) {
	return &Mobile{}, nil
}

type Mobile struct{}
