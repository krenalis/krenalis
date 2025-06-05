//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"
	"time"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func Test_RemoveUsersWhenDeletingConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Create two Dummy connections for importing users.
	dummy1 := c.CreateDummy("Dummy 1", meergotester.Source)
	dummy2 := c.CreateDummy("Dummy 2", meergotester.Source)

	// Create two identical actions for two different connections.
	actionParams := meergotester.ActionToSet{
		Enabled: true,
		Name:    "Import users from Dummy",
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

	// Now there should be total of 20 users.
	_, _, total := c.Users([]string{"email"}, "", false, 0, 100)
	if total != 20 {
		t.Fatalf("expected 20 users, got %d", total)
	}

	// Delete one Dummy, wait for the identities to be purged, resolve
	// identities, and ensure that only 10 users remain.
	c.DeleteConnection(dummy1)
	time.Sleep(time.Second)
	c.RunIdentityResolution()
	_, _, total = c.Users([]string{"email"}, "", false, 0, 100)
	if total != 10 {
		t.Fatalf("expected 10 users, got %d", total)
	}

	// Delete also the other Dummy connection; now the total number of users
	// should be zero.
	c.DeleteConnection(dummy2)
	time.Sleep(time.Second)
	c.RunIdentityResolution()
	_, _, total = c.Users([]string{"email"}, "", false, 0, 100)
	if total != 0 {
		t.Fatalf("expected no users, got %d", total)
	}

}
