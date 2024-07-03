//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestImportExportUsersToDummy(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	c.SetWorkspaceIdentifiers([]string{"email"})

	// Load some users in the data warehouse.
	{
		dummySrc := c.AddDummy("Dummy (source)", chichitester.Source)
		importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
			Name: "Import users from Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "first_name", Type: types.Text()},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":      "email",
					"first_name": "firstName",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

	// Export the users to Dummy.
	{
		dummyDest := c.AddDummy("Dummy (destination)", chichitester.Destination)
		exportUsersActionID := c.AddAction(dummyDest, "Users", chichitester.ActionToSet{
			Name: "Export users to Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":    "email",
					"lastName": "email", // this is intended.
				},
			},
			ExportMode: chichitester.ExportModeCreateOrUpdate,
			MatchingProperties: &chichitester.MatchingProperties{
				Internal: "email",
				External: types.Property{
					Name: "email",
					Type: types.Text(),
				},
			},
			ExportOnDuplicatedUsers: &[]bool{false}[0],
		})
		c.ExecuteAction(dummyDest, exportUsersActionID, true)
		c.WaitActionsToFinish(dummyDest)
	}

	// Import from Dummy - again - to check if the users have been updated
	// successfully.
	{
		dummySrc := c.AddDummy("Dummy (source 2)", chichitester.Source)
		importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
			Name: "Import users from Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "first_name", Type: types.Text()},
				{Name: "last_name", Type: types.Text()},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":      "email",
					"first_name": "firstName",
					"last_name":  "lastName",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
		users, _, _ := c.Users([]string{"email", "first_name", "last_name"}, "", 0, 100)
		if len(users) == 0 {
			t.Fatal("no users re-imported from Dummy")
		}
		for _, user := range users {
			if user.Properties["email"] != user.Properties["last_name"] {
				t.Fatalf("expecting 'email' to be equal to 'last_name', got '%v' != '%v'", user.Properties["email"], user.Properties["last_name"])
			}
		}
	}

}
