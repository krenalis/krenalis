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

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"

	"github.com/segmentio/analytics-go/v3"
)

func TestUserIdentitiesFromEvents(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	var javaScriptKey string
	{
		javaScriptID := c.AddJavaScriptSource("JavaScript (source)", "example.com", nil)
		keys := c.ConnectionKeys(javaScriptID)
		if len(keys) != 1 {
			t.Fatalf("expecting one key, got %d keys", len(keys))
		}
		javaScriptKey = keys[0]
		c.AddAction(javaScriptID, "Events", chichitester.ActionToSet{
			Name:    "JavaScript events",
			Enabled: true,
		})
		c.AddAction(javaScriptID, "Users", chichitester.ActionToSet{
			Name:     "JavaScript users",
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

	ctx := context.Background()

	c.WaitEventsStoredIntoWarehouse(ctx, 1)

	// Retrieve the user imported from the event.
	users, _, count := c.Users([]string{"email"}, "", 0, 100)
	const expectedUsersCount = 1
	if expectedUsersCount != count {
		t.Fatalf("expecting %d user(s), got %d", expectedUsersCount, count)
	}
	found := false
	for _, user := range users {
		email, _ := user.Properties["email"].(string)
		if email == eventUserEmail {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("user with email %q not found", eventUserEmail)
	}

}
