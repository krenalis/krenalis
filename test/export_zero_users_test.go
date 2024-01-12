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

	"chichi/apis"
	"chichi/connector"
	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestExportZeroUsers(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	c.SetWorkspaceIdentifiers([]string{"email"}, apis.AnonymousIdentifiers{})

	// Test the export of zero users to an app (Dummy).
	func() {
		dummyDest := c.AddDummy("Dummy (destination)", connector.Destination)
		exportUsersActionID := c.AddAction(dummyDest, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Export users to Dummy",
				"InSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "lastName", Type: types.Text()},
				}),
				"OutSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "lastName", Type: types.Text()},
				}),
				"Transformation": map[string]any{
					"Mapping": map[string]string{
						"email":    "email",
						"lastName": "lastName",
					},
				},
				"ExportMode": "CreateOrUpdate",
				"MatchingProperties": map[string]any{
					"Internal": "email",
					"External": types.Property{
						Name: "email",
						Type: types.Text(),
					},
				},
			},
		})
		c.ExecuteAction(dummyDest, exportUsersActionID, true)
		c.WaitActionsToFinish(dummyDest)
	}()

	// Test the export of zero users to file (CSV).
	func() {

		// Create the temporary storage.
		storage := chichitester.NewTempStorage(t)

		// Create the Filesystem connection.
		fsID := c.AddConnection(map[string]any{
			"Connection": map[string]any{
				"Name":      "Filesystem",
				"Role":      "Destination",
				"Enabled":   true,
				"Connector": 19, // Filesystem.
				"Settings": map[string]any{
					"Root": storage.Root(),
				},
			},
		})

		// Create the CSV connection.
		csvID := c.AddConnection(map[string]any{
			"Connection": map[string]any{
				"Name":      "CSV",
				"Role":      "Destination",
				"Enabled":   true,
				"Connector": 5, // CSV.
				"Storage":   fsID,
				"Settings": map[string]any{
					"Comma": ",",
				},
			},
		})

		exportedFilename := "exported-users.tmp.csv"
		exportFilePath := filepath.Join(storage.Root(), exportedFilename)

		// Add an action to the CSV for exporting the users.
		exportUsersActionID := c.AddAction(csvID, map[string]any{
			"Target": "Users",
			"Action": map[string]any{
				"Name": "Export users to the CSV on Filesystem",
				"Path": exportedFilename,
				"OutSchema": types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "firstName", Type: types.Text()},
					{Name: "lastName", Type: types.Text()},
					{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
				}),
			},
		})

		// Remove the export file, if exists.
		err := os.Remove(exportFilePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("POST", "/api/workspaces/1/connections/"+strconv.Itoa(csvID), map[string]any{
			"Connection": map[string]any{
				"Name":        "CSV",
				"Enabled":     true,
				"Storage":     fsID,
				"Compression": apis.NoCompression,
			},
		})

		// Execute the action that export users.
		c.ExecuteAction(csvID, exportUsersActionID, true)

		// Wait for the import to finish.
		c.WaitActionsToFinish(csvID)

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

		const expected = "email,firstName,lastName,gender\n"

		if !bytes.Equal(content, []byte(expected)) {
			t.Fatalf("file content not matching expected content. Expected %q, got %q", expected, string(content))
		}

		// The test completed successfully, so the storage can be removed.
		storage.Remove()
	}()

}
