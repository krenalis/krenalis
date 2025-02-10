//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package mysql

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/testimages"
	"github.com/meergo/meergo/types"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Test_Merge_Query tests the Merge and Query methods on supported types. It
// creates a table, inserts a row, and retrieves the data, verifying that the
// returned columns and values match the expected results.
func Test_Merge_Query(t *testing.T) {

	cols := []struct {
		DriverType  string
		DriverValue any
		MeergoType  types.Type
		MeergoValue any
	}{
		{"TINYINT", int64(-112), types.Int(8), -112},
		{"SMALLINT", int64(1427), types.Int(16), 1427},
		{"MEDIUMINT", int64(5038561), types.Int(24), 5038561},
		{"INT", int64(-105722812), types.Int(32), -105722812},
		{"BIGINT", int64(4946287520337), types.Int(64), 4946287520337},
		{"TINYINT UNSIGNED", int64(213), types.Uint(8), uint(213)},
		{"SMALLINT UNSIGNED", int64(3092), types.Uint(16), uint(3092)},
		{"MEDIUMINT UNSIGNED", int64(5038561), types.Uint(24), uint(5038561)},
		{"INT UNSIGNED", int64(3841006923), types.Uint(32), uint(3841006923)},
		{"BIGINT UNSIGNED", uint64(18192650825384015325), types.Uint(64), uint(18192650825384015325)},
		{"FLOAT", float32(1.123), types.Float(32), float64(float32(1.123))},
		{"DOUBLE", 390.491835234, types.Float(64), 390.491835234},
		{"DECIMAL(10,3)", []byte("1.123"), types.Decimal(10, 3), decimal.MustParse("1.123")},
		{"DATETIME(6)", time.Date(2023, 1, 1, 1, 2, 3, 830511000, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 830511000, time.UTC)},
		{"TIMESTAMP", time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"DATE", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"TIME", []byte("02:03:00"), types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC)},
		{"YEAR", int64(2024), types.Year(), 2024},
		{"JSON", []byte(`{"foo": "bar"}`), types.JSON(), json.Value(`{"foo":"bar"}`)},
		{"VARCHAR(100)", []byte("foo"), types.Text(), "foo"},
		{"CHAR(10)", []byte("foo"), types.Text(), "foo"},
		{"TEXT", []byte("foo"), types.Text(), "foo"},
		{"MEDIUMTEXT", []byte("foo"), types.Text(), "foo"},
		{"LONGTEXT", []byte("foo"), types.Text(), "foo"},
		{"VARBINARY(100)", []byte("foo"), types.Text(), "foo"},
		{"BINARY(10)", []byte("foo\x00\x00\x00\x00\x00\x00\x00"), types.Text(), "foo\x00\x00\x00\x00\x00\x00\x00"},
		{"BLOB", []byte("foo"), types.Text(), "foo"},
		{"MEDIUMBLOB", []byte("foo"), types.Text(), "foo"},
		{"LONGBLOB", []byte("foo"), types.Text(), "foo"},
		{"ENUM('x-small','small','medium','large','x-large')", []byte("small"), types.Text(), "small"},
		// TODO(marco): SET can be implemented as an array(T), but the driver only returns the first element of the set.
		//{"SET('one','two','three')", []byte("two,tree"), types.Array(types.Text()), []any{"two", "tree"}},
	}

	table := meergo.Table{
		Name:    "test_meergo_query",
		Columns: make([]meergo.Column, len(cols)),
		Keys:    []string{"c0"},
	}
	for i, c := range cols {
		table.Columns[i] = meergo.Column{
			Name:     fmt.Sprintf("c%d", i),
			Type:     c.MeergoType,
			Nullable: i > 0,
		}
	}

	// Run the MySQL container.
	const (
		database = "meergo"
		username = "meergo"
		password = "meergo"
	)
	var mysqlContainer testcontainers.Container
	ctx := context.Background()
	{
		// Note that a generic container is used here instead of:
		//
		//    github.com/testcontainers/testcontainers-go/modules/mysql
		//
		// because 'testcontainers-go/modules/mysql' seems to not allow to
		// increase the startup timeout above 1 minute, causing the startup to
		// fail in cases when the host is under load (for example when executing
		// many tests).
		req := testcontainers.ContainerRequest{
			Image:        testimages.MySQL,
			ExposedPorts: []string{"3306/tcp", "33060/tcp"},
			Env: map[string]string{
				"MYSQL_USER":                 username,
				"MYSQL_PASSWORD":             password,
				"MYSQL_DATABASE":             database,
				"MYSQL_ALLOW_EMPTY_PASSWORD": "true",
			},
			WaitingFor: wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(3 * time.Minute),
		}
		genericContainerReq := testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		}
		var err error
		mysqlContainer, err = testcontainers.GenericContainer(ctx, genericContainerReq)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := testcontainers.TerminateContainer(mysqlContainer); err != nil {
				t.Fatal(err)
			}
		}()
	}
	host, err := mysqlContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	if err != nil {
		t.Fatal(err)
	}

	// Open the MySQL connector.
	settings, err := json.Marshal(innerSettings{
		Host:     host,
		Port:     port.Int(),
		Username: username,
		Password: password,
		Database: database,
	})
	if err != nil {
		t.Fatal(err)
	}
	var config = meergo.DatabaseConfig{Settings: settings}
	connector, err := New(&config)
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	if err = connector.openDB(); err != nil {
		t.Fatalf("cannot open the database: %s", err)
	}

	// Create the table and add a row.
	create := bytes.NewBufferString("CREATE TABLE " + table.Name + " (\n\t")
	for i, c := range table.Columns {
		if i > 0 {
			create.WriteString(",\n\t")
		}
		create.WriteString(c.Name)
		create.WriteByte(' ')
		create.WriteString(cols[i].DriverType)
		if i == 0 {
			create.WriteString(" PRIMARY KEY")
		} else {
			create.WriteString(" NULL")
		}
	}
	create.WriteString("\n)")
	_, err = connector.db.ExecContext(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = connector.db.ExecContext(context.Background(), "DROP TABLE "+table.Name)
		if err != nil {
			t.Logf("cannot drop %s table: %s", table.Name, err)
		}
	}()
	row := make([]any, len(cols))
	for i, c := range cols {
		row[i] = c.MeergoValue
	}
	err = connector.Merge(context.Background(), table, [][]any{row})
	if err != nil {
		t.Fatalf("cannot upsert: %s", err)
	}

	// Execute the query.
	query := "SELECT "
	for i, c := range table.Columns {
		if i > 0 {
			query += ", "
		}
		query += c.Name
	}
	query += " FROM " + table.Name
	rows, columns, err := connector.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query execution is failed: %s", err)
	}
	defer rows.Close()
	if len(table.Columns) != len(columns) {
		t.Fatalf("expected %d columns, got %d", len(table.Columns), len(columns))
	}
	for i, c := range table.Columns {
		if !reflect.DeepEqual(c, columns[i]) {
			t.Fatalf("unexpected column:\nexpected: %#v\ngot:      %#v", c, columns[i])
		}
	}
	scanner := scanner{
		values: make([]any, len(columns)),
	}
	dest := make([]any, len(columns))
	for i := range columns {
		dest[i] = &scanner
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			_ = rows.Close()
			t.Fatalf("cannot scan row: %s", err)
		}
		for i, v := range scanner.values {
			if expected := cols[i].DriverValue; !reflect.DeepEqual(expected, v) {
				t.Fatalf("column %q: expected %#v (%T), got %#v (%T)", table.Columns[i].Name, expected, expected, v, v)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("cannot scan row: %s", err)
	}

}

// scanner implements the sql.Scanner interface to read the database values.
type scanner struct {
	index  int
	values []any
}

func (sv *scanner) Scan(src any) error {
	sv.values[sv.index] = src
	sv.index++
	sv.index %= len(sv.values)
	return nil
}

func (sv *scanner) reset() {
	sv.index = 0
}
