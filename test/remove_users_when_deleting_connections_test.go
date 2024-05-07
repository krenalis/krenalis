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

func Test_RemoveUsersWhenDeletingConnections(t *testing.T) {

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
			{Name: "first_name", Type: types.Text(), Nullable: true},
			{Name: "last_name", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
				"last_name":  "lastName",
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

	// Now there should be total of 20 users.
	_, _, count := c.Users([]string{"__id__", "email"}, "__id__", 0, 100)
	if count != 20 {
		t.Fatalf("expected 20 users, got %d", count)
	}

	// Delete one Dummy, run the Workspace Identity Resolution and ensure that
	// only 10 users are left.
	c.DeleteConnection(dummy1)
	c.RunWorkspaceIdentityResolution()
	_, _, count = c.Users([]string{"__id__", "email"}, "__id__", 0, 100)
	if count != 10 {
		t.Fatalf("expected 10 users, got %d", count)
	}

	// Delete also the other Dummy connection; now the total count of users
	// should be zero.
	c.DeleteConnection(dummy2)
	c.RunWorkspaceIdentityResolution()
	_, _, count = c.Users([]string{"__id__", "email"}, "__id__", 0, 100)
	if count != 0 {
		t.Fatalf("expected no users, got %d", count)
	}

}
