//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestImportExportUsersToDummy(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	// Load some users in the data warehouse.
	{
		dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
		importUsersID := c.CreateAction(dummySrc, "Users", meergotester.ActionToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "firstName", Type: types.Text(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":      "email",
					"first_name": "firstName",
				},
			},
		})
		exec := c.ExecuteAction(importUsersID)
		c.WaitForExecutionsCompletion(dummySrc, exec)
	}

	// Export the users to Dummy.
	{
		dummyDest := c.CreateDummy("Dummy (destination)", meergotester.Destination)
		exportUsersActionID := c.CreateAction(dummyDest, "Users", meergotester.ActionToSet{
			Name:    "Export users to Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "lastName", Type: types.Text(), Nullable: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"lastName": "email", // this is intended.
				},
			},
			ExportMode: meergotester.CreateOrUpdate,
			Matching: meergotester.Matching{
				In:  "email",
				Out: "email",
			},
			ExportOnDuplicates: false,
		})
		exec := c.ExecuteAction(exportUsersActionID)
		c.WaitForExecutionsCompletion(dummyDest, exec)
	}

	// Import from Dummy - again - to check if the users have been updated
	// successfully.
	{
		dummySrc := c.CreateDummy("Dummy (source 2)", meergotester.Source)
		importUsersID := c.CreateAction(dummySrc, "Users", meergotester.ActionToSet{
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
		})
		exec := c.ExecuteAction(importUsersID)
		c.WaitForExecutionsCompletion(dummySrc, exec)
		users, _, _ := c.Users([]string{"email", "first_name", "last_name"}, "", false, 0, 100)
		if len(users) == 0 {
			t.Fatal("no users re-imported from Dummy")
		}
		for _, user := range users {
			if user.Traits["email"] != user.Traits["last_name"] {
				t.Fatalf("expected 'email' to be equal to 'last_name', got '%v' != '%v'", user.Traits["email"], user.Traits["last_name"])
			}
		}
	}

}
