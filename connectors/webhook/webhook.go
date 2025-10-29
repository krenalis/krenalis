// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package webhook provides a connector for webhook.
package webhook

import (
	_ "embed"

	"github.com/meergo/meergo"
)

//go:embed documentation/overview.md
var overview string

func init() {
	meergo.RegisterWebhook(meergo.WebhookSpec{
		Code:       "webhook",
		Label:      "Webhook",
		Categories: meergo.CategoryWebhook,
		Documentation: meergo.ConnectorDocumentation{
			Source: meergo.ConnectorRoleDocumentation{
				Summary:  "Import events and users from your application with a webhook",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for webhook.
func New(env *meergo.WebhookEnv) (*Webhook, error) {
	return &Webhook{}, nil
}

type Webhook struct{}
