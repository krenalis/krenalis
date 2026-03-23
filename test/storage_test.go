// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/meergotester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestStorage(t *testing.T) {

	// Determine the storage directory.
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

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	// Create a file storage connection.
	storage := c.CreateSourceFileSystem()

	// Test the "/files/sheets" endpoint.
	expectedSheets := []string{"First sheet", "Second sheet", "Third sheet"}
	gotSheets := c.Sheets(storage, "file_with_3_sheets.xlsx", "excel", meergotester.NoCompression, json.Value("{}"))
	if !reflect.DeepEqual(expectedSheets, gotSheets) {
		t.Fatalf("expected sheets %#v, got %#v", expectedSheets, gotSheets)
	}

	// Test the "/files/absolute" endpoint.
	expectedPathSuffix := "/testdata/storage/file_with_3_sheets.xlsx"
	if runtime.GOOS == "windows" {
		expectedPathSuffix = "\\testdata\\storage\\file_with_3_sheets.xlsx"
	}
	gotPath := c.AbsolutePath(storage, "file_with_3_sheets.xlsx")
	if !strings.HasSuffix(gotPath, expectedPathSuffix) {
		t.Fatalf("expected absolute path to end with suffix %q, but it the absolute path is %q", expectedPathSuffix, gotPath)
	}

	// Test the "/files" endpoint.
	excelSettings := meergotester.JSONEncodeSettings(map[string]any{
		"hasColumnNames": true,
	})
	records, schema := c.File(storage, "storage_users.xlsx", "excel", "Sheet1", meergotester.NoCompression, excelSettings, 100)

	expectedRecords := []map[string]any{
		{"customer_id": "1234", "email": "john.smith@example.com", "first_name": "John", "last_name": "Smith"},
		{"customer_id": "5678", "email": "paul.jordan@example.com", "first_name": "Paul", "last_name": "Jordan"},
	}
	if !reflect.DeepEqual(expectedRecords, records) {
		t.Fatalf("expected '%#v', got '%#v'", expectedRecords, records)
	}

	expectedSchema := types.Object([]types.Property{
		{Name: "customer_id", Type: types.String()},
		{Name: "email", Type: types.String()},
		{Name: "first_name", Type: types.String()},
		{Name: "last_name", Type: types.String()},
	})
	if !types.Equal(expectedSchema, schema) {
		t.Fatal("schemas do not match")
	}

}
