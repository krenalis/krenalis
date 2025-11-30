// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"path/filepath"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestSourceFileStorageUsersFiltering(t *testing.T) {

	// Determine the storage directory.
	storageDir, err := filepath.Abs("testdata/source_file_storage_users_filtering")
	if err != nil {
		t.Fatal(err)
	}

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	fs1 := c.CreateSourceFileSystem()

	pipeline1 := c.CreatePipeline(fs1, "User", meergotester.PipelineToSet{
		Name:    "CSV",
		Enabled: true,
		Path:    "users.csv",
		Filter: &meergotester.Filter{
			Logical: meergotester.OpAnd,
			Conditions: []meergotester.FilterCondition{
				{
					Property: "email",
					Operator: meergotester.OpIsNot,
					Values:   []string{"et@example.com"},
				},
			},
		},
		InSchema: types.Object([]types.Property{
			{Name: "CSV_id", Type: types.String()},
			{Name: "email", Type: types.String()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityColumn: "CSV_id",
		Format:         "csv",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Separator":      ",",
			"HasColumnNames": true,
		}),
	})

	exec1 := c.ExecutePipeline(pipeline1)

	c.WaitForExecutionsCompletionAllowFailed(fs1, exec1)

	_, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)

	// The CSV file contains 10 profiles, but one of them was filtered out, so
	// there must be 9.
	const expectedTotal = 9
	if expectedTotal != total {
		t.Fatalf("expected %d profiles, got %d", expectedTotal, total)
	}
	t.Logf("the APIs successfully returned %d profiles", total)
}
