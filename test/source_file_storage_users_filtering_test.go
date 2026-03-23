// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"path/filepath"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
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
	c := krenalistester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	fs1 := c.CreateSourceFileSystem()

	pipeline1 := c.CreatePipeline(fs1, "User", krenalistester.PipelineToSet{
		Name:    "CSV",
		Enabled: true,
		Path:    "users.csv",
		Filter: &krenalistester.Filter{
			Logical: krenalistester.OpAnd,
			Conditions: []krenalistester.FilterCondition{
				{
					Property: "email",
					Operator: krenalistester.OpIsNot,
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
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		UserIDColumn: "CSV_id",
		Format:       "csv",
		FormatSettings: krenalistester.JSONEncodeSettings(map[string]any{
			"separator":      ",",
			"hasColumnNames": true,
		}),
	})

	run1 := c.RunPipeline(pipeline1)

	c.WaitForRunsCompletionAllowFailed(fs1, run1)

	_, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)

	// The CSV file contains 10 profiles, but one of them was filtered out, so
	// there must be 9.
	const expectedTotal = 9
	if expectedTotal != total {
		t.Fatalf("expected %d profiles, got %d", expectedTotal, total)
	}
	t.Logf("the APIs successfully returned %d profiles", total)
}
