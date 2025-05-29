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
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/meergo/meergo/test/analytics-go"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

func TestEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Load some users in the data warehouse from Dummy.
	dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
	importUsersID := c.CreateAction(dummySrc, "Users", meergotester.ActionToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "firstName", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
			},
		},
	})
	exec := c.ExecuteAction(importUsersID)
	c.WaitForExecutionsCompletion(dummySrc, exec)

	// Create a JavaScript connection with 2 actions (one for importing events,
	// one for importing user identities) and retrieve its key.
	var javaScriptID int
	var javaScriptKey string
	{
		javaScriptID = c.CreateJavaScriptSource("JavaScript (source)", nil)
		keys := c.EventWriteKeys(javaScriptID)
		if len(keys) != 1 {
			t.Fatalf("expected one key, got %d keys", len(keys))
		}
		javaScriptKey = keys[0]
		c.CreateAction(javaScriptID, "Events", meergotester.ActionToSet{
			Name:    "JavaScript",
			Enabled: true,
		})
		c.CreateAction(javaScriptID, "Users", meergotester.ActionToSet{
			Name:     "JavaScript",
			Enabled:  true,
			InSchema: types.Type{},
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
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

	// Make a Group call.
	c.SendEvent(javaScriptKey, analytics.Group{
		UserId:  "f4ca124298",
		GroupId: "uy55IELNg",
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

	time.Sleep(time.Second)
	c.RunIdentityResolution()

	ctx := context.Background()

	const expectedEventsCount = 5

	c.WaitEventsStoredIntoWarehouse(ctx, expectedEventsCount)

	// Run the identity resolution, so that the events GID are updated.
	time.Sleep(time.Second)
	c.RunIdentityResolution()

	// Retrieve the user imported from the event.
	users, _, total := c.Users([]string{"email"}, "", false, 0, 100)
	const expectedUsersTotal = 10 + 1 // 10 imported from Dummy, 1 imported from JavaScript, with the identity call
	if expectedUsersTotal != total {
		t.Fatalf("expected %d users, got %d", expectedUsersTotal, total)
	}
	var userGID uuid.UUID
	for _, user := range users {
		email, _ := user.Traits["email"].(string)
		if email == eventUserEmail {
			userGID = user.ID
			break
		}
	}
	if userGID == (uuid.UUID{}) {
		t.Fatalf("user with email %q not found", eventUserEmail)
	}
	t.Logf("user imported from event has GID %s", userGID)

	// Retrieve the first event for the user.
	var event map[string]any
	events := c.UserEvents(userGID, []string{"anonymousId", "context", "event", "properties", "connection", "traits", "type", "userId", "groupId"})
	if len(events) != expectedEventsCount {
		t.Fatalf("expected %d events for user %s, got %d", expectedEventsCount, userGID, len(events))
	}
	event = events[0] // most recent event.

	// Validate some fields of the first event.
	{
		var expectedProperties = map[string]any{"plan": "Enterprise", "some-index": json.Number("44")}
		var expectedTraits = map[string]any{}
		const (
			expectedAnonymousId = "baeeb556-96f3-4631-a22d-928431af8bf6"
			expectedIP          = "127.0.0.1"
			expectedEvent       = "Signed Up"
			expectedType        = "track"
			expectedUserId      = "f4ca124298"
		)
		var expectedUserAgent = regexp.MustCompile(`^analytics-go \(version: \d+\.\d+\.\d+\)$`)
		if event["anonymousId"] != expectedAnonymousId {
			t.Fatalf("expected anonymous ID %q, got %#v", expectedAnonymousId, event["anonymousId"])
		}
		if ip := event["context"].(map[string]any)["ip"]; ip != expectedIP {
			t.Fatalf("expected IP %q, got %#v", expectedIP, ip)
		}
		if ua := event["context"].(map[string]any)["userAgent"].(string); !expectedUserAgent.MatchString(ua) {
			t.Fatalf("expected user agent %q, got %#v", expectedUserAgent, ua)
		}
		if event["event"] != expectedEvent {
			t.Fatalf("expected event %q, got %#v", expectedEvent, event["event"])
		}
		if !reflect.DeepEqual(event["properties"], expectedProperties) {
			t.Fatalf("expected properties %#v, got %#v", expectedProperties, event["properties"])
		}
		if source, err := strconv.Atoi(string(event["connection"].(json.Number))); err != nil || source != javaScriptID {
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

	// Validate some fields of the Group call event.
	var groupEvent map[string]any
	for _, event := range events {
		if event["type"] == "group" {
			groupEvent = event
			break
		}
	}
	if groupEvent == nil {
		t.Fatalf("event corresponding to the Group call not found")
	}
	if groupEvent["userId"] != "f4ca124298" {
		t.Fatalf("expected a userId = %q, got %q", "f4ca124298", groupEvent["userId"])
	}
	if groupEvent["groupId"] != "uy55IELNg" {
		t.Fatalf("expected a groupId = %q, got %q", "uy55IELNg", groupEvent["groupId"])
	}

	// Test importing a user identity with an action that has no mapping.
	javaScript2ID := c.CreateJavaScriptSource("JavaScript (source 2)", nil)
	javaScript2Key := c.EventWriteKeys(javaScript2ID)[0]
	c.CreateAction(javaScript2ID, "Users", meergotester.ActionToSet{
		Name:    "JavaScript",
		Enabled: true,
	})
	c.SendEvent(javaScript2Key, analytics.Identify{
		UserId:      "Zny0kLMyz",
		AnonymousId: "bd857fe0-8f62-4d36-8e47-0161db0cc513",
	})
	time.Sleep(time.Second)
	_, total = c.ConnectionIdentities(javaScript2ID, 0, 100)
	if total != 1 {
		t.Fatalf("expected one identity, got %d", total)
	}

}
