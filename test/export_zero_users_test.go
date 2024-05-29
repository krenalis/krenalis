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

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestExportZeroUsers(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	c.SetWorkspaceIdentifiers([]string{"email"})

	// Test the export of zero users to an app (Dummy).
	func() {
		dummyDest := c.AddDummy("Dummy (destination)", chichitester.Destination)
		exportUsersActionID := c.AddAction(dummyDest, "Users", chichitester.ActionToSet{
			Name: "Export users to Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "last_name", Type: types.Text(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":    "email",
					"lastName": "last_name",
				},
			},
			ExportMode: chichitester.ExportModeCreateOrUpdate,
			MatchingProperties: &chichitester.MatchingProperties{
				Internal: "email",
				External: types.Property{
					Name: "email",
					Type: types.Text(),
				},
			},
			ExportOnDuplicatedUsers: &[]bool{false}[0],
		})
		c.ExecuteAction(dummyDest, exportUsersActionID, true)
		c.WaitActionsToFinish(dummyDest)
	}()

	// Test the export of zero users to file (CSV).
	func() {

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

		// Add an action to the Filesystem for exporting the users.
		exportUsersActionID := c.AddAction(fsID, "Users", chichitester.ActionToSet{
			Name: "Export users to the CSV on Filesystem",
			Path: exportedFilename,
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "first_name", Type: types.Text(), Nullable: true},
				{Name: "last_name", Type: types.Text(), Nullable: true},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other"), Nullable: true},
			}),
			Connector: "CSV",
			UIValues: chichitester.JSONEncodeUIValues(map[string]any{
				"Comma": ",",
			}),
			FileOrderingPropertyPath: "email",
		})

		// Remove the export file, if exists.
		err := os.Remove(exportFilePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		c.MustCall("PUT", "/api/workspaces/1/connections/"+strconv.Itoa(fsID), map[string]any{
			"Connection": map[string]any{
				"Name":        "Storage",
				"Enabled":     true,
				"Compression": apis.NoCompression,
			},
		}, nil)

		// Execute the action that export users.
		c.ExecuteAction(fsID, exportUsersActionID, true)

		// Wait for the import to finish.
		c.WaitActionsToFinish(fsID)

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
