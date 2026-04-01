// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package webhook provides a connector for webhook.
package webhook

import (
	_ "embed"

	"github.com/krenalis/krenalis/connectors"
)

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterWebhook(connectors.WebhookSpec{
		Code:       "webhook",
		Label:      "Webhook",
		Categories: connectors.CategoryWebhook,
		Documentation: connectors.Documentation{
			Source: connectors.RoleDocumentation{
				Summary:  "Import events and users from your application with a webhook",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for webhook.
func New(env *connectors.WebhookEnv) (*Webhook, error) {
	return &Webhook{}, nil
}

type Webhook struct{}
