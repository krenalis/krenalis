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

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestImportUsersFromFile(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
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

	c.ChangeIdentityResolutionSettings(true, []string{"email"})

	// Add an action to the CSV for importing the users.
	importUsersActionID := c.AddAction(fsID, "Users", meergotester.ActionToSet{
		Name: "Import users from CSV on Filesystem",
		Path: "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "name", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
			Mapping: map[string]string{
				"first_name": "name",
				"email":      "email",
			},
		},
		IdentityProperty: "identity",
		Format:           "CSV",
		UIValues: meergotester.JSONEncodeUIValues(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	// Execute the action that imports users.
	exec := c.ExecuteAction(fsID, importUsersActionID, true)

	// Wait for the import to finish.
	c.WaitForExecutionsCompletion(fsID, exec)

	// Retrieve the users and test them.
	const (
		expectedCount    = 2
		expectedUsersLen = 2
	)
	users, _, count := c.Users([]string{"email"}, "", false, 0, 100)
	usersLen := len(users)
	if usersLen != expectedUsersLen {
		t.Fatalf("expected %d users, got %d", expectedUsersLen, usersLen)
	}
	if count != expectedCount {
		t.Fatalf("expected \"count\" to be %d, got %d", expectedCount, count)
	}

	// Retrieve the user identities and test them.
	identities, count := c.ConnectionIdentities(fsID, 0, 100)
	if count != 2 {
		t.Fatalf("expected 2 user identities, got %d", count)
	}
	for _, identity := range identities {
		if identity.Connection != fsID {
			t.Fatalf("expected connection %d, got %d", fsID, identity.Connection)
		}
		if identity.Action != importUsersActionID {
			t.Fatalf("expected action %d, got %d", importUsersActionID, identity.Action)
		}
		if len(identity.AnonymousIds) != 0 {
			t.Fatalf("expected zero anonymous ID for the identity, got %v", identity.AnonymousIds)
		}
	}

}
