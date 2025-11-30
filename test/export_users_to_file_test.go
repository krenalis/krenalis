// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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
	"github.com/meergo/meergo/tools/types"

	"github.com/klauspost/compress/snappy"
)

func TestExportUsersToFile(t *testing.T) {

	// Create the temporary storage.
	storage := meergotester.NewTempStorage(t)

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storage.Root())
	c.Start()
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	// Load some users in the data warehouse.
	{
		dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
		importUsersID := c.CreatePipeline(dummySrc, "User", meergotester.PipelineToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
				{Name: "firstName", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithCharLen(300), ReadOptional: true},
				{Name: "gender", Type: types.String(), ReadOptional: true},
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
		exec := c.ExecutePipeline(importUsersID)
		c.WaitForExecutionsCompletion(dummySrc, exec)
	}

	// Create the File System connection.
	fsID := c.CreateConnection(meergotester.ConnectionToCreate{
		Name:      "File System",
		Role:      meergotester.Destination,
		Connector: "filesystem",
		Settings: meergotester.JSONEncodeSettings(map[string]any{
			"SimulateHighIOLatency": false,
		}),
	})

	exportedFilename := "exported-profiles.tmp.csv"
	exportFilePath := filepath.Join(storage.Root(), exportedFilename)

	// Create a pipeline for the CSV for exporting the users.
	exportUsersPipelineID := c.CreatePipeline(fsID, "User", meergotester.PipelineToSet{
		Name:    "Export users to the CSV on File System",
		Enabled: true,
		Path:    exportedFilename,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.String().WithCharLen(300), ReadOptional: true},
			{Name: "gender", Type: types.String(), ReadOptional: true},
		}),
		Format: "csv",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Separator": ",",
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

		if compression == core.NoCompression {
			t.Logf("[info] exporting file without compression")
		} else {
			t.Logf("[info] exporting file with compression %q", compression)
		}

		// Remove the export file, if exists.
		err := os.Remove(exportFilePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("PUT", "/api/v1/pipelines/"+strconv.Itoa(exportUsersPipelineID), meergotester.PipelineToSet{
			Name:    "Export users to the CSV on File System",
			Enabled: true,
			Path:    exportedFilename,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithCharLen(300), ReadOptional: true},
				{Name: "gender", Type: types.String(), ReadOptional: true},
			}),
			Format: "csv",
			FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
				"Separator": ",",
			}),
			Compression: meergotester.Compression(compression),
			OrderBy:     "email",
		}, nil)

		// Execute the pipeline that export users.
		exec := c.ExecutePipeline(exportUsersPipelineID)

		// Wait for the export to finish.
		c.WaitForExecutionsCompletion(fsID, exec)

		// Check if the file has been created successfully.
		fi, err := os.Open(exportFilePath)
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
		err = fi.Close()
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
