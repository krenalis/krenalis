// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"log/slog"
	"net"
	"runtime"

	"github.com/google/uuid"
	"github.com/krenalis/analytics-go"
)

// sendNotificationStats sends information about notification n to Krenalis.
func (state *State) sendNotificationStats(client analytics.Client, organization uuid.UUID, n notification) {
	go func() {
		err := client.Enqueue(analytics.Track{
			UserId: organization.String(),
			Event:  "State Changed",
			Context: &analytics.Context{
				OS: analytics.OSInfo{
					Name: eventContextOs,
				},
				IP: net.IP{255, 255, 0, 0},
			},
			Properties: analytics.NewProperties().Set("notification_name", n.Name),
		})
		if err != nil {
			slog.Error("cannot enqueue Track event when sending stats to Krenalis", "error", err)
		}
	}()
}

// eventContextOs is the name of the OS on which this Krenalis instance is
// running, in a format accepted by the 'context.os.name' field of the Krenalis
// API endpoint that ingests events.
var eventContextOs string

func init() {
	switch runtime.GOOS {
	case "darwin":
		eventContextOs = "macOS"
	case "linux":
		eventContextOs = "Linux"
	case "windows":
		eventContextOs = "Windows"
	default:
		eventContextOs = "Other"
	}
}

// discardLogger is an 'analytics-go/Logger' that discard everything without
// logging anything.
type discardLogger struct{}

func (dl discardLogger) Logf(format string, args ...any) {
	// Do nothing.
}

func (dl discardLogger) Errorf(format string, args ...any) {
	// Do nothing.
}
