// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package rudderstack provides a connector for RudderStack.
// (https://www.rudderstack.com/docs/)
//
// RudderStack is a trademark of RudderStack, Inc.
// This connector is not affiliated with or endorsed by RudderStack, Inc.
package rudderstack

import (
	_ "embed"

	"github.com/meergo/meergo/connectors"
)

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterWebhook(connectors.WebhookSpec{
		Code:       "rudderstack",
		Label:      "RudderStack",
		Categories: connectors.CategorySaaS,
		Documentation: connectors.ConnectorDocumentation{
			Source: connectors.ConnectorRoleDocumentation{
				Summary:  "Import events and users from RudderStack",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for RudderStack.
func New(env *connectors.WebhookEnv) (*RudderStack, error) {
	return &RudderStack{}, nil
}

type RudderStack struct{}
