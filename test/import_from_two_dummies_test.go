//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"encoding/json"
	"testing"

	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestImportFromTwoDummies(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create two Dummy connections for importing users.
	dummy1 := c.AddDummy("Dummy 1", chichitester.Source)
	dummy2 := c.AddDummy("Dummy 2", chichitester.Source)

	// Add two identical actions on two different connections.
	actionParams := chichitester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
			{Name: "lastName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "firstName", Type: types.Text(), Nullable: true},
			{Name: "lastName", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"firstName": "firstName",
				"lastName":  "lastName",
			},
		},
	}
	action1 := c.AddAction(dummy1, "Users", actionParams)
	action2 := c.AddAction(dummy2, "Users", actionParams)

	// Import from both actions - and implicitly trigger the identity resolution
	// process.
	c.ExecuteAction(dummy1, action1, true)
	c.ExecuteAction(dummy2, action2, true)
	c.WaitActionsToFinish(dummy1)
	c.WaitActionsToFinish(dummy2)

	// Ensure that the connection have the correct identities associated.
	{
		identities, count := c.ConnectionIdentities(dummy1, 0, 100)
		if count != 10 {
			t.Fatalf("expected count 10, got %d", count)
		}
		for _, identity := range identities {
			if identity.Connection != dummy1 {
				t.Fatalf("unexpected connection %d, expecting %d", identity.Connection, dummy1)
			}
		}
		identities, count = c.ConnectionIdentities(dummy2, 0, 100)
		if count != 10 {
			t.Fatalf("expected count 10, got %d", count)
		}
		for _, identity := range identities {
			if identity.Connection != dummy2 {
				t.Fatalf("unexpected connection %d, expecting %d", identity.Connection, dummy2)
			}
		}
	}

	// Since the users have been imported from two different connections without
	// any identity resolution identifier configured, there should be a total of
	// 20 users, even if they have the same properties.
	users, _, count := c.Users([]string{"Id", "email", "firstName", "lastName"}, "Id", 0, 100)
	expectedCount := 20
	if expectedCount != count {
		t.Fatalf("expected count %d, got %d", expectedCount, count)
	}

	// Every user now should have just one identity associated.
	totalUsers := 0
	for _, user := range users {
		id, _ := user["Id"].(json.Number).Int64()
		_, count := c.UserIdentities(int(id), 0, 100)
		const expectedCount = 1
		if expectedCount != count {
			t.Fatalf("expecting %d identities for user %d, got %d", count, id, count)
		}
		totalUsers++
	}
	if expectedCount != totalUsers { // ensure that the number of users matches with the returned 'count' value.
		t.Fatalf("expecting %d users returned, got %d", expectedCount, totalUsers)
	}

	// Change the workspace identifiers and run the Workspace Identity
	// Resolution.
	c.SetWorkspaceIdentifiers([]string{"email"})
	c.RunWorkspaceIdentityResolution()

	// Now the users should be merged, resulting in a total of 10 users.
	users, _, count = c.Users([]string{"Id", "email", "firstName", "lastName"}, "Id", 0, 100)
	expectedCount = 10
	if expectedCount != count {
		t.Fatalf("expected count %d, got %d", expectedCount, count)
	}

	// Every user now should have two identities associated.
	totalUsers = 0
	for _, user := range users {
		id, _ := user["Id"].(json.Number).Int64()
		_, count := c.UserIdentities(int(id), 0, 100)
		const expectedCount = 2
		if expectedCount != count {
			t.Fatalf("expecting %d identities for user %d, got %d", count, id, count)
		}
		totalUsers++
	}
	if expectedCount != totalUsers { // ensure that the number of users matches with the returned 'count' value.
		t.Fatalf("expecting %d users returned, got %d", expectedCount, totalUsers)
	}

}
