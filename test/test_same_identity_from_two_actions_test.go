//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestSameIdentityFromTwoActions(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	dummy := c.AddDummy("Dummy", chichitester.Source)

	// Import the "first_name" property from the first action.
	action1 := c.AddAction(dummy, "Users", chichitester.ActionToSet{
		Name: "Import users (1)",
		InSchema: types.Object([]types.Property{
			{Name: "firstName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.Text()},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"first_name": "firstName",
			},
		},
	})

	// Import the "last_name" property from the second action: this will create
	// separated identities that refer to the same "identity" - from the app's
	// point of view.
	action2 := c.AddAction(dummy, "Users", chichitester.ActionToSet{
		Name: "Import users (2)",
		InSchema: types.Object([]types.Property{
			{Name: "lastName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "last_name", Type: types.Text()},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"last_name": "lastName",
			},
		},
	})

	c.ExecuteAction(dummy, action1, false)
	c.ExecuteAction(dummy, action2, false)
	c.WaitActionsToFinish(dummy)

	// Check that there are 10 users.
	users, _, count := c.Users([]string{"first_name", "last_name"}, "first_name", 0, 100)
	if count != 10 {
		t.Fatalf("expected 10 users, got %d", count)
	}
	user := users[0]
	if user.Properties["first_name"] != "Ariela" {
		t.Fatalf("expected first name %q, got %q", "Ariela", user.Properties["first_name"])
	}

	// Check that there are 20 user identities in total.
	identities, count := c.ConnectionIdentities(dummy, 0, 100)
	if count != 20 {
		t.Fatalf("expected 20 identities, got %d", count)
	}

	// Make sure both actions appear 10 times, respectively each among all
	// identities imported by this connection.
	action1Count, action2Count := 0, 0
	for _, identity := range identities {
		switch identity.Action {
		case action1:
			action1Count++
		case action2:
			action2Count++
		default:
			t.Fatalf("unexpected identity action %d", identity.Action)
		}
	}
	if action1Count != 10 {
		t.Fatalf("expected 10 identities with action %d, got %d", action1, action1Count)
	}
	if action2Count != 10 {
		t.Fatalf("expected 10 identities with action %d, got %d", action2, action2Count)
	}

}
