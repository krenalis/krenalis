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

	"chichi/test/chichitester"
	"chichi/types"

	"github.com/segmentio/analytics-go/v3"
)

func TestEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Load some users in the data warehouse from Dummy.
	dummySrc := c.AddDummy("Dummy (source)", chichitester.Source, "")
	importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "firstName", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"firstName": "firstName",
			},
		},
	})
	c.ExecuteAction(dummySrc, importUsersID, true)
	c.WaitActionsToFinish(dummySrc)

	// Add a JavaScript connection with two actions (one for importing events,
	// one for importing users identities) and retrieve its key.
	var javaScriptID int
	var javaScriptKey string
	{
		javaScriptID = c.AddJavaScriptSource("JavaScript (source)", "example.com", "")
		keys := c.ConnectionKeys(javaScriptID)
		if len(keys) != 1 {
			t.Fatalf("expecting one key, got %d keys", len(keys))
		}
		javaScriptKey = keys[0]
		c.AddAction(javaScriptID, "Events", chichitester.ActionToSet{
			Name:    "JavaScript",
			Enabled: true,
		})
		c.AddAction(javaScriptID, "Users", chichitester.ActionToSet{
			Name:     "JavaScript",
			Enabled:  true,
			InSchema: types.Type{},
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email": "traits.email",
				},
			},
		})
	}

	const eventUserEmail = "event-user@example.com"

	// Send an identity event. More than importing an event, this should create
	// a user identity.
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "f4ca124298",
		Traits: map[string]interface{}{
			"email": eventUserEmail,
		},
	})

	// Send 3 events.
	for i := 0; i < 3; i++ {
		c.SendEvent(javaScriptKey, analytics.Track{
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

	const expectedEventsCount = 4

	c.WaitEventsStoredIntoWarehouse(ctx, expectedEventsCount)

	// Trigger the identity resolution, so that the events GID are updated.
	c.ExecuteAction(dummySrc, importUsersID, true)
	c.WaitActionsToFinish(dummySrc)

	// Retrieve the user imported from the event.
	users, _, count := c.Users([]string{"Id", "email"}, "", 0, 100)
	const expectedUsersCount = 10 + 1 // 10 imported from Dummy, 1 imported from JavaScript, with the identity call
	if expectedUsersCount != count {
		t.Fatalf("expecting %d users, got %d", expectedUsersCount, count)
	}
	var userGID int64
	for _, user := range users {
		email, _ := user["email"].(string)
		if email == eventUserEmail {
			userGID, _ = user["Id"].(json.Number).Int64()
			if userGID <= 0 {
				t.Fatalf("invalid user GID %d", userGID)
			}
			break
		}
	}
	if userGID == 0 {
		t.Fatalf("user with email %q not found", eventUserEmail)
	}
	t.Logf("user imported from event has GID %d", userGID)

	// Retrieve the first event for the user.
	var event map[string]any
	events := c.UserEvents(int(userGID))
	if len(events) != expectedEventsCount {
		t.Fatalf("expecting %d events for user %d, got %d", expectedEventsCount, userGID, len(events))
	}
	event = events[0] // most recent event.

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
		if source, err := strconv.Atoi(string(event["source"].(json.Number))); err != nil || source != javaScriptID {
			t.Fatalf("expected source %d, got %#v", javaScriptID, event["source"])
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
