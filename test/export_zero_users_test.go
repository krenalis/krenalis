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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestExportZeroUsers(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	// Test the export of zero users to an app (Dummy).
	func() {
		dummyDest := c.CreateDummy("Dummy (destination)", meergotester.Destination)
		exportUsersActionID := c.CreateAction(dummyDest, "Users", meergotester.ActionToSet{
			Name:    "Export users to Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
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
			ExportOnDuplicates: false,
		})
		exec := c.ExecuteAction(exportUsersActionID)
		c.WaitForExecutionsCompletion(dummyDest, exec)
	}()

	// Test the export of zero users to file (CSV).
	func() {

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

		// Create an action for the Filesystem for exporting the users.
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
			Format:  "CSV",
			OrderBy: "email",
			FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
				"Comma": ",",
			}),
		})

		// Remove the export file, if exists.
		err := os.Remove(exportFilePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("PUT", "/api/connections/"+strconv.Itoa(fsID), map[string]any{
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
