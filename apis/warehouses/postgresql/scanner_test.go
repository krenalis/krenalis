//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"chichi/apis/postgres"
)

const settingsEnvKey = "CHICHI_TEST_PATH_POSTGRESQL"

func TestScanToAny(t *testing.T) {

	columns := []struct {
		Type            string
		Value           string
		ExpectedSQLType string
		ExpectedGoType  string
	}{
		{"SERIAL", "", "", "int32"},
		{"smallint", "1", "", "int16"},
		{"integer", "1", "", "int32"},
		{"bigint", "1", "", "int64"},
		{"numeric(10,3)", "1.123", "", "pgtype.Numeric"},
		{"real", "1.123", "", "float32"},
		{"double precision", "1.123", "", "float64"},
		{"character varying", "'foo'", "", "string"},
		{"character(3)", "'foo'", "", "string"},
		{"text", "'FOO'", "", "string"},
		{"timestamp without time zone", "'2023-01-01 01:02:03'", "", "time.Time"},
		{"timestamp with time zone", "'2023-01-01 01:02:03 PST'", "", "time.Time"},
		{"date", "'2023-01-01'", "", "time.Time"},
		{"time", "'01:02:03'", "", "pgtype.Time"},
		{"boolean", "true", "", "bool"},
		{"inet", "'127.0.0.1'", "", "netip.Prefix"},
		{"uuid", "'4d92d698-687d-4447-b34f-6b29d74a9730'", "", "[16]uint8"},
		{"integer[]", "'{1,2,3}'", "", "[]interface {}"},
	}

	settingsFile, ok := os.LookupEnv(settingsEnvKey)
	if !ok {
		t.Skipf("the %s environment variable is not present", settingsEnvKey)
	}

	// Read the configuration file.
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
	}
	var settings psSettings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		t.Fatalf("cannot unmarshal warehouse settings specified in the %s environment variable: %s", settingsEnvKey, err)
	}

	// Open the data warehouse.
	db, err := postgres.Open(settings.options())
	if err != nil {
		t.Fatalf("cannot open the warehouse from settings in the %s environment variable: %s", settingsEnvKey, err)
	}
	defer db.Close()

	// Create the table.
	query := "CREATE TABLE test_chichi_scanner (\n\t"
	for i, c := range columns {
		if i > 0 {
			query += ",\n\t"
		}
		query += fmt.Sprintf("c%d %s", i, c.Type)
	}
	query += "\n)"
	_, err = db.Exec(context.Background(), query)
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err := db.Exec(context.Background(), "DROP TABLE test_chichi_scanner")
		if err != nil {
			t.Logf("cannot drop test_chichi_scanner table: %s", err)
		}
	}()
	query = `INSERT INTO test_chichi_scanner (`
	vals := ""
	for i, c := range columns {
		if i == 0 {
			continue
		}
		if i > 1 {
			query += ","
			vals += ","
		}
		query += fmt.Sprintf("c%d", i)
		vals += c.Value
	}
	query += ") VALUES (" + vals + ")"
	_, err = db.Exec(context.Background(), query)
	if err != nil {
		t.Fatalf("cannot insert values into table: %s", err)
	}

	// Read the row.
	query = "SELECT "
	for i := range columns {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("c%d", i)
	}
	query += " FROM test_chichi_scanner"
	values := make([]any, len(columns))
	for i := range values {
		var v any
		values[i] = &v
	}
	rows, err := db.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("cannot query rows: %s", err)
	}

	// Check the returned rows.
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			rows.Close()
			t.Fatalf("cannot scan row: %s", err)
		}
		for i, v := range values {
			c := columns[i]
			got := fmt.Sprintf("%T", *(v.(*any)))
			if got != c.ExpectedGoType {
				t.Fatalf("column %q: expected Go type %s, got %s", c.Type, c.ExpectedGoType, got)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("cannot scan row: %s", err)
	}

}
