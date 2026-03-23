// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"context"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/meergo/analytics-go"
)

func TestIdentitiesFromEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	javaScriptID := c.CreateJavaScriptSource("JavaScript (source)", nil)
	javaScriptKey := c.EventWriteKeys(javaScriptID)[0]
	c.CreatePipeline(javaScriptID, "Event", krenalistester.PipelineToSet{
		Name:    "JavaScript events",
		Enabled: true,
	})
	importUsersPipeline := c.CreatePipeline(javaScriptID, "User", krenalistester.PipelineToSet{
		Name:     "JavaScript users",
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

	ctx := context.Background()

	const eventProfileEmail = "event-profile@example.com"
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "f4ca124298",
		Traits: map[string]any{
			"email": eventProfileEmail,
		},
		Context: &analytics.Context{
			Device: analytics.DeviceInfo{
				Id: "MY-DEVICE-ID-1234",
			},
		},
	})
	c.WaitEventsStoredIntoWarehouse(ctx, 1)
	time.Sleep(time.Second)
	c.RunIdentityResolution()

	// Retrieve the profile imported from the event.
	profiles, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 1 {
		t.Fatalf("expected one profile, got %d", total)
	}
	found := false
	for _, profile := range profiles {
		email, _ := profile.Attributes["email"].(string)
		if email == eventProfileEmail {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("profile with email %q not found", eventProfileEmail)
	}

	// Update the pipeline to import identities through a constant mapping.
	c.UpdatePipeline(importUsersPipeline, krenalistester.PipelineToSet{
		Name:     "JavaScript users",
		Enabled:  true,
		InSchema: types.Type{},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email": "'a@b'", // a constant email for every user
			},
		},
	})

	// Send an event identify and wait for the event to be stored in the
	// warehouse.
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "uT8VT5tx1A",
		Traits: map[string]any{
			"email": eventProfileEmail,
		},
	})
	c.WaitEventsStoredIntoWarehouse(ctx, 2)
	time.Sleep(time.Second)
	c.RunIdentityResolution()

	// Check that the profile has been created.
	_, _, total = c.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 2 {
		t.Fatalf("expected 2 profiles, got %d", total)
	}

	// Update the pipeline to import identities through a transformation function.
	c.UpdatePipeline(importUsersPipeline, krenalistester.PipelineToSet{
		Name:     "JavaScript users",
		Enabled:  true,
		InSchema: types.Type{},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Function: &krenalistester.TransformationFunction{
				Language: "Python",
				Source: `import random

def transform(event: dict) -> dict:
	return {
		"email": event["userId"],
	}`,
				InPaths:  []string{"userId"},
				OutPaths: []string{"email"},
			},
		},
	})

	// Send an event identify and wait for the event to be stored in the
	// warehouse.
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "Kw5vKdDYBQ",
		Traits: map[string]any{
			"email": eventProfileEmail,
		},
	})
	c.WaitEventsStoredIntoWarehouse(ctx, 3)
	time.Sleep(time.Second)
	c.RunIdentityResolution()

	// Check that the profile has been created.
	_, _, total = c.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 3 {
		t.Fatalf("expected 3 profiles, got %d", total)
	}

}
