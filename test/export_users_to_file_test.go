//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"chichi/connector"
	"chichi/test/chichitester"
)

func TestExportUsersToFile(t *testing.T) {

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
				"Mapping": map[string]string{
					"Email":     "email",
					"FirstName": "first_name",
					"LastName":  "last_name",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

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
	exportedFilename := "exported-users.tmp.csv"
	exportFilePath := filepath.Join(storageDir, exportedFilename)

	// Remove the export file, if exists.
	err = os.Remove(exportFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	// Create the Filesystem connection.
	fsID := c.AddConnection(map[string]any{
		"Connector": 19, // Filesystem.
		"Role":      "Destination",
		"Options": map[string]any{
			"Name":    "Filesystem",
			"Enabled": true,
		},
		"Settings": map[string]any{
			"Root": storageDir,
		},
	})

	// Create the CSV connection.
	csvID := c.AddConnection(map[string]any{
		"Connector": 5, // CSV.
		"Role":      "Destination",
		"Options": map[string]any{
			"Name":    "CSV",
			"Enabled": true,
			"Storage": fsID,
		},
		"Settings": map[string]any{
			"Comma": ",",
		},
	})

	// Add an action to the CSV for exporting the users.
	exportUsersActionID := c.AddAction(csvID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name": "Export users to the CSV on Filesystem",
			"Path": exportedFilename,
		},
	})

	// Execute the action that imports users.
	c.ExecuteAction(csvID, exportUsersActionID, true)

	// Wait for the import to finish.
	c.WaitActionsToFinish(csvID)

	// Check if the file has been created successfully.
	content, err := os.ReadFile(exportFilePath)
	if err != nil {
		t.Fatal(err)
	}
	expectedStrings := []string{
		"id,creation_time,timestamp,FirstName,LastName,Email,Gender,FoodPreferences,PhoneNumbers,FavouriteMovie",
		"Janifer,Sharpin,jsharpin8@example.com,<nil>,map[Drink:<nil> Fruit:<nil>],<nil>,<nil>",
	}
	for _, expected := range expectedStrings {
		if !bytes.Contains(content, []byte(expected)) {
			t.Fatalf("string %q not found in file %q", expected, exportFilePath)
		}
	}

}
