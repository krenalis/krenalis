//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package state

import (
	"log/slog"

	"github.com/meergo/meergo/analytics-go"
)

// sendNotificationStats sends information about notification n to Meergo.
func (state *State) sendNotificationStats(client analytics.Client, n notification) {
	if n.Name == "SeeLeader" {
		// Many "SeeLeader" notifications are received, and they're mostly
		// irrelevant. That's why they're not sent.
		return
	}
	go func() {
		err := client.Enqueue(analytics.Track{
			UserId:     state.metadata.installationID,
			Event:      "State Changed",
			Properties: analytics.NewProperties().Set("notification_name", n.Name),
		})
		if err != nil {
			slog.Error("cannot enqueue Track event when sending stats to Meergo", "err", err)
		}
	}()
}
