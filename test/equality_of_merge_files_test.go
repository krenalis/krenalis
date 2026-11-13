// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"os"
	"testing"
)

// TestEqualityOfMergeFiles ensures that the contents of the 'merge.go' files in
// the repository, kept in sync across connectors and data warehouses, are
// identical and synchronized.
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

	// Admit only the warehouse's simulated system writer wrapper and branch.
	for _, fragment := range [][]byte{
		[]byte("\treturn mergeWithSimulatedSystemWriter(ctx, conn, table, rows, deleted, nil)\n" +
			"}\n\n" +
			"// mergeWithSimulatedSystemWriter stages rows before acquiring warehouse authority.\n" +
			"func mergeWithSimulatedSystemWriter(ctx context.Context, conn *pgxpool.Conn, table warehouses.Table, " +
			"rows [][]any, deleted []any, writer *simulatedSystemWriter) error {\n"),
		[]byte("\tif writer != nil {\n" +
			"\t\treturn mergeSimulatedSystemWriter(ctx, conn, *writer, merge)\n" +
			"\t}\n"),
	} {
		if count := bytes.Count(file2, fragment); count != 1 {
			t.Fatalf("expected one occurrence of %q in warehouses/postgresql/merge.go, got %d", fragment, count)
		}
		file2 = bytes.Replace(file2, fragment, nil, 1)
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

	// Admit only the warehouse's simulated system writer wrapper and branch.
	for _, fragment := range [][]byte{
		[]byte("\treturn mergeWithSimulatedSystemWriter(ctx, conn, table, rows, deleted, nil)\n" +
			"}\n\n" +
			"// mergeWithSimulatedSystemWriter stages rows before acquiring warehouse authority.\n" +
			"func mergeWithSimulatedSystemWriter(ctx context.Context, conn *sql.Conn, table warehouses.Table, " +
			"rows [][]any, deleted []any, writer *simulatedSystemWriter) error {\n"),
		[]byte("\tif writer != nil {\n" +
			"\t\treturn mergeSimulatedSystemWriter(ctx, conn, *writer, merge)\n" +
			"\t}\n"),
	} {
		if count := bytes.Count(file2, fragment); count != 1 {
			t.Fatalf("expected one occurrence of %q in warehouses/snowflake/merge.go, got %d", fragment, count)
		}
		file2 = bytes.Replace(file2, fragment, nil, 1)
	}

	file2 = bytes.Replace(file2, []byte("\t\"github.com/krenalis/krenalis/warehouses\"\n"), nil, 1)
	file2 = bytes.ReplaceAll(file2, []byte(`warehouses.`), []byte(`connectors.`))
	if !bytes.Equal(file1, file2) {
		t.Fatal("the content of 'connectors/snowflake/merge.go' differs from " +
			"'warehouses/snowflake/merge.go', whereas the two files should be identical " +
			"and kept in sync")
	}
}
