//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"context"
	"net"
	"reflect"
	"testing"

	"github.com/meergo/meergo/analytics-go"
	"github.com/meergo/meergo/test/meergotester"
)

func TestEventsContext(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Create a Meergo API connection, with an action to ingest the events.
	var meergoAPIID int
	var meergoAPIKey string
	{
		meergoAPIID = c.CreateMeergoAPISource("Meergo API", nil)
		keys := c.EventWriteKeys(meergoAPIID)
		if len(keys) != 1 {
			t.Fatalf("expected one key, got %d keys", len(keys))
		}
		meergoAPIKey = keys[0]
		c.CreateAction(meergoAPIID, "Event", meergotester.ActionToSet{
			Name:    "Ingest events",
			Enabled: true,
		})
	}

	// Send various events, with various user agent and OS configurations.
	c.SendEvent(meergoAPIKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 1",
		Context: &analytics.Context{
			UserAgent: "N/A",
		},
	})
	c.SendEvent(meergoAPIKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 2",
		Context: &analytics.Context{
			UserAgent: "",
		},
	})
	c.SendEvent(meergoAPIKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 3",
		Context: &analytics.Context{
			UserAgent: "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
		},
	})
	c.SendEvent(meergoAPIKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 4",
		Context: &analytics.Context{
			UserAgent: "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
			OS: analytics.OSInfo{
				Name:    "Linux",
				Version: "1",
			},
		},
	})
	c.SendEvent(meergoAPIKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 5",
		Context: &analytics.Context{
			IP:        net.IPv4zero,
			UserAgent: "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
			OS: analytics.OSInfo{
				Name:    "Linux",
				Version: "1",
			},
		},
	})

	// Wait until all events have been stored in the data warehouse, then
	// retrieve them.
	ctx := context.Background()
	const expectedEventsCount = 5
	c.WaitEventsStoredIntoWarehouse(ctx, expectedEventsCount)
	events := c.Events([]string{"event", "context"})
	if len(events) != expectedEventsCount {
		t.Fatal("unexpected error while retrieving events")
	}

	// Iterate over the received events and test that their context is the
	// expected one.
	for _, event := range events {
		var expectedContext map[string]any
		switch event["event"].(string) {
		case "Test Event 1":
			expectedContext = map[string]any{
				"ip": "127.0.0.1",
			}
		case "Test Event 2":
			expectedContext = map[string]any{
				"userAgent": "analytics-go (version: 0.0.4)",
				"browser": map[string]any{
					"name":  "Other",
					"other": "Unknown",
				},
				"ip": "127.0.0.1",
				"os": map[string]any{
					"name":  "Other",
					"other": "Unknown",
				},
			}
		case "Test Event 3":
			expectedContext = map[string]any{
				"userAgent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
				"browser": map[string]any{
					"name":    "Firefox",
					"version": "47.0.0",
				},
				"ip": "127.0.0.1",
				"os": map[string]any{
					"name":    "Windows",
					"version": "6.1.0",
				},
			}
		case "Test Event 4":
			expectedContext = map[string]any{
				"userAgent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
				"browser": map[string]any{
					"name":    "Firefox",
					"version": "47.0.0",
				},
				"ip": "127.0.0.1",
				"os": map[string]any{
					"name":    "Linux",
					"version": "1",
				},
			}
		case "Test Event 5":
			expectedContext = map[string]any{
				"userAgent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
				"browser": map[string]any{
					"name":    "Firefox",
					"version": "47.0.0",
				},
				"os": map[string]any{
					"name":    "Linux",
					"version": "1",
				},
			}
		default:
			t.Fatalf("unexpected event %q", event["event"].(string))
		}
		gotContext, _ := event["context"].(map[string]any)
		if !reflect.DeepEqual(gotContext, expectedContext) {
			t.Fatalf("%s: expected context %#v, got %#v", event["event"], expectedContext, gotContext)
		}
	}

}
