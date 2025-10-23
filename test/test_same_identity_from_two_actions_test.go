//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func TestSameIdentityFromTwoActions(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Prevents Identity Resolution from running automatically and ensures there
	// are no identifiers.
	c.UpdateIdentityResolution(false, nil)

	dummy := c.CreateDummy("Dummy", meergotester.Source)

	// Import the "first_name" property from the first action.
	action1 := c.CreateAction(dummy, "User", meergotester.ActionToSet{
		Name:    "Import users (1)",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "firstName", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"first_name": "firstName",
			},
		},
	})

	// Import the "last_name" property from the second action: this will create
	// separated identities that refer to the same "identity" - from the API's
	// point of view.
	action2 := c.CreateAction(dummy, "User", meergotester.ActionToSet{
		Name:    "Import users (2)",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "lastName", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"last_name": "lastName",
			},
		},
	})

	// Executes the two actions and waits for them to complete.
	exec1 := c.ExecuteAction(action1)
	exec2 := c.ExecuteAction(action2)
	c.WaitForExecutionsCompletion(dummy, exec1, exec2)

	// Run the Identity Resolution and wait for its completion.
	c.RunIdentityResolution()

	// Check that there are 10 users.
	users, _, total := c.Users([]string{"first_name", "last_name"}, "first_name", false, 0, 100)
	if total != 10 {
		t.Fatalf("expected 10 users, got %d", total)
	}
	user := users[0]
	if user.Traits["first_name"] != "Ariela" {
		t.Fatalf("expected first name %q, got %q", "Ariela", user.Traits["first_name"])
	}

	// Check that there are 20 user identities in total.
	identities, total := c.ConnectionIdentities(dummy, 0, 100)
	if total != 20 {
		t.Fatalf("expected 20 identities, got %d", total)
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
