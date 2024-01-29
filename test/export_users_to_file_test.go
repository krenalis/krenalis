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

	c.SetWorkspaceIdentifiers([]string{"email"}, apis.AnonymousIdentifiers{})

	// Load some users in the data warehouse.
	{
		dummySrc := c.AddDummy("Dummy (source)", chichitester.Source)
		importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
			Name: "Import users from Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":     "coalesce(email, 'default.email@example.com')",
					"firstName": "firstName",
					"lastName":  "lastName",
					"gender":    "'male'",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

	// Create the temporary storage.
	storage := chichitester.NewTempStorage(t)

	// Create the Filesystem connection.
	fsID := c.AddConnection(chichitester.ConnectionToAdd{
		Name:      "Filesystem",
		Role:      chichitester.Destination,
		Enabled:   true,
		Connector: 19, // Filesystem.
		Settings: chichitester.JSONEncodeSettings(map[string]any{
			"Root": storage.Root(),
		}),
	})

	// Create the CSV connection.
	csvID := c.AddConnection(chichitester.ConnectionToAdd{
		Name:      "CSV",
		Role:      chichitester.Destination,
		Enabled:   true,
		Connector: 5, // CSV.
		Storage:   fsID,
		Settings: chichitester.JSONEncodeSettings(map[string]any{
			"Comma": ",",
		}),
	})

	exportedFilename := "exported-users.tmp.csv"
	exportFilePath := filepath.Join(storage.Root(), exportedFilename)

	// Add an action to the CSV for exporting the users.
	exportUsersActionID := c.AddAction(csvID, "Users", chichitester.ActionToSet{
		Name: "Export users to the CSV on Filesystem",
		Path: exportedFilename,
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
			{Name: "lastName", Type: types.Text()},
			{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
		}),
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
		err := os.Remove(exportFilePath + ext)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("POST", "/api/workspaces/1/connections/"+strconv.Itoa(csvID), map[string]any{
			"Connection": map[string]any{
				"Name":        "CSV",
				"Enabled":     true,
				"Storage":     fsID,
				"Compression": compression,
			},
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

		expected := `email,firstName,lastName,gender
kbuessen0@example.com,Kinsley,Buessen,male
jdebrett9@example.com,Jerad,Debrett,male
emoakes2r@example.com,Edyth,Moakes,male
lwhitesonrr@example.com,Leann,Whiteson,male
sattestone2s@example.com,Susanne,Atte-Stone,male
aquittonden2t@example.com,Aimil,Quittonden,male
tbrayson2u@example.com,Teodora,Brayson,male
csifflett2v@example.com,Cristiano,Sifflett,male
mpordal2w@example.com,Mona,Pordal,male
aniece2x@example.com,Ashil,Niece,male` + "\n"

		if !bytes.EqualFold([]byte(expected), content) {
			t.Fatalf("expecting content %q, got %q", expected, string(content))
		}

	}

	// The test completed successfully, so the storage can be removed.
	storage.Remove()

}
