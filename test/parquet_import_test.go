//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestParquetImport(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	storageDir, err := filepath.Abs("./testdata/parquet_import_test")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(storageDir); err != nil {
		t.Fatal(err)
	}

	fs := c.CreateSourceFilesystem(storageDir)

	action1 := c.CreateAction(fs, "Users", meergotester.ActionToSet{
		Name:    "Parquet",
		Enabled: true,
		Path:    "test.parquet",
		InSchema: types.Object([]types.Property{
			{Name: "parquet_id", Type: types.Int(64), Nullable: true},
			{Name: "first_name", Type: types.Text(), Nullable: true},
			{Name: "last_name", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"first_name": "first_name",
				"last_name":  "last_name",
			},
		},
		IdentityColumn: "parquet_id",
		Format:         "Parquet",
	})

	exec1 := c.ExecuteAction(action1)

	c.WaitForExecutionsCompletionAllowFailed(fs, exec1)

	_, _, count := c.Users([]string{"first_name"}, "dummy_id", false, 0, 1000)
	const expectedCount = 9
	if count != expectedCount {
		t.Fatalf("expected %d user(s), got %d", expectedCount, count)
	}

}
