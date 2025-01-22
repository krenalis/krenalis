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

func init() {
	mobiles := []meergo.MobileInfo{
		{
			Name:              "Android",
			SourceDescription: "Import events and users from an Android mobile device",
			Icon:              iconAndroid,
		},
		{
			Name:              "Apple",
			SourceDescription: "Import events and users from an Apple mobile device",
			Icon:              iconApple,
		},
	}
	for _, srv := range mobiles {
		meergo.RegisterMobile(srv, New)
	}
}

// New returns a new Mobile connector instance.
func New(*meergo.MobileConfig) (*Mobile, error) {
	return &Mobile{}, nil
}

type Mobile struct{}
