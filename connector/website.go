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

// WebsiteConfig represents the configuration of a website connection.
type WebsiteConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// WebsiteConnectionFunc represents functions that create new website
// connections.
type WebsiteConnectionFunc func(context.Context, *WebsiteConfig) (WebsiteConnection, error)

// WebsiteConnection is the interface implemented by website connections.
type WebsiteConnection interface {
	Connection
}
