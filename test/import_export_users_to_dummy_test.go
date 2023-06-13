//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"testing"

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

	// Load some users in the data warehouse.
	{
		dummySrc := c.AddDummy("Dummy (source)", connector.SourceRole)
		importUsersID := c.AddAction(dummySrc, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Import users from Dummy",
				"InSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "Email", Type: types.Text()},
					{Name: "FirstName", Type: types.Text()},
				}),
				"Mapping": map[string]string{
					"Email":     "email",
					"FirstName": "first_name",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

	// Export the users to Dummy.
	{
		dummyDest := c.AddDummy("Dummy (destination)", connector.DestinationRole)
		exportUsersActionID := c.AddAction(dummyDest, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Export users to Dummy",
				"InSchema": types.Object([]types.Property{
					{Name: "Email", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				"Mapping": map[string]string{
					"email":     "Email",
					"last_name": "Email", // this is intended.
				},
				"ExportMode": "CreateOrUpdate",
				"MatchingProperties": map[string]string{
					"Internal": "Email",
					"External": "email",
				},
			},
		})
		c.ExecuteAction(dummyDest, exportUsersActionID, true)
		c.WaitActionsToFinish(dummyDest)
	}

	// Import from Dummy - again - to check if the users have been updated
	// successfully.
	{
		dummySrc := c.AddDummy("Dummy (source 2)", connector.SourceRole)
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
					{Name: "Email", Type: types.Text()},
					{Name: "FirstName", Type: types.Text()},
					{Name: "LastName", Type: types.Text()},
				}),
				"Mapping": map[string]string{
					"Email":     "email",
					"FirstName": "first_name",
					"LastName":  "last_name",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
		users := c.Users([]string{"Email", "FirstName", "LastName"}, 0, 100)["users"].([]any)
		if len(users) == 0 {
			t.Fatal("no users re-imported from Dummy")
		}
		for _, user := range users {
			u := user.([]any)
			email := u[0].(string)
			lastName := u[2].(string)
			if email != lastName {
				t.Fatalf("expecting Email to be equal to LastName, got %q != %q", email, lastName)
			}
		}
	}

}
