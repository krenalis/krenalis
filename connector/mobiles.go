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
	Name    string
	Icon    []byte // icon in SVG format
	Connect MobileConnectFunc
}

// MobileConfig represents the configuration of a mobile connection.
type MobileConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// MobileConnectFunc represents functions that create new mobile connections.
type MobileConnectFunc func(context.Context, *MobileConfig) (MobileConnection, error)

// MobileConnection is the interface implemented by mobile connections.
type MobileConnection interface {
	Connection
}
