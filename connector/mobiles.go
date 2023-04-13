//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"reflect"
)

// Mobile represents a mobile connector.
type Mobile struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	open reflect.Value
}

// Open opens a mobile connection.
func (mobile Mobile) Open(ctx context.Context, conf *MobileConfig) (MobileConnection, error) {
	out := mobile.open.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conf)})
	c := out[0].Interface().(MobileConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// MobileConfig represents the configuration of a mobile connection.
type MobileConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenMobileFunc represents functions that open mobile connections.
type OpenMobileFunc[T MobileConnection] func(context.Context, *MobileConfig) (T, error)

// MobileConnection is the interface implemented by mobile connections.
type MobileConnection interface{}
