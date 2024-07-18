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
	dummy1 := c.AddDummy("Dummy 1", meergotester.Source)
	dummy2 := c.AddDummy("Dummy 2", meergotester.Source)

	// Add two identical actions on two different connections.
	actionParams := meergotester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
			{Name: "lastName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
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
	_, _, count := c.Users([]string{"email"}, "", false, 0, 100)
	if count != 20 {
		t.Fatalf("expected 20 users, got %d", count)
	}

	// Delete one Dummy, wait for the identities to be purged, run Identity Resolution,
	// and ensure that only 10 users remain.
	c.DeleteConnection(dummy1)
	time.Sleep(time.Second)
	c.RunIdentityResolution()
	_, _, count = c.Users([]string{"email"}, "", false, 0, 100)
	if count != 10 {
		t.Fatalf("expected 10 users, got %d", count)
	}

	// Delete also the other Dummy connection; now the total count of users
	// should be zero.
	c.DeleteConnection(dummy2)
	time.Sleep(time.Second)
	c.RunIdentityResolution()
	_, _, count = c.Users([]string{"email"}, "", false, 0, 100)
	if count != 0 {
		t.Fatalf("expected no users, got %d", count)
	}

}
