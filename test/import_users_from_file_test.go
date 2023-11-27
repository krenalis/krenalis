//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"os"
	"path/filepath"
	"testing"

	"chichi/apis"
	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestImportUsersFromFile(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Determine the storage directory and assert that such directory exists.
	storageDir, err := filepath.Abs("testdata/storage")
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(storageDir)
	if err != nil {
		t.Fatal(err)
	}
	if !stat.IsDir() {
		t.Fatalf("%q is not a dir", storageDir)
	}

	// Create the Filesystem connection.
	fsID := c.AddSourceFilesystem(storageDir)

	// Create the CSV connection.
	csvID := c.AddSourceCSV(fsID)

	c.SetWorkspaceIdentifiers([]string{"email"}, apis.AnonymousIdentifiers{})

	// Add an action to the CSV for importing the users.
	importUsersActionID := c.AddAction(csvID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name": "Import users from CSV on Filesystem",
			"Path": "users.csv",
			"InSchema": types.Object([]types.Property{
				{Name: "name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
			}),
			"OutSchema": types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
			}),
			"Transformation": map[string]any{
				"Mapping": map[string]string{
					"first_name": "name",
					"email":      "email",
				},
			},
			"IdentityColumn": "identity",
		},
	})

	// Execute the action that imports users.
	c.ExecuteAction(csvID, importUsersActionID, true)

	// Wait for the import to finish.
	c.WaitActionsToFinish(csvID)

	// Retrieve the users.
	ret := c.Users([]string{"email"}, 0, 100)
	count := int(ret["count"].(float64))
	if count != 2 {
		t.Fatalf("expecting %d users, got %d", 2, count)
	}

}
