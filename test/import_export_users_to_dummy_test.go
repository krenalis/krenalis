//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"testing"

	"chichi/apis"
	"chichi/connector"
	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestImportExportUsersToDummy(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	c.SetWorkspaceIdentifiers([]string{"email"}, apis.AnonymousIdentifiers{})

	// Load some users in the data warehouse.
	{
		dummySrc := c.AddDummy("Dummy (source)", connector.Source)
		importUsersID := c.AddAction(dummySrc, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Import users from Dummy",
				"InSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
				}),
				"Transformation": map[string]any{
					"Mapping": map[string]string{
						"email":      "email",
						"first_name": "first_name",
					},
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

	// Export the users to Dummy.
	{
		dummyDest := c.AddDummy("Dummy (destination)", connector.Destination)
		exportUsersActionID := c.AddAction(dummyDest, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Export users to Dummy",
				"InSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				"Transformation": map[string]any{
					"Mapping": map[string]string{
						"email":     "email",
						"last_name": "email", // this is intended.
					},
				},
				"ExportMode": "CreateOrUpdate",
				"MatchingProperties": map[string]any{
					"Internal": "email",
					"External": types.Property{
						Name: "email",
						Type: types.Text(),
					},
				},
			},
		})
		c.ExecuteAction(dummyDest, exportUsersActionID, true)
		c.WaitActionsToFinish(dummyDest)
	}

	// Import from Dummy - again - to check if the users have been updated
	// successfully.
	{
		dummySrc := c.AddDummy("Dummy (source 2)", connector.Source)
		importUsersID := c.AddAction(dummySrc, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Import users from Dummy",
				"InSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				"Transformation": map[string]any{
					"Mapping": map[string]string{
						"email":      "email",
						"first_name": "first_name",
						"last_name":  "last_name",
					},
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
		users := c.Users([]string{"email", "first_name", "last_name"}, "", 0, 100)["users"].([]any)
		if len(users) == 0 {
			t.Fatal("no users re-imported from Dummy")
		}
		for _, user := range users {
			u := user.(map[string]any)
			if u["email"] != u["last_name"] {
				t.Fatalf("expecting 'email' to be equal to 'last_name', got '%v' != '%v'", u["email"], u["last_name"])
			}
		}
	}

}
