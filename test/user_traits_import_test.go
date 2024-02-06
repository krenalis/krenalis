//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"context"
	"encoding/json"
	"testing"

	"chichi/connector/types"
	"chichi/test/chichitester"

	"github.com/segmentio/analytics-go/v3"
)

func TestUserTraitsImport(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	var websiteKey string
	{
		websiteID := c.AddWebsiteSource("Website (source)", "example.com")
		keys := c.ConnectionKeys(websiteID)
		if len(keys) != 1 {
			t.Fatalf("expecting one key, got %d keys", len(keys))
		}
		websiteKey = keys[0]
		c.AddAction(websiteID, "Users", chichitester.ActionToSet{
			Name:     "Website",
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
	c.SendEvent(websiteKey, analytics.Identify{
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
	response := c.Users([]string{"Id", "email"}, "", 0, 100)
	count, _ := response["count"].(json.Number).Int64()
	const expectedUsersCount = 1
	if expectedUsersCount != count {
		t.Fatalf("expecting %d user(s), got %d", expectedUsersCount, count)
	}
	var userGID int64
	for _, user := range response["users"].([]any) {
		email, _ := user.(map[string]any)["email"].(string)
		if email == eventUserEmail {
			userGID, _ = user.(map[string]any)["Id"].(json.Number).Int64()
			if userGID <= 0 {
				t.Fatalf("invalid user GID %d", userGID)
			}
			break
		}
	}

}
