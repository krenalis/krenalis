//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"testing"

	"chichi/test/chichitester"
	"chichi/types"
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
		dummySrc := c.AddDummy("Dummy (source)", chichitester.Source, "")
		importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
			Name: "Import users from Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "firstName", Type: types.Text(), Nullable: true},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":     "email",
					"firstName": "firstName",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

	// Export the users to Dummy.
	{
		dummyDest := c.AddDummy("Dummy (destination)", chichitester.Destination, "")
		exportUsersActionID := c.AddAction(dummyDest, "Users", chichitester.ActionToSet{
			Name: "Export users to Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
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
		dummySrc := c.AddDummy("Dummy (source 2)", chichitester.Source, "")
		importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
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
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
		users, _, _ := c.Users([]string{"email", "firstName", "lastName"}, "", 0, 100)
		if len(users) == 0 {
			t.Fatal("no users re-imported from Dummy")
		}
		for _, user := range users {
			if user["email"] != user["lastName"] {
				t.Fatalf("expecting 'email' to be equal to 'lastName', got '%v' != '%v'", user["email"], user["lastName"])
			}
		}
	}

}
