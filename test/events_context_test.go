// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"context"
	"net"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"

	"github.com/krenalis/analytics-go"
)

func TestEventsContext(t *testing.T) {

	// TODO: skipped until https://github.com/krenalis/krenalis/issues/2150 is fixed.
	t.Skip()

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	// Create a webhook connection, with a pipeline to ingest the events.
	var webhookID int
	var webhookEventWriteKey string
	{
		webhookID = c.CreateWebhookSource("Krenalis API", nil)
		keys := c.EventWriteKeys(webhookID)
		if len(keys) != 1 {
			t.Fatalf("expected one key, got %d keys", len(keys))
		}
		webhookEventWriteKey = keys[0]
		c.CreatePipeline(webhookID, "Event", krenalistester.PipelineToSet{
			Name:    "Ingest events",
			Enabled: true,
		})
	}

	// Send various events, with various user agent and OS configurations.
	c.SendEvent(webhookEventWriteKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 1",
	})
	c.SendEvent(webhookEventWriteKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 2",
		Context: &analytics.Context{
			UserAgent: "",
			IP:        net.ParseIP("255.255.255.255"),
		},
	})
	c.SendEvent(webhookEventWriteKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 3",
		Context: &analytics.Context{
			UserAgent: "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
			IP:        net.ParseIP("255.255.255.255"),
		},
	})
	c.SendEvent(webhookEventWriteKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 4",
		Context: &analytics.Context{
			UserAgent: "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
			OS: analytics.OSInfo{
				Name:    "Linux",
				Version: "1",
			},
			IP: net.ParseIP("255.255.255.0"),
		},
	})
	c.SendEvent(webhookEventWriteKey, analytics.Track{
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
	c.SendEvent(webhookEventWriteKey, analytics.Track{
		AnonymousId: "ff8dee31-fd87-45bb-978b-c7b3e2c52128",
		Event:       "Test Event 6",
		Context: &analytics.Context{
			IP:        net.IPv4zero,
			UserAgent: "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
			OS: analytics.OSInfo{
				Name:    "darwin",
				Version: "12",
			},
		},
	})

	// Wait until all events have been stored in the data warehouse, then
	// retrieve them.
	ctx := context.Background()
	const expectedEventsCount = 6
	c.WaitEventsStoredIntoWarehouse(ctx, expectedEventsCount)
	events := c.Events([]string{"event", "context"})
	if len(events) != expectedEventsCount {
		t.Fatal("unexpected error while retrieving events")
	}

	library := map[string]any{
		"name":    "analytics-go",
		"version": analytics.Version,
	}

	// Iterate over the received events and test that their context is the
	// expected one.
	for _, event := range events {
		var expectedContext map[string]any
		switch event["event"].(string) {
		case "Test Event 1":
			expectedContext = map[string]any{
				"library": library,
			}
		case "Test Event 2":
			expectedContext = map[string]any{
				"ip":      "127.0.0.1",
				"library": library,
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
				"library": library,
			}
		case "Test Event 4":
			expectedContext = map[string]any{
				"userAgent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
				"browser": map[string]any{
					"name":    "Firefox",
					"version": "47.0.0",
				},
				"ip": "127.0.0.0",
				"os": map[string]any{
					"name":    "Linux",
					"version": "1",
				},
				"library": library,
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
				"library": library,
			}
		case "Test Event 6":
			expectedContext = map[string]any{
				"userAgent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0",
				"browser": map[string]any{
					"name":    "Firefox",
					"version": "47.0.0",
				},
				"os": map[string]any{
					"name":    "macOS",
					"version": "12",
				},
				"library": library,
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
