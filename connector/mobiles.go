//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
)

// Mobile represents a mobile connector.
type Mobile struct {
	Name string
	Icon string // icon in SVG format
	Open OpenMobileFunc
}

// MobileConfig represents the configuration of a mobile connection.
type MobileConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenMobileFunc represents functions that open mobile connections.
type OpenMobileFunc func(context.Context, *MobileConfig) (MobileConnection, error)

// MobileConnection is the interface implemented by mobile connections.
type MobileConnection interface{}
