// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package segment provides a connector for Segment.
// (https://segment.com/docs/)
//
// Segment is a trademark of Twilio, Inc.
// This connector is not affiliated with or endorsed by Twilio, Inc.
package segment

import (
	_ "embed"

	"github.com/meergo/meergo/connectors"
)

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterWebhook(connectors.WebhookSpec{
		Code:       "segment",
		Label:      "Segment",
		Categories: connectors.CategorySaaS,
		Documentation: connectors.Documentation{
			Source: connectors.RoleDocumentation{
				Summary:  "Import events and users from Segment",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for Segment.
func New(env *connectors.WebhookEnv) (*Segment, error) {
	return &Segment{}, nil
}

type Segment struct{}
