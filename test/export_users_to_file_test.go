//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"chichi/apis"
	"chichi/connector"
	"chichi/connector/types"
	"chichi/test/chichitester"

	"github.com/golang/snappy"
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
				"InSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "Email", Type: types.Text()},
					{Name: "FirstName", Type: types.Text()},
					{Name: "LastName", Type: types.Text()},
					{Name: "Gender", Type: types.Text().WithEnum([]string{"male", "female", "other"})},
				}),
				"Identifiers": []string{"Email"},
				"Mapping": map[string]string{
					"Email":     "coalesce(email, 'default.email@example.com')",
					"FirstName": "first_name",
					"LastName":  "last_name",
					"Gender":    "'male'",
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

	exportedFilename := "exported-users.tmp.csv"
	exportFilePath := filepath.Join(storageDir, exportedFilename)

	// Add an action to the CSV for exporting the users.
	exportUsersActionID := c.AddAction(csvID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name": "Export users to the CSV on Filesystem",
			"Path": exportedFilename,
		},
	})

	compressions := []apis.Compression{
		apis.NoCompression,
		apis.ZipCompression,
		apis.GzipCompression,
		apis.SnappyCompression,
	}

	for _, compression := range compressions {

		t.Logf("[info] export %s compressed file", compression)

		var ext string
		switch compression {
		case apis.ZipCompression:
			ext = ".zip"
		case apis.GzipCompression:
			ext = ".gz"
		case apis.SnappyCompression:
			ext = ".sz"
		}

		// Remove the export file, if exists.
		err = os.Remove(exportFilePath + ext)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("POST", "/api/connections/"+strconv.Itoa(csvID)+"/storage", map[string]any{
			"Storage":     fsID,
			"Compression": compression,
		})

		// Execute the action that export users.
		c.ExecuteAction(csvID, exportUsersActionID, true)

		// Wait for the import to finish.
		c.WaitActionsToFinish(csvID)

		// Check if the file has been created successfully.
		fi, err := os.Open(exportFilePath + ext)
		if err != nil {
			t.Fatal(err)
		}
		var r io.Reader = fi
		switch compression {
		case apis.ZipCompression:
			st, err := fi.Stat()
			if err != nil {
				t.Fatal(err)
			}
			zr, err := zip.NewReader(fi, st.Size())
			if err != nil {
				t.Fatal(err)
			}
			r, err = zr.Open(exportedFilename)
			if err != nil {
				t.Fatal(err)
			}
		case apis.GzipCompression:
			r, err = gzip.NewReader(fi)
			if err != nil {
				t.Fatal(err)
			}
		case apis.SnappyCompression:
			r = snappy.NewReader(fi)
		}
		content, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}

		expectedStrings := []string{
			"id,creation_time,timestamp,dummy_id,FirstName,LastName,Email,Gender,FoodPreferences,PhoneNumbers,FavouriteMovie",
			`Janifer,Sharpin,jsharpin8@example.com,male,"{""Drink"":null,""Fruit"":null}",,`,
		}
		for _, expected := range expectedStrings {
			if !bytes.Contains(content, []byte(expected)) {
				t.Fatalf("string %q not found in file %q", expected, exportFilePath)
			}
		}

	}

}
