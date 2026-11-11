// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// jsonReadConnector simulates server text and records executed SQL locally.
type jsonReadConnector struct {
	kind    string
	values  []driver.Value
	queries []string
}

// Connect returns the query-capable test connection.
func (c *jsonReadConnector) Connect(context.Context) (driver.Conn, error) {
	return &jsonReadConn{connector: c}, nil
}

// Driver returns the test driver.
func (c *jsonReadConnector) Driver() driver.Driver {
	return c
}

// Open is unused because the test uses sql.OpenDB.
func (c *jsonReadConnector) Open(string) (driver.Conn, error) {
	return nil, errors.New("connectors/snowflake: jsonReadConnector.Open is unused")
}

type jsonReadConn struct {
	connector *jsonReadConnector
}

// Begin is unused because reading the result needs no transaction.
func (c *jsonReadConn) Begin() (driver.Tx, error) {
	return nil, errors.New("connectors/snowflake: jsonReadConn.Begin is unused")
}

// Close releases the test connection.
func (c *jsonReadConn) Close() error {
	return nil
}

// Prepare is unused because the test exercises QueryContext directly.
func (c *jsonReadConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("connectors/snowflake: jsonReadConn.Prepare is unused")
}

// QueryContext records the query and returns the simulated server result.
func (c *jsonReadConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.connector.queries = append(c.connector.queries, query)
	if len(c.connector.queries) != 1 {
		return nil, errors.New("connectors/snowflake: unexpected additional query")
	}
	return &jsonReadRows{kind: c.connector.kind, values: c.connector.values}, nil
}

type jsonReadRows struct {
	kind   string
	values []driver.Value
	next   int
}

// Close releases the result.
func (r *jsonReadRows) Close() error {
	return nil
}

// ColumnTypeDatabaseTypeName returns the semistructured server type.
func (r *jsonReadRows) ColumnTypeDatabaseTypeName(int) string {
	return r.kind
}

// ColumnTypeNullable allows SQL NULL independently of JSON null.
func (r *jsonReadRows) ColumnTypeNullable(int) (bool, bool) {
	return true, true
}

// Columns returns the original result column.
func (r *jsonReadRows) Columns() []string {
	return []string{"value"}
}

// Next returns each row in the configured order without changing its text.
func (r *jsonReadRows) Next(dest []driver.Value) error {
	if r.next == len(r.values) {
		return io.EOF
	}
	dest[0] = r.values[r.next]
	r.next++
	return nil
}

// TestQueryJSONTransport checks the real connector entry point with a local
// driver: one unchanged query, stable rows and numeric text preserved verbatim.
func TestQueryJSONTransport(t *testing.T) {
	const query = "  WITH source AS (SELECT * FROM INPUT)\n" +
		"SELECT payload AS \"value\" FROM source ORDER BY updated_at; -- preserve SQL\n"
	for _, test := range []struct {
		kind   string
		typ    types.Type
		values []driver.Value
	}{
		{"VARIANT", types.JSON(), []driver.Value{nil, "null", "12345678901234567890123456789012345678", "0.0000000000000000000000000000000000001", "1.000000000000000e+02", "1.000000000000000e+02", "1.0000000000000001e+02", "1.797693134862316e+308", ` {"value":[1.000000000000000e+02,12345678901234567890123456789012345678]} `}},
		{"ARRAY", types.Array(types.JSON()), []driver.Value{nil, `[]`, `[1.000000000000000e+02,1.000000000000000e+02,12345678901234567890123456789012345678]`}},
		{"OBJECT", types.Map(types.JSON()), []driver.Value{nil, `{}`, `{"a.b[0]":1.000000000000000e+02,"fixed":12345678901234567890123456789012345678}`}},
	} {
		t.Run(test.kind, func(t *testing.T) {
			response := &jsonReadConnector{kind: test.kind, values: test.values}
			db := sql.OpenDB(response)
			t.Cleanup(func() {
				err := db.Close()
				if err != nil {
					t.Fatalf("expected database close, got %v", err)
				}
			})
			sf := &Snowflake{db: db}
			rows, columns, err := sf.Query(t.Context(), query)
			if err != nil {
				t.Fatalf("expected JSON query result, got %v", err)
			}
			defer rows.Close()
			if len(columns) != 1 || columns[0].Name != "value" || !reflect.DeepEqual(columns[0].Type, test.typ) {
				t.Fatalf("expected original JSON column, got %#v", columns)
			}
			count := 0
			for rows.Next() {
				var got any
				if err := rows.Scan(&got); err != nil {
					t.Fatalf("expected JSON row, got %v", err)
				}
				if count >= len(test.values) || !reflect.DeepEqual(got, test.values[count]) {
					t.Fatalf("expected unchanged row %d, got %#v", count, got)
				}
				count++
			}
			err = rows.Err()
			if err != nil {
				t.Fatalf("expected complete JSON result, got %v", err)
			}
			err = rows.Close()
			if err != nil {
				t.Fatalf("expected result close, got %v", err)
			}
			if count != len(test.values) || len(response.queries) != 1 || response.queries[0] != query {
				t.Fatalf("expected %d rows and one unchanged query, got %d rows and %q", len(test.values), count, response.queries)
			}
		})
	}
}
