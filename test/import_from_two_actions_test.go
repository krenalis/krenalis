//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestImportUsersFromFileWithTwoActions(t *testing.T) {

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

	// Add the first action to the CSV for importing "email" and "name".
	actionFirstName := c.AddAction(fsID, "Users", chichitester.ActionToSet{
		Name: "Import users' email and name from CSV on Filesystem",
		Path: "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "name", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "firstName", Type: types.Text(), Nullable: true},
			{Name: "email", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"firstName": "name",
				"email":     "email",
			},
		},
		IdentityProperty: "identity",
		Connector:        chichitester.CSVConnector,
		UIValues: chichitester.JSONEncodeUIValues(map[string]interface{}{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	// Add the second action to the CSV for importing "email" and "lastName".
	actionLastName := c.AddAction(fsID, "Users", chichitester.ActionToSet{
		Name: "Import users' email and lastName from CSV on Filesystem",
		Path: "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "lastname", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "lastName", Type: types.Text(), Nullable: true},
			{Name: "email", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"lastName": "lastname",
				"email":    "email",
			},
		},
		IdentityProperty: "identity",
		Connector:        chichitester.CSVConnector,
		UIValues: chichitester.JSONEncodeUIValues(map[string]interface{}{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	// Import from the first action, which should import just the firstName.
	c.ExecuteAction(fsID, actionFirstName, false)
	c.WaitActionsToFinish(fsID)

	// Check the users.
	assertEq := func(msg string, expected, got any) {
		if !reflect.DeepEqual(expected, got) {
			t.Fatalf("%s: expected value %#v, got %#v", msg, expected, got)
		}
		t.Logf("%s: value %#v matches the expected value", msg, expected)
	}
	const expectedCount = 2
	users, _, count := c.Users([]string{"email", "firstName", "lastName"}, "email", 0, 2)
	if count != expectedCount {
		t.Fatalf("expecting a total of %d users, got %d", expectedCount, count)
	}
	assertEq("run #1: first  user email", "mario.rossi@example.com", users[0]["email"])
	assertEq("run #1: first  user firstName", "Mario", users[0]["firstName"])
	assertEq("run #1: first  user last name", nil, users[0]["lastName"])
	assertEq("run #1: second user email", "luigi.rossi@example.com", users[1]["email"])
	assertEq("run #1: second user firstName", "Luigi", users[1]["firstName"])
	assertEq("run #1: second user last name", nil, users[0]["lastName"])

	// Import from the second action, which should import just the lastName, and
	// that should result in users with both firstName and lastName.
	c.ExecuteAction(fsID, actionLastName, false)
	c.WaitActionsToFinish(fsID)

	// Check the users.
	users, _, count = c.Users([]string{"email", "firstName", "lastName"}, "email", 0, 2)
	if count != expectedCount {
		t.Fatalf("expecting a total of %d users, got %d", expectedCount, count)
	}
	assertEq("run #2: first  user email", "mario.rossi@example.com", users[0]["email"])
	assertEq("run #2: first  user firstName", "Mario", users[0]["firstName"])
	assertEq("run #2: first  user last name", "Rossi", users[0]["lastName"])
	assertEq("run #2: second user email", "luigi.rossi@example.com", users[1]["email"])
	assertEq("run #2: second user firstName", "Luigi", users[1]["firstName"])
	assertEq("run #2: second user last name", "Bianchi", users[1]["lastName"])
}
