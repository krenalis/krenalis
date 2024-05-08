//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestStorage(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create a file storage connection.
	storageDir, err := filepath.Abs("testdata/storage")
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(storageDir)
	if err != nil {
		t.Fatal(err)
	}
	if !stat.IsDir() {
		t.Fatalf("%q is not a dir", storageDir)
	}
	storage := c.AddSourceFilesystem(storageDir)

	// Test the "/sheets" method.
	expectedSheets := []string{"First sheet", "Second sheet", "Third sheet"}
	gotSheets := c.Sheets(storage, "Excel", "file_with_3_sheets.xlsx", chichitester.NoCompression, []byte("{}"))
	if !reflect.DeepEqual(expectedSheets, gotSheets) {
		t.Fatalf("expected sheets %#v, got %#v", expectedSheets, gotSheets)
	}

	// Test the "/complete-path" method.
	expectedPathSuffix := "/testdata/storage/file_with_3_sheets.xlsx"
	if runtime.GOOS == "windows" {
		expectedPathSuffix = "\\testdata\\storage\\file_with_3_sheets.xlsx"
	}
	gotPath := c.CompletePath(storage, "file_with_3_sheets.xlsx")
	if !strings.HasSuffix(gotPath, expectedPathSuffix) {
		t.Fatalf("expected complete path to end with suffix %q, but it the complete path is %q", expectedPathSuffix, gotPath)
	}

	// Test the "/records" method.
	excelUIValues := chichitester.JSONEncodeUIValues(map[string]any{
		"HasColumnNames": true,
	})
	records, schema := c.Records(storage, "Excel", "storage_users.xlsx", "Sheet1", chichitester.NoCompression, excelUIValues, 100)

	expectedRecords := []map[string]any{
		{"customer_id": "1234", "email": "john.smith@example.com", "first_name": "John", "last_name": "Smith"},
		{"customer_id": "5678", "email": "paul.jordan@example.com", "first_name": "Paul", "last_name": "Jordan"},
	}
	if !reflect.DeepEqual(expectedRecords, records) {
		t.Fatalf("expected '%#v', got '%#v'", expectedRecords, records)
	}

	expectedSchema := types.Object([]types.Property{
		{Name: "customer_id", Type: types.Text()},
		{Name: "email", Type: types.Text()},
		{Name: "first_name", Type: types.Text()},
		{Name: "last_name", Type: types.Text()},
	})
	if !expectedSchema.EqualTo(schema) {
		t.Fatal("schemas do not match")
	}

}
