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

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"

	"github.com/golang/snappy"
)

func TestExportUsersToFile(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	c.SetWorkspaceIdentifiers([]string{"email"})

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
				{Name: "first_name", Type: types.Text()},
				{Name: "last_name", Type: types.Text()},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":      "coalesce(email, 'default.email@example.com')",
					"first_name": "firstName",
					"last_name":  "lastName",
					"gender":     "'male'",
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
		Connector: "Filesystem",
		UIValues: chichitester.JSONEncodeUIValues(map[string]any{
			"Root": storage.Root(),
		}),
	})

	exportedFilename := "exported-users.tmp.csv"
	exportFilePath := filepath.Join(storage.Root(), exportedFilename)

	// Add an action to the CSV for exporting the users.
	exportUsersActionID := c.AddAction(fsID, "Users", chichitester.ActionToSet{
		Name: "Export users to the CSV on Filesystem",
		Path: exportedFilename,
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "first_name", Type: types.Text()},
			{Name: "last_name", Type: types.Text()},
			{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
		}),
		Connector: "CSV",
		UIValues: chichitester.JSONEncodeUIValues(map[string]any{
			"Comma": ",",
		}),
		FileOrderingPropertyPath: "email",
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

		c.MustCall("PUT", "/api/workspaces/1/connections/"+strconv.Itoa(fsID)+"/actions/"+strconv.Itoa(exportUsersActionID), chichitester.ActionToSet{
			Name: "Export users to the CSV on Filesystem",
			Path: exportedFilename,
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "first_name", Type: types.Text()},
				{Name: "last_name", Type: types.Text()},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
			}),
			Connector: "CSV",
			UIValues: chichitester.JSONEncodeUIValues(map[string]any{
				"Comma": ",",
			}),
			Compression:              chichitester.Compression(compression),
			FileOrderingPropertyPath: "email",
		}, nil)

		// Execute the action that export users.
		c.ExecuteAction(fsID, exportUsersActionID, true)

		// Wait for the import to finish.
		c.WaitActionsToFinish(fsID)

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

		expected := `email,first_name,last_name,gender
abenois2@example.com,Ariela,Benois,male
bdroghan5@example.com,Bryon,Droghan,male
ctroy7@example.com,Codie,Troy,male
cveschambes3@example.com,Conroy,Veschambes,male
gclother1@example.com,Glyn,Clother,male
jdebrett9@example.com,Jerad,Debrett,male
jsharpin8@example.com,Janifer,Sharpin,male
kbuessen0@example.com,Kinsley,Buessen,male
kdericut4@example.com,Kingsly,Dericut,male
kfellon6@example.com,Katine,Fellon,male` + "\n"

		if !bytes.EqualFold([]byte(expected), content) {
			t.Fatalf("expecting content %q, got %q", expected, string(content))
		}

	}

	// The test completed successfully, so the storage can be removed.
	storage.Remove()

}
