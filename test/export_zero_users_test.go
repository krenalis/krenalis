// Copyright 2025 Open2b. All rights reserved.
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
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func TestExportZeroUsers(t *testing.T) {

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

	// Test the export of zero users to an API (Dummy).
	func() {
		dummyDest := c.CreateDummy("Dummy (destination)", meergotester.Destination)
		exportUsersActionID := c.CreateAction(dummyDest, "User", meergotester.ActionToSet{
			Name:    "Export users to Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "lastName", Type: types.Text(), Nullable: true},
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
		exec := c.ExecuteAction(exportUsersActionID)
		c.WaitForExecutionsCompletion(dummyDest, exec)
	}()

	// Test the export of zero users to file (CSV).
	func() {

		// Create the File System connection.
		fsID := c.CreateConnection(meergotester.ConnectionToCreate{
			Name:      "File System",
			Role:      meergotester.Destination,
			Connector: "filesystem",
			Settings: meergotester.JSONEncodeSettings(map[string]any{
				"Root": storage.Root(),
			}),
		})

		exportedFilename := "exported-users.tmp.csv"
		exportFilePath := filepath.Join(storage.Root(), exportedFilename)

		// Create an action for the File System for exporting the users.
		exportUsersActionID := c.CreateAction(fsID, "User", meergotester.ActionToSet{
			Name:    "Export users to the CSV on File System",
			Enabled: true,
			Path:    exportedFilename,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "gender", Type: types.Text(), ReadOptional: true},
			}),
			Format:  "csv",
			OrderBy: "email",
			FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
				"Separator": ",",
			}),
		})

		// Remove the export file, if exists.
		err := os.Remove(exportFilePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("PUT", "/api/v1/connections/"+strconv.Itoa(fsID), map[string]any{
			"name":        "Storage",
			"compression": core.NoCompression,
		}, nil)

		// Execute the action that export users.
		exec := c.ExecuteAction(exportUsersActionID)

		// Wait for the import to finish.
		c.WaitForExecutionsCompletion(fsID, exec)

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
