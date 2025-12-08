// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestParquetImport(t *testing.T) {

	// Retrieve the storage directory that contains the Parquet file to import.
	storageDir, err := filepath.Abs("./testdata/parquet_import_test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storageDir); err != nil {
		t.Fatal(err)
	}

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.PopulateProfileSchema(false)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	// Change the profile schema, leaving only the properties used by this test.
	profileSchemaProperties := []types.Property{}
	profileSchemaProperties = append(profileSchemaProperties, types.Property{
		Name:         "parquet_id",
		Type:         types.Int(64),
		ReadOptional: true,
	})
	profileSchemaProperties = append(profileSchemaProperties, types.Property{
		Name:         "parquet_imported",
		Type:         types.JSON(),
		ReadOptional: true,
	})
	c.AlterProfileSchema(types.Object(profileSchemaProperties), nil, nil)

	// Create a File System source connection, with a pipeline that imports from the Parquet file.
	fs := c.CreateSourceFileSystem()
	pipeline1 := c.CreatePipeline(fs, "User", meergotester.PipelineToSet{
		Name:    "Parquet",
		Enabled: true,
		Path:    "test.parquet",
		InSchema: types.Object([]types.Property{
			{Name: "parquet_id", Type: types.Int(64), Nullable: true},
			{Name: "first_name", Type: types.String(), Nullable: true},
			{Name: "last_name", Type: types.String(), Nullable: true},
			{Name: "date_of_birth", Type: types.Date(), Nullable: true},
			{Name: "updated_at", Type: types.DateTime(), Nullable: true},
			{Name: "lunch_time", Type: types.Time(), Nullable: true},
			{Name: "score", Type: types.Decimal(9, 5), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "parquet_id", Type: types.Int(64), ReadOptional: true},
			{Name: "parquet_imported", Type: types.JSON(), ReadOptional: true},
		}),
		// Define a transformation function that imports the Parquet row IDs in
		// "parquet_id" (it is used for sorting the profiles before comparing them)
		// and any other property into the JSON property "parquet_imported".
		Transformation: &meergotester.Transformation{
			Function: &meergotester.TransformationFunction{
				Language: "Python",
				Source: strings.Join([]string{
					`def transform(user: dict) -> dict:`,
					`    return {`,
					`        "parquet_id": user["parquet_id"],`,
					`        "parquet_imported": {k: v for k, v in user.items() if v is not None}`,
					`    }`,
				}, "\n"),
				InPaths:  []string{"parquet_id", "first_name", "last_name", "date_of_birth", "updated_at", "lunch_time", "score"},
				OutPaths: []string{"parquet_id", "parquet_imported"},
			},
		},
		IdentityColumn: "parquet_id",
		Format:         "parquet",
	})

	// Import and wait.
	run1 := c.RunPipeline(pipeline1)
	c.WaitForRunsCompletionAllowFailed(fs, run1)

	// Check that the count of profiles imported from the file is correct.
	profiles, _, count := c.Profiles([]string{"parquet_imported"}, "parquet_id", false, 0, 1000)
	if count != len(expectedProfiles) {
		t.Fatalf("expected %d user(s), got %d", len(expectedProfiles), count)
	}

	// Check that the profiles properties imported in Meergo match the user
	// properties in the Parquet file.
	var fail bool
	for i := range profiles {
		gotAttributes := profiles[i].Attributes["parquet_imported"].(map[string]any)
		expectedAttributes := expectedProfiles[i]
		if !reflect.DeepEqual(gotAttributes, expectedAttributes) {
			t.Errorf("profiles[%d]: expected attributes %#v, got %#v", i, expectedAttributes, gotAttributes)
			fail = true
		}
	}
	if fail {
		t.Fatal("profiles do not match")
	}

}

var expectedProfiles = []map[string]any{
	{"parquet_id": json.Number("100"), "first_name": "John", "last_name": "Lemon"},
	{"parquet_id": json.Number("101"), "first_name": "Ringo", "last_name": "Planett"},
	{"parquet_id": json.Number("102")},
	{"parquet_id": json.Number("103")},
	{"parquet_id": json.Number("104"), "date_of_birth": "1980-01-02"},
	{"parquet_id": json.Number("105"), "date_of_birth": "1935-01-02"},
	{"parquet_id": json.Number("106"), "updated_at": "2012-01-20 07:20:01"},
	{"parquet_id": json.Number("107"), "lunch_time": "13:30:00"},
	{"parquet_id": json.Number("108"), "score": "-1234.56789"},
}
