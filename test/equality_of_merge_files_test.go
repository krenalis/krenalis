// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"os"
	"testing"
)

// TestEqualityOfMergeFiles checks that each connector's merge.go stays in sync
// with its warehouse counterpart for PostgreSQL and Snowflake. For Snowflake,
// it also checks that the two merge_test.go files are identical.
func TestEqualityOfMergeFiles(t *testing.T) {

	// Check #1: the 'merge.go' file in the PostgreSQL connector must be
	// identical to the 'merge.go' file in the PostgreSQL warehouse.
	file1, err := os.ReadFile("../connectors/postgresql/merge.go")
	if err != nil {
		t.Fatal(err)
	}
	file1 = bytes.Replace(file1, []byte("\t\"github.com/krenalis/krenalis/connectors\"\n"), nil, 1)
	file2, err := os.ReadFile("../warehouses/postgresql/merge.go")
	if err != nil {
		t.Fatal(err)
	}
	file2 = bytes.Replace(file2, []byte("\t\"github.com/krenalis/krenalis/warehouses\"\n"), nil, 1)
	file2 = bytes.ReplaceAll(file2, []byte(`warehouses.`), []byte(`connectors.`))
	if !bytes.Equal(file1, file2) {
		t.Fatal("the content of 'connectors/postgresql/merge.go' differs from " +
			"'warehouses/postgresql/merge.go', whereas the two files should be identical " +
			"and kept in sync")
	}

	// Check #2: the 'merge.go' file in the Snowflake connector must be
	// identical to the 'merge.go' file in the Snowflake warehouse.
	file1, err = os.ReadFile("../connectors/snowflake/merge.go")
	if err != nil {
		t.Fatal(err)
	}
	file1 = bytes.Replace(file1, []byte("\t\"github.com/krenalis/krenalis/connectors\"\n"), nil, 1)
	file2, err = os.ReadFile("../warehouses/snowflake/merge.go")
	if err != nil {
		t.Fatal(err)
	}
	file2 = bytes.Replace(file2, []byte("\t\"github.com/krenalis/krenalis/warehouses\"\n"), nil, 1)
	file2 = bytes.ReplaceAll(file2, []byte(`warehouses.`), []byte(`connectors.`))
	file2 = bytes.ReplaceAll(file2, []byte("warehouses/snowflake:"), []byte("connectors/snowflake:"))
	if !bytes.Equal(file1, file2) {
		t.Fatal("the content of 'connectors/snowflake/merge.go' differs from " +
			"'warehouses/snowflake/merge.go', whereas the two files should be identical " +
			"and kept in sync")
	}

	file1, err = os.ReadFile("../connectors/snowflake/merge_test.go")
	if err != nil {
		t.Fatalf("expected connector file, got %v", err)
	}
	file2, err = os.ReadFile("../warehouses/snowflake/merge_test.go")
	if err != nil {
		t.Fatalf("expected warehouse file, got %v", err)
	}
	if !bytes.Equal(file1, file2) {
		t.Fatal("expected identical Snowflake merge_test.go files, got different contents")
	}

}
