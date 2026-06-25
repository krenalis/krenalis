// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"context"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/krenalis/analytics-go"
)

func TestIdentitiesFromEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	// Disable automatic execution of Identity Resolution.
	k.UpdateIdentityResolution(false, nil)

	javaScriptID := k.CreateJavaScriptSource("JavaScript (source)", nil)
	javaScriptKey := k.EventWriteKeys(javaScriptID)[0]
	k.CreatePipeline(javaScriptID, "Event", krenalistester.PipelineToSet{
		Name:    "JavaScript events",
		Enabled: true,
	})
	importUsersPipeline := k.CreatePipeline(javaScriptID, "User", krenalistester.PipelineToSet{
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
	k.SendEvent(javaScriptKey, analytics.Identify{
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
	k.WaitEventsStoredIntoWarehouse(ctx, 1)
	k.WaitConnectionIdentitiesStoredIntoWarehouse(ctx, javaScriptID, 1)
	k.RunIdentityResolutionAndWait()

	// Retrieve the profile imported from the event.
	profiles, _, total := k.Profiles([]string{"email"}, "", false, 0, 100)
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
	k.UpdatePipeline(importUsersPipeline, krenalistester.PipelineToSet{
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
	k.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "uT8VT5tx1A",
		Traits: map[string]any{
			"email": eventProfileEmail,
		},
	})
	k.WaitEventsStoredIntoWarehouse(ctx, 2)
	k.WaitConnectionIdentitiesStoredIntoWarehouse(ctx, javaScriptID, 2)
	k.RunIdentityResolutionAndWait()

	// Check that the profile has been created.
	_, _, total = k.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 2 {
		t.Fatalf("expected 2 profiles, got %d", total)
	}

	// Update the pipeline to import identities through a transformation function.
	k.UpdatePipeline(importUsersPipeline, krenalistester.PipelineToSet{
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
	k.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "Kw5vKdDYBQ",
		Traits: map[string]any{
			"email": eventProfileEmail,
		},
	})
	k.WaitEventsStoredIntoWarehouse(ctx, 3)
	k.WaitConnectionIdentitiesStoredIntoWarehouse(ctx, javaScriptID, 3)
	k.RunIdentityResolutionAndWait()

	// Check that the profile has been created.
	_, _, total = k.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 3 {
		t.Fatalf("expected 3 profiles, got %d", total)
	}

}
