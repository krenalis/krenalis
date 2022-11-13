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

// MobileConfig represents the configuration of a mobile connection.
type MobileConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// MobileConnectionFunc represents functions that create new mobile
// connections.
type MobileConnectionFunc func(context.Context, *MobileConfig) (MobileConnection, error)

// MobileConnection is the interface implemented by mobile connections.
type MobileConnection interface {
	Connection
}
