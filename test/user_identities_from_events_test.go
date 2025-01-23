//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"context"
	"testing"
	"time"

	"github.com/meergo/meergo/test/analytics-go"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestUserIdentitiesFromEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	javaScriptID := c.CreateJavaScriptSource("JavaScript (source)", "example.com", nil)
	javaScriptKey := c.EventWriteKeys(javaScriptID)[0]
	c.CreateAction(javaScriptID, "Events", meergotester.ActionToSet{
		Name:    "JavaScript events",
		Enabled: true,
	})
	importUsersAction := c.CreateAction(javaScriptID, "Users", meergotester.ActionToSet{
		Name:     "JavaScript users",
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

	ctx := context.Background()

	const eventUserEmail = "event-user@example.com"
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "f4ca124298",
		Traits: map[string]interface{}{
			"email": eventUserEmail,
		},
		Context: &analytics.Context{
			Device: analytics.DeviceInfo{
				Id: "MY-DEVICE-ID-1234",
			},
		},
	})
	c.WaitEventsStoredIntoWarehouse(ctx, 1)
	time.Sleep(time.Second)
	c.StartIdentityResolution()

	// Retrieve the user imported from the event.
	users, _, total := c.Users([]string{"email"}, "", false, 0, 100)
	if 1 != total {
		t.Fatalf("expected one user, got %d", total)
	}
	found := false
	for _, user := range users {
		email, _ := user.Traits["email"].(string)
		if email == eventUserEmail {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("user with email %q not found", eventUserEmail)
	}

	// Update the action to import identities through a constant mapping.
	c.UpdateAction(importUsersAction, meergotester.ActionToSet{
		Name:     "JavaScript users",
		Enabled:  true,
		InSchema: types.Type{},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "'a@b'", // a constant email for every user
			},
		},
	})

	// Send an event identify and wait for the event to be stored in the
	// warehouse.
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "uT8VT5tx1A",
		Traits: map[string]interface{}{
			"email": eventUserEmail,
		},
	})
	c.WaitEventsStoredIntoWarehouse(ctx, 2)
	time.Sleep(time.Second)
	c.StartIdentityResolution()

	// Check that the user has been created.
	_, _, total = c.Users([]string{"email"}, "", false, 0, 100)
	if total != 2 {
		t.Fatalf("expected 2 users, got %d", total)
	}

	// Update the action to import identities through a transformation function.
	c.UpdateAction(importUsersAction, meergotester.ActionToSet{
		Name:     "JavaScript users",
		Enabled:  true,
		InSchema: types.Type{},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Function: &meergotester.TransformationFunction{
				Source: `import random

def transform(event: dict) -> dict:
	return {
		"email": event["userId"],
	}`,
				Language: "Python",
				InPaths:  []string{"userId"},
				OutPaths: []string{"email"},
			},
		},
	})

	// Send an event identify and wait for the event to be stored in the
	// warehouse.
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId: "Kw5vKdDYBQ",
		Traits: map[string]interface{}{
			"email": eventUserEmail,
		},
	})
	c.WaitEventsStoredIntoWarehouse(ctx, 3)
	time.Sleep(time.Second)
	c.StartIdentityResolution()

	// Check that the user has been created.
	_, _, total = c.Users([]string{"email"}, "", false, 0, 100)
	if total != 3 {
		t.Fatalf("expected 3 users, got %d", total)
	}

}
