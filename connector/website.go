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
	Name string
	Icon string // icon in SVG format
	Open OpenWebsiteFunc
}

// WebsiteConfig represents the configuration of a website connection.
type WebsiteConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenWebsiteFunc represents functions that open website connections.
type OpenWebsiteFunc func(context.Context, *WebsiteConfig) (WebsiteConnection, error)

// WebsiteConnection is the interface implemented by website connections.
type WebsiteConnection interface{}
