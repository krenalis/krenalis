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

	"chichi/connector"
)

// Connector icon.
var iconAndroid = "<svg></svg>"

// Connector icon.
var iconApple = "<svg></svg>"

func init() {
	mobiles := []connector.Mobile{
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
		connector.RegisterMobile(srv, new)
	}
}

// new returns a new Mobile connection.
func new(*connector.MobileConfig) (*connection, error) {
	return &connection{}, nil
}

type connection struct{}
