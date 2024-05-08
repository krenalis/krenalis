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

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
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

	c.SetWorkspaceIdentifiers([]string{"email"})

	// Add an action to the CSV for importing the users.
	importUsersActionID := c.AddAction(fsID, "Users", chichitester.ActionToSet{
		Name: "Import users from CSV on Filesystem",
		Path: "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "name", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.Text(), Nullable: true},
			{Name: "email", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"first_name": "name",
				"email":      "email",
			},
		},
		IdentityProperty: "identity",
		Connector:        "CSV",
		UIValues: chichitester.JSONEncodeUIValues(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	// Execute the action that imports users.
	c.ExecuteAction(fsID, importUsersActionID, true)

	// Wait for the import to finish.
	c.WaitActionsToFinish(fsID)

	// Retrieve the users and test them.
	const (
		expectedCount    = 2
		expectedUsersLen = 2
	)
	users, _, count := c.Users([]string{"email"}, "", 0, 100)
	usersLen := len(users)
	if usersLen != expectedUsersLen {
		t.Fatalf("expecting %d users, got %d", expectedUsersLen, usersLen)
	}
	if count != expectedCount {
		t.Fatalf("expecting \"count\" to be %d, got %d", expectedCount, count)
	}

}
