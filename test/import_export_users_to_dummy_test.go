// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestImportExportUsersToDummy(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	// Load some users in the data warehouse.
	{
		dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
		importUsersID := c.CreatePipeline(dummySrc, "User", meergotester.PipelineToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
				{Name: "firstName", Type: types.String(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":      "email",
					"first_name": "firstName",
				},
			},
		})
		exec := c.ExecutePipeline(importUsersID)
		c.WaitForExecutionsCompletion(dummySrc, exec)
	}

	// Export the profiles to Dummy.
	{
		dummyDest := c.CreateDummy("Dummy (destination)", meergotester.Destination)
		exportProfilesPipelineID := c.CreatePipeline(dummyDest, "User", meergotester.PipelineToSet{
			Name:    "Export users to Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
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
			UpdateOnDuplicates: false,
		})
		exec := c.ExecutePipeline(exportProfilesPipelineID)
		c.WaitForExecutionsCompletion(dummyDest, exec)
	}

	// Import from Dummy - again - to check if the users have been updated
	// successfully.
	{
		dummySrc := c.CreateDummy("Dummy (source 2)", meergotester.Source)
		importUsersID := c.CreatePipeline(dummySrc, "User", meergotester.PipelineToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
				{Name: "firstName", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":      "email",
					"first_name": "firstName",
					"last_name":  "lastName",
				},
			},
		})
		exec := c.ExecutePipeline(importUsersID)
		c.WaitForExecutionsCompletion(dummySrc, exec)
		profiles, _, _ := c.Profiles([]string{"email", "first_name", "last_name"}, "", false, 0, 100)
		if len(profiles) == 0 {
			t.Fatal("no profiles re-imported from Dummy")
		}
		for _, user := range profiles {
			if user.Attributes["email"] != user.Attributes["last_name"] {
				t.Fatalf("expected 'email' to be equal to 'last_name', got '%v' != '%v'", user.Attributes["email"], user.Attributes["last_name"])
			}
		}
	}

}
