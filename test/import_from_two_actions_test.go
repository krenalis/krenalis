// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func TestImportUsersFromFileWithTwoActions(t *testing.T) {

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

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	// Create the File System connection.
	fsID := c.CreateSourceFileSystem()

	// Create the first action for the CSV for importing "email" and "name".
	actionFirstName := c.CreateAction(fsID, "User", meergotester.ActionToSet{
		Name:    "Import users' email and name from CSV on File System",
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
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"first_name": "name",
				"email":      "email",
			},
		},
		IdentityColumn: "identity",
		Format:         "csv",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]interface{}{
			"Separator":      ",",
			"HasColumnNames": true,
		}),
	})

	// Create the second action for the CSV for importing "email" and "lastName".
	actionLastName := c.CreateAction(fsID, "User", meergotester.ActionToSet{
		Name:    "Import users' email and lastName from CSV on File System",
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
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"last_name": "lastname",
				"email":     "email",
			},
		},
		IdentityColumn: "identity",
		Format:         "csv",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]interface{}{
			"Separator":      ",",
			"HasColumnNames": true,
		}),
	})

	// Import from the first action, which should import just the first name.
	exec := c.ExecuteAction(actionFirstName)
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
	assertEq("run #1: second user last name", nil, users[1].Traits["last_name"])

	// Import from the second action, which should import just the last name,
	// and that should result in users with both first name and last name.
	exec = c.ExecuteAction(actionLastName)
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
