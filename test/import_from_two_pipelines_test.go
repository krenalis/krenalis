// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestImportUsersFromFileWithTwoPipelines(t *testing.T) {

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

	// Create the first pipeline for the CSV for importing "email" and "name".
	pipelineFirstName := c.CreatePipeline(fsID, "User", meergotester.PipelineToSet{
		Name:    "Import users' email and name from CSV on File System",
		Enabled: true,
		Path:    "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.String()},
			{Name: "name", Type: types.String()},
			{Name: "email", Type: types.String()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
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

	// Create the second pipeline for the CSV for importing "email" and "lastName".
	pipelineLastName := c.CreatePipeline(fsID, "User", meergotester.PipelineToSet{
		Name:    "Import users' email and lastName from CSV on File System",
		Enabled: true,
		Path:    "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.String()},
			{Name: "lastname", Type: types.String()},
			{Name: "email", Type: types.String()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
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

	// Import from the first pipeline, which should import just the first name.
	exec := c.ExecutePipeline(pipelineFirstName)
	c.WaitForExecutionsCompletion(fsID, exec)

	// Check the profiles.
	assertEq := func(msg string, expected, got any) {
		if !reflect.DeepEqual(expected, got) {
			t.Fatalf("%s: expected value %#v, got %#v", msg, expected, got)
		}
		t.Logf("%s: value %#v matches the expected value", msg, expected)
	}
	const expectedTotal = 2
	profiles, _, total := c.Profiles([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if total != expectedTotal {
		t.Fatalf("expected a total of %d profiles, got %d", expectedTotal, total)
	}
	assertEq("run #1: first  profile email", "luigi.rossi@example.com", profiles[0].Attributes["email"])
	assertEq("run #1: first  profile first name", "Luigi", profiles[0].Attributes["first_name"])
	assertEq("run #1: first  profile last name", nil, profiles[0].Attributes["last_name"])
	assertEq("run #1: second profile email", "mario.rossi@example.com", profiles[1].Attributes["email"])
	assertEq("run #1: second profile first name", "Mario", profiles[1].Attributes["first_name"])
	assertEq("run #1: second profile last name", nil, profiles[1].Attributes["last_name"])

	// Import from the second pipeline, which should import just the last name,
	// and that should result in profiles with both first name and last name.
	exec = c.ExecutePipeline(pipelineLastName)
	c.WaitForExecutionsCompletion(fsID, exec)

	// Check the profiles.
	profiles, _, total = c.Profiles([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if total != expectedTotal {
		t.Fatalf("expected a total of %d profiles, got %d", expectedTotal, total)
	}
	assertEq("run #2: first  profile email", "luigi.rossi@example.com", profiles[0].Attributes["email"])
	assertEq("run #2: first  profile first name", "Luigi", profiles[0].Attributes["first_name"])
	assertEq("run #2: first  profile last name", "Bianchi", profiles[0].Attributes["last_name"])
	assertEq("run #2: second profile email", "mario.rossi@example.com", profiles[1].Attributes["email"])
	assertEq("run #2: second profile first name", "Mario", profiles[1].Attributes["first_name"])
	assertEq("run #2: second profile last name", "Rossi", profiles[1].Attributes["last_name"])
}
