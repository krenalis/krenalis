//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
	"time"

	"chichi/backoff"
	"chichi/connector"
	"chichi/connector/types"
	"chichi/test/chichitester"

	"github.com/segmentio/analytics-go/v3"
)

func TestEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Load some users in the data warehouse.
	{
		dummySrc := c.AddDummy("Dummy (source)", connector.Source)
		importUsersID := c.AddAction(dummySrc, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Import users from Dummy",
				"InSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "firstName", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "firstName", Type: types.Text()},
				}),
				"Transformation": map[string]any{
					"Mapping": map[string]string{
						"email":     "email",
						"firstName": "firstName",
					},
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

	// Add a website connection (with an enabled action) and retrieve its key.
	var websiteID int
	var websiteKey string
	{
		websiteID = c.AddWebsiteSource("Website (source)", "example.com")
		keys := c.ConnectionKeys(websiteID)
		if len(keys) != 1 {
			t.Fatalf("expecting one key, got %d keys", len(keys))
		}
		websiteKey = keys[0]
		c.AddAction(websiteID, map[string]any{
			"Target": "Events",
			"Action": map[string]any{
				"Name":    "Website",
				"Enabled": true,
			},
		})
	}

	// Send 3 events.
	for i := 0; i < 3; i++ {
		c.SendEvent(websiteKey, analytics.Track{
			UserId:      "f4ca124298",
			AnonymousId: "baeeb556-96f3-4631-a22d-928431af8bf6",
			Event:       "Signed Up",
			Properties: analytics.Properties{
				"plan":       "Enterprise",
				"some-index": 42 + i,
			},
			Context: &analytics.Context{
				Page: analytics.PageInfo{
					URL: "https://example.com",
				},
			},
		})
	}

	ctx := context.Background()

	const expectedEventsCount = 3

	// Wait for the events to be stored in the warehouse.
	bo := backoff.New(20)
	bo.SetAttempts(10)
	bo.SetCap(2 * time.Second)
	bo.SetNextWaitTime(200 * time.Millisecond)
	for bo.Next(ctx) {
		count := c.CountEventsInWarehouse(ctx)
		if count == expectedEventsCount {
			break
		}
		t.Logf("[attempt %d] %d event(s) stored in warehouse until now", bo.Attempt(), count)
		if bo.WaitTime() == 0 {
			t.Fatalf("too many failed attempts")
		}
	}

	// Choose a GID to associate to events.
	userGID := 1

	// As a workaround, "manually" assign the GID to the events.
	count := c.AssociateGIDToEvents(ctx, userGID)
	if expectedEventsCount != count {
		t.Fatalf("expecting %d events affected, got %d", expectedEventsCount, count)
	}

	// Retrieve the first event for the user.
	var event map[string]any
	{
		events := c.UserEvents(userGID)
		if len(events) != expectedEventsCount {
			t.Fatalf("expecting %d events, got %d", expectedEventsCount, len(events))
		}
		event = events[0] // most recent event.
	}

	// Validate some fields of the event.
	{
		const (
			expectedAnonymousId = "baeeb556-96f3-4631-a22d-928431af8bf6"
			expectedIP          = "127.0.0.1"
			expectedUserAgent   = "analytics-go (version: 3.0.0)"
			expectedEvent       = "Signed Up"
			expectedProperties  = `{"plan":"Enterprise","some-index":44}`
			expectedTraits      = "{}"
			expectedType        = "track"
			expectedUserId      = "f4ca124298"
		)
		if event["anonymousId"] != expectedAnonymousId {
			t.Fatalf("expected anonymous ID %q, got %#v", expectedAnonymousId, event["anonymousId"])
		}
		if ip := event["context"].(map[string]any)["ip"]; ip != expectedIP {
			t.Fatalf("expected IP %q, got %#v", expectedIP, ip)
		}
		if ua := event["context"].(map[string]any)["userAgent"]; ua != expectedUserAgent {
			t.Fatalf("expected user agent %q, got %#v", expectedUserAgent, ua)
		}
		if event["event"] != expectedEvent {
			t.Fatalf("expected event %q, got %#v", expectedEvent, event["event"])
		}
		if !reflect.DeepEqual(event["properties"], expectedProperties) {
			t.Fatalf("expected properties %#v, got %#v", expectedProperties, event["properties"])
		}
		if source, err := strconv.Atoi(string(event["source"].(json.Number))); err != nil || source != websiteID {
			t.Fatalf("expected source %d, got %#v", websiteID, event["source"])
		}
		if !reflect.DeepEqual(event["traits"], expectedTraits) {
			t.Fatalf("expected traits %#v, got %#v", expectedTraits, event["traits"])
		}
		if event["type"] != expectedType {
			t.Fatalf("expected event type %q, got %#v", expectedType, event["type"])
		}
		if event["userId"] != expectedUserId {
			t.Fatalf("expected user ID %q, got %#v", expectedUserId, event["userId"])
		}
	}

}
