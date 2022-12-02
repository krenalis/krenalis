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

// Website represents a website connector.
type Website struct {
	Name    string
	Icon    []byte // icon in SVG format
	Connect WebsiteConnectFunc
}

// WebsiteConfig represents the configuration of a website connection.
type WebsiteConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// WebsiteConnectFunc represents functions that create new website connections.
type WebsiteConnectFunc func(context.Context, *WebsiteConfig) (WebsiteConnection, error)

// WebsiteConnection is the interface implemented by website connections.
type WebsiteConnection interface {
	Connection
}
