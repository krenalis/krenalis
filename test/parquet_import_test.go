//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
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
	c.PopulateUserSchema(false)
	c.SetFilesystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	// Change the user schema, leaving only the properties used by this test.
	userSchemaProperties := []types.Property{}
	userSchemaProperties = append(userSchemaProperties, types.Property{
		Name:         "parquet_id",
		Type:         types.Int(64),
		ReadOptional: true,
	})
	userSchemaProperties = append(userSchemaProperties, types.Property{
		Name:         "parquet_imported",
		Type:         types.JSON(),
		ReadOptional: true,
	})
	c.AlterUserSchema(types.Object(userSchemaProperties), nil, nil)

	// Create a Filesystem source connection, with an action that imports from the Parquet file.
	fs := c.CreateSourceFilesystem()
	action1 := c.CreateAction(fs, "User", meergotester.ActionToSet{
		Name:    "Parquet",
		Enabled: true,
		Path:    "test.parquet",
		InSchema: types.Object([]types.Property{
			{Name: "parquet_id", Type: types.Int(64), Nullable: true},
			{Name: "first_name", Type: types.Text(), Nullable: true},
			{Name: "last_name", Type: types.Text(), Nullable: true},
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
		// "parquet_id" (it is used for sorting the users before comparing them)
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
		Format:         "Parquet",
	})

	// Import and wait.
	exec1 := c.ExecuteAction(action1)
	c.WaitForExecutionsCompletionAllowFailed(fs, exec1)

	// Check that the count of users imported from the file is correct.
	users, _, count := c.Users([]string{"parquet_imported"}, "parquet_id", false, 0, 1000)
	if count != len(expectedUsers) {
		t.Fatalf("expected %d user(s), got %d", len(expectedUsers), count)
	}

	// Check that the users properties imported in Meergo match the user
	// properties in the Parquet file.
	var fail bool
	for i := range users {
		gotTraits := users[i].Traits["parquet_imported"].(map[string]any)
		expectedTraits := expectedUsers[i]
		if !reflect.DeepEqual(gotTraits, expectedTraits) {
			t.Errorf("users[%d]: expected traits %#v, got %#v", i, expectedTraits, gotTraits)
			fail = true
		}
	}
	if fail {
		t.Fatal("users do not match")
	}

}

var expectedUsers = []map[string]any{
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
