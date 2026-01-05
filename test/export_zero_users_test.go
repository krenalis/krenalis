// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestExportZeroProfiles(t *testing.T) {

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

	// Test the export of zero profiles to an application (Dummy).
	func() {
		dummyDest := c.CreateDummy("Dummy (destination)", meergotester.Destination)
		exportProfilesPipelineID := c.CreatePipeline(dummyDest, "User", meergotester.PipelineToSet{
			Name:    "Export profiles to Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"lastName": "last_name",
				},
			},
			ExportMode: meergotester.CreateOrUpdate,
			Matching: meergotester.Matching{
				In:  "email",
				Out: "email",
			},
			UpdateOnDuplicates: false,
		})
		run := c.RunPipeline(exportProfilesPipelineID)
		c.WaitRunsCompletion(dummyDest, run)
	}()

	// Test the export of zero profiles to file (CSV).
	func() {

		// Create the File System connection.
		fsID := c.CreateConnection(meergotester.ConnectionToCreate{
			Name:      "File System",
			Role:      meergotester.Destination,
			Connector: "filesystem",
			Settings: meergotester.JSONEncodeSettings(map[string]any{
				"root": storage.Root(),
			}),
		})

		exportedFilename := "exported-profiles.tmp.csv"
		exportFilePath := filepath.Join(storage.Root(), exportedFilename)

		// Create a pipeline for the File System for exporting the profiles.
		exportProfilesPipelineID := c.CreatePipeline(fsID, "User", meergotester.PipelineToSet{
			Name:    "Export profiles to the CSV on File System",
			Enabled: true,
			Path:    exportedFilename,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "gender", Type: types.String(), ReadOptional: true},
			}),
			Format:  "csv",
			OrderBy: "email",
			FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
				"separator": ",",
			}),
		})

		// Remove the export file, if exists.
		err := os.Remove(exportFilePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("PUT", "/v1/connections/"+strconv.Itoa(fsID), map[string]any{
			"name":        "Storage",
			"compression": core.NoCompression,
		}, nil)

		// Run the pipeline that export profiles.
		run := c.RunPipeline(exportProfilesPipelineID)

		// Wait for the export to finish.
		c.WaitRunsCompletion(fsID, run)

		// Check if the file has been created successfully.
		fi, err := os.Open(exportFilePath)
		if err != nil {
			t.Fatal(err)
		}
		var r io.Reader = fi

		content, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}

		const expected = "email,first_name,last_name,gender\n"

		if !bytes.Equal(content, []byte(expected)) {
			t.Fatalf("file content not matching expected content. Expected %q, got %q", expected, string(content))
		}

		// The test completed successfully, so the storage can be removed.
		storage.Remove()
	}()

}
