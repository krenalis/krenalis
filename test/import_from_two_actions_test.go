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

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestImportUsersFromFileWithTwoActions(t *testing.T) {

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
	fsID := c.CreateSourceFilesystem(storageDir)

	// Create the first action for the CSV for importing "email" and "name".
	actionFirstName := c.CreateAction(fsID, "Users", meergotester.ActionToSet{
		Name:    "Import users' email and name from CSV on Filesystem",
		Enabled: true,
		Path:    "users.csv",
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
		FormatSettings: meergotester.JSONEncodeSettings(map[string]interface{}{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	// Create the second action for the CSV for importing "email" and "lastName".
	actionLastName := c.CreateAction(fsID, "Users", meergotester.ActionToSet{
		Name:    "Import users' email and lastName from CSV on Filesystem",
		Enabled: true,
		Path:    "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "lastname", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
			Mapping: map[string]string{
				"last_name": "lastname",
				"email":     "email",
			},
		},
		IdentityProperty: "identity",
		Format:           "CSV",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]interface{}{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	// Import from the first action, which should import just the first name.
	exec := c.ExecuteAction(actionFirstName, true)
	c.WaitForExecutionsCompletion(fsID, exec)

	// Check the users.
	assertEq := func(msg string, expected, got any) {
		if !reflect.DeepEqual(expected, got) {
			t.Fatalf("%s: expected value %#v, got %#v", msg, expected, got)
		}
		t.Logf("%s: value %#v matches the expected value", msg, expected)
	}
	const expectedTotal = 2
	users, _, total := c.Users([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if total != expectedTotal {
		t.Fatalf("expected a total of %d users, got %d", expectedTotal, total)
	}
	assertEq("run #1: first  user email", "luigi.rossi@example.com", users[0].Traits["email"])
	assertEq("run #1: first  user first name", "Luigi", users[0].Traits["first_name"])
	assertEq("run #1: first  user last name", nil, users[0].Traits["last_name"])
	assertEq("run #1: second user email", "mario.rossi@example.com", users[1].Traits["email"])
	assertEq("run #1: second user first name", "Mario", users[1].Traits["first_name"])
	assertEq("run #1: second user last name", nil, users[0].Traits["last_name"])

	// Import from the second action, which should import just the last name,
	// and that should result in users with both first name and last name.
	exec = c.ExecuteAction(actionLastName, true)
	c.WaitForExecutionsCompletion(fsID, exec)

	// Check the users.
	users, _, total = c.Users([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if total != expectedTotal {
		t.Fatalf("expected a total of %d users, got %d", expectedTotal, total)
	}
	assertEq("run #2: first  user email", "luigi.rossi@example.com", users[0].Traits["email"])
	assertEq("run #2: first  user first name", "Luigi", users[0].Traits["first_name"])
	assertEq("run #2: first  user last name", "Bianchi", users[0].Traits["last_name"])
	assertEq("run #2: second user email", "mario.rossi@example.com", users[1].Traits["email"])
	assertEq("run #2: second user first name", "Mario", users[1].Traits["first_name"])
	assertEq("run #2: second user last name", "Rossi", users[1].Traits["last_name"])
}
