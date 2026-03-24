// Copyright 2026 Open2b. All rights reserved.
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

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/klauspost/compress/snappy"
)

func TestExportUsersToFile(t *testing.T) {

	// Create the temporary storage.
	storage := krenalistester.NewTempStorage(t)

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.SetFileSystemRoot(storage.Root())
	c.Start()
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	// Load some users in the data warehouse.
	{
		dummySrc := c.CreateDummy("Dummy (source)", krenalistester.Source)
		importUsersID := c.CreatePipeline(dummySrc, "User", krenalistester.PipelineToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
				{Name: "firstName", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "gender", Type: types.String(), ReadOptional: true},
			}),
			Transformation: &krenalistester.Transformation{
				Mapping: map[string]string{
					"email":      "coalesce(email, 'default.email@example.com')",
					"first_name": "firstName",
					"last_name":  "lastName",
					"gender":     "'male'",
				},
			},
		})
		run := c.RunPipeline(importUsersID)
		c.WaitRunsCompletion(dummySrc, run)
	}

	// Create the File System connection.
	fsID := c.CreateConnection(krenalistester.ConnectionToCreate{
		Name:      "File System",
		Role:      krenalistester.Destination,
		Connector: "filesystem",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{
			"simulateHighIOLatency": false,
		}),
	})

	exportedFilename := "exported-profiles.tmp.csv"
	exportFilePath := filepath.Join(storage.Root(), exportedFilename)

	// Create a pipeline for the CSV for exporting the users.
	exportUsersPipelineID := c.CreatePipeline(fsID, "User", krenalistester.PipelineToSet{
		Name:    "Export users to the CSV on File System",
		Enabled: true,
		Path:    exportedFilename,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "gender", Type: types.String(), ReadOptional: true},
		}),
		Format: "csv",
		FormatSettings: krenalistester.JSONEncodeSettings(map[string]any{
			"separator": ",",
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

		c.MustCall("PUT", "/v1/pipelines/"+strconv.Itoa(exportUsersPipelineID), krenalistester.PipelineToSet{
			Name:    "Export users to the CSV on File System",
			Enabled: true,
			Path:    exportedFilename,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "gender", Type: types.String(), ReadOptional: true},
			}),
			Format: "csv",
			FormatSettings: krenalistester.JSONEncodeSettings(map[string]any{
				"separator": ",",
			}),
			Compression: krenalistester.Compression(compression),
			OrderBy:     "email",
		}, nil)

		// Run the pipeline that export users.
		run := c.RunPipeline(exportUsersPipelineID)

		// Wait for the export to finish.
		c.WaitRunsCompletion(fsID, run)

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
