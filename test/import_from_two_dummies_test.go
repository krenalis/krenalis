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

func TestImportFromTwoDummies(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Create two Dummy connections for importing users.
	dummy1 := c.CreateDummy("Dummy 1", meergotester.Source)
	dummy2 := c.CreateDummy("Dummy 2", meergotester.Source)

	// Create two identical actions for two different connections.
	actionParams := meergotester.ActionToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "firstName", Type: types.Text(), Nullable: true},
			{Name: "lastName", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
				"last_name":  "lastName",
			},
		},
	}
	action1 := c.CreateAction(dummy1, "User", actionParams)
	action2 := c.CreateAction(dummy2, "User", actionParams)

	// Import from both actions - and implicitly trigger the identity resolution
	// process.
	exec1 := c.ExecuteAction(action1)
	exec2 := c.ExecuteAction(action2)
	c.WaitForExecutionsCompletion(dummy1, exec1)
	c.WaitForExecutionsCompletion(dummy2, exec2)

	// Ensure that the connection have the correct identities associated.
	{
		identities, total := c.ConnectionIdentities(dummy1, 0, 100)
		if total != 10 {
			t.Fatalf("expected total 10, got %d", total)
		}
		for _, identity := range identities {
			if identity.Action != action1 {
				t.Fatalf("expected action %d, got %d, ", action1, identity.Action)
			}
		}
		identities, total = c.ConnectionIdentities(dummy2, 0, 100)
		if total != 10 {
			t.Fatalf("expected total 10, got %d", total)
		}
		for _, identity := range identities {
			if identity.Action != action2 {
				t.Fatalf("expected action %d, got %d", action2, identity.Action)
			}
		}
	}

	// Since the users have been imported from two different connections without
	// any identity resolution identifier configured, there should be a total of
	// 20 users, even if they have the same properties.
	users, _, total := c.Users([]string{"email", "first_name", "last_name"}, "", false, 0, 100)
	expectedTotal := 20
	if expectedTotal != total {
		t.Fatalf("expected total %d, got %d", expectedTotal, total)
	}

	// Every user now should have just one identity associated.
	totalUsers := 0
	for _, user := range users {
		_, total := c.UserIdentities(user.ID, 0, 100)
		const expectedTotal = 1
		if expectedTotal != total {
			t.Fatalf("expected %d identities for user %s, got %d", total, user.ID, total)
		}
		totalUsers++
	}
	if expectedTotal != totalUsers { // ensure that the number of users matches with the returned 'total' value.
		t.Fatalf("expected %d users returned, got %d", expectedTotal, totalUsers)
	}

	// Update the workspace identifiers and run the Identity Resolution.
	c.UpdateIdentityResolution(true, []string{"email"})
	c.RunIdentityResolution()

	// Now the users should be merged, resulting in a total of 10 users.
	users, _, total = c.Users([]string{"email", "first_name", "last_name"}, "", false, 0, 100)
	expectedTotal = 10
	if expectedTotal != total {
		t.Fatalf("expected total %d, got %d", expectedTotal, total)
	}

	// Every user now should have two identities associated.
	totalUsers = 0
	for _, user := range users {
		_, total := c.UserIdentities(user.ID, 0, 100)
		const expectedTotal = 2
		if expectedTotal != total {
			t.Fatalf("expected %d identities for user %s, got %d", total, user.ID, total)
		}
		totalUsers++
	}
	if expectedTotal != totalUsers { // ensure that the total number of users matches with the returned 'total' value.
		t.Fatalf("expected %d users returned, got %d", expectedTotal, totalUsers)
	}

}
