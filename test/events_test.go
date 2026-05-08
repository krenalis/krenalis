// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/uuid"
	"github.com/krenalis/analytics-go"
)

func TestEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	// Disable automatic execution of Identity Resolution.
	c.UpdateIdentityResolution(false, nil)

	// Load some users in the data warehouse from Dummy.
	dummySrc := c.CreateDummy("Dummy (source)", krenalistester.Source)
	importUsersID := c.CreatePipeline(dummySrc, "User", krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "firstName", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
			},
		},
	})
	run := c.RunPipeline(importUsersID)
	c.WaitRunsCompletion(dummySrc, run)
	c.RunIdentityResolution()

	// Create a JavaScript connection with 2 pipelines (one for importing events,
	// one for importing identities) and retrieve its key.
	var javaScriptID int
	var javaScriptKey string
	{
		javaScriptID = c.CreateJavaScriptSource("JavaScript (source)", nil)
		keys := c.EventWriteKeys(javaScriptID)
		if len(keys) != 1 {
			t.Fatalf("expected one key, got %d keys", len(keys))
		}
		javaScriptKey = keys[0]
		c.CreatePipeline(javaScriptID, "Event", krenalistester.PipelineToSet{
			Name:    "JavaScript",
			Enabled: true,
		})
		c.CreatePipeline(javaScriptID, "User", krenalistester.PipelineToSet{
			Name:     "JavaScript",
			Enabled:  true,
			Filter:   krenalistester.DefaultFilterUserFromEvents,
			InSchema: types.Type{},
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			}),
			Transformation: &krenalistester.Transformation{
				Mapping: map[string]string{
					"email": "traits.email",
				},
			},
		})
	}

	const eventProfileEmail = "event-profile@example.com"

	// Send an identity event. More than importing an event, this should create an identity.
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "f4ca124298",
		Traits: map[string]any{
			"email": eventProfileEmail,
		},
	})

	// Make a Group call.
	c.SendEvent(javaScriptKey, analytics.Group{
		UserId:  "f4ca124298",
		GroupId: "uy55IELNg",
	})

	// Send 3 events.
	for i := range 3 {
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
		// Wait 1ms to ensure distinct timestamps and preserve event order.
		time.Sleep(time.Millisecond)
	}

	ctx := context.Background()

	c.WaitConnectionIdentitiesStoredIntoWarehouse(ctx, javaScriptID, 1)
	c.RunIdentityResolution()

	const expectedEventsCount = 5

	c.WaitEventsStoredIntoWarehouse(ctx, expectedEventsCount)

	// Run the identity resolution, so that the events KPID are updated.
	c.RunIdentityResolution()

	// Retrieve the profile imported from the event.
	profiles, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)
	const expectedProfilesTotal = 10 + 1 // 10 imported from Dummy, 1 imported from JavaScript, with the identity call
	if expectedProfilesTotal != total {
		t.Fatalf("expected %d profiles, got %d", expectedProfilesTotal, total)
	}
	var kpid uuid.UUID
	for _, profile := range profiles {
		email, _ := profile.Attributes["email"].(string)
		if email == eventProfileEmail {
			kpid = profile.KPID
			break
		}
	}
	if kpid == (uuid.UUID{}) {
		t.Fatalf("profile with email %q not found", eventProfileEmail)
	}
	t.Logf("profile imported from event has KPID %s", kpid)

	// Retrieve the first event for the profile.
	var event map[string]any
	events := c.ProfileEvents(kpid, []string{"anonymousId", "context", "event", "properties", "connectionId", "traits", "type", "userId", "groupId"})
	if len(events) != expectedEventsCount {
		t.Fatalf("expected %d events for profile %s, got %d", expectedEventsCount, kpid, len(events))
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
		if event["anonymousId"] != expectedAnonymousId {
			t.Fatalf("expected anonymous ID %q, got %#v", expectedAnonymousId, event["anonymousId"])
		}
		if ip := event["context"].(map[string]any)["ip"]; ip != expectedIP {
			t.Fatalf("expected IP %q, got %#v", expectedIP, ip)
		}
		if event["event"] != expectedEvent {
			t.Fatalf("expected event %q, got %#v", expectedEvent, event["event"])
		}
		if !reflect.DeepEqual(event["properties"], expectedProperties) {
			t.Fatalf("expected properties %#v, got %#v", expectedProperties, event["properties"])
		}
		if connection, err := strconv.Atoi(string(event["connectionId"].(json.Number))); err != nil || connection != javaScriptID {
			t.Fatalf("expected connection %d, got %#v", javaScriptID, event["connectionId"])
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

	// Test importing an identity with a pipeline that has no mapping.
	javaScript2ID := c.CreateJavaScriptSource("JavaScript (source 2)", nil)
	javaScript2Key := c.EventWriteKeys(javaScript2ID)[0]
	c.CreatePipeline(javaScript2ID, "User", krenalistester.PipelineToSet{
		Name:    "JavaScript",
		Enabled: true,
	})
	c.SendEvent(javaScript2Key, analytics.Identify{
		UserId:      "Zny0kLMyz",
		AnonymousId: "bd857fe0-8f62-4d36-8e47-0161db0cc513",
	})

	c.WaitConnectionIdentitiesStoredIntoWarehouse(ctx, javaScript2ID, 1)
}
