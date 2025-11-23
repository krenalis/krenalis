// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"
	"time"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func Test_RemoveUsersWhenDeletingConnections(t *testing.T) {

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

	// Now there should be total of 20 profiles.
	_, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 20 {
		t.Fatalf("expected 20 profiles, got %d", total)
	}

	// Delete one Dummy, wait for the identities to be purged, resolve
	// identities, and ensure that only 10 profiles remain.
	c.DeleteConnection(dummy1)
	time.Sleep(time.Second)
	c.RunIdentityResolution()
	_, _, total = c.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 10 {
		t.Fatalf("expected 10 profiles, got %d", total)
	}

	// Delete also the other Dummy connection; now the total number of profiles
	// should be zero.
	c.DeleteConnection(dummy2)
	time.Sleep(time.Second)
	c.RunIdentityResolution()
	_, _, total = c.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 0 {
		t.Fatalf("expected no profiles, got %d", total)
	}

}
