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

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"

	"github.com/klauspost/compress/snappy"
)

func TestExportUsersToFile(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	// Load some users in the data warehouse.
	{
		dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
		importUsersID := c.CreateAction(dummySrc, "Users", meergotester.ActionToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "firstName", Type: types.Text(), Nullable: true},
				{Name: "lastName", Type: types.Text(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "gender", Type: types.Text(), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":      "coalesce(email, 'default.email@example.com')",
					"first_name": "firstName",
					"last_name":  "lastName",
					"gender":     "'male'",
				},
			},
		})
		exec := c.ExecuteAction(importUsersID)
		c.WaitForExecutionsCompletion(dummySrc, exec)
	}

	// Create the temporary storage.
	storage := meergotester.NewTempStorage(t)

	// Create the Filesystem connection.
	fsID := c.CreateConnection(meergotester.ConnectionToCreate{
		Name:      "Filesystem",
		Role:      meergotester.Destination,
		Connector: "Filesystem",
		Settings: meergotester.JSONEncodeSettings(map[string]any{
			"Root": storage.Root(),
		}),
	})

	exportedFilename := "exported-users.tmp.csv"
	exportFilePath := filepath.Join(storage.Root(), exportedFilename)

	// Create an action for the CSV for exporting the users.
	exportUsersActionID := c.CreateAction(fsID, "Users", meergotester.ActionToSet{
		Name:    "Export users to the CSV on Filesystem",
		Enabled: true,
		Path:    exportedFilename,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "gender", Type: types.Text(), ReadOptional: true},
		}),
		Format: "CSV",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Comma": ",",
		}),
		OrderBy: "email",
	})

	compressions := []core.Compression{
		core.NoCompression,
		core.ZipCompression,
		core.GzipCompression,
		core.SnappyCompression,
	}

	for _, compression := range compressions {

		t.Logf("[info] export %s compressed file", compression)

		var ext string
		switch compression {
		case core.ZipCompression:
			ext = ".zip"
		case core.GzipCompression:
			ext = ".gz"
		case core.SnappyCompression:
			ext = ".sz"
		}

		// Remove the export file, if exists.
		err := os.Remove(exportFilePath + ext)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("PUT", "/api/v1/actions/"+strconv.Itoa(exportUsersActionID), meergotester.ActionToSet{
			Name:    "Export users to the CSV on Filesystem",
			Enabled: true,
			Path:    exportedFilename,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "gender", Type: types.Text(), ReadOptional: true},
			}),
			Format: "CSV",
			FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
				"Comma": ",",
			}),
			Compression: meergotester.Compression(compression),
			OrderBy:     "email",
		}, nil)

		// Execute the action that export users.
		exec := c.ExecuteAction(exportUsersActionID)

		// Wait for the import to finish.
		c.WaitForExecutionsCompletion(fsID, exec)

		// Check if the file has been created successfully.
		fi, err := os.Open(exportFilePath + ext)
		if err != nil {
			t.Fatal(err)
		}
		var r io.Reader = fi
		switch compression {
		case core.ZipCompression:
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
		case core.GzipCompression:
			r, err = gzip.NewReader(fi)
			if err != nil {
				t.Fatal(err)
			}
		case core.SnappyCompression:
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
			t.Fatalf("expected content %q, got %q", expected, string(content))
		}

	}

	// The test completed successfully, so the storage can be removed.
	storage.Remove()

}
