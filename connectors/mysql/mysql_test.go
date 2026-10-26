// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mysql

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestStringValuesWithSQLModes verifies that string values are preserved
// across different SQL modes when querying and merging data.
func TestStringValuesWithSQLModes(t *testing.T) {

	const database, username, password = "krenalis", "krenalis", "krenalis"
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testimages.MySQL,
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"MYSQL_DATABASE":             database,
				"MYSQL_USER":                 username,
				"MYSQL_PASSWORD":             password,
				"MYSQL_ALLOW_EMPTY_PASSWORD": "true",
				"MYSQL_ROOT_HOST":            "%",
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(3*time.Minute),
				wait.ForListeningPort("3306/tcp"),
			),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("expected MySQL container, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container cleanup, got %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected MySQL host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "3306/tcp")
	if err != nil {
		t.Fatalf("expected MySQL port, got %v", err)
	}
	settings := innerSettings{Host: host, Port: int(port.Num()), Username: "root", Database: database}
	admin, err := testconnector.NewDatabase[*MySQL]("mysql", settings)
	if err != nil {
		t.Fatalf("expected administrative connector, got %v", err)
	}
	t.Cleanup(func() {
		err := admin.Close()
		if err != nil {
			t.Errorf("expected administrative connector cleanup, got %v", err)
		}
	})
	err = admin.openDB(t.Context())
	if err != nil {
		t.Fatalf("expected administrative database, got %v", err)
	}
	_, err = admin.db.ExecContext(t.Context(), "CREATE TABLE string_values (id INT PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("expected fixture table, got %v", err)
	}
	settings.Username = username
	settings.Password = password

	for _, mode := range []string{
		"STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION",
		"STRICT_TRANS_TABLES,NO_BACKSLASH_ESCAPES,NO_ENGINE_SUBSTITUTION",
		"STRICT_TRANS_TABLES,ANSI_QUOTES,NO_ENGINE_SUBSTITUTION",
		"STRICT_TRANS_TABLES,ANSI,NO_BACKSLASH_ESCAPES,NO_ENGINE_SUBSTITUTION",
	} {
		t.Run(mode, func(t *testing.T) {

			_, err := admin.db.ExecContext(t.Context(), "SET GLOBAL sql_mode = ?", mode)
			if err != nil {
				t.Fatalf("expected updated server SQL mode, got %v", err)
			}
			var globalMode string
			err = admin.db.QueryRowContext(t.Context(), "SELECT @@GLOBAL.sql_mode").Scan(&globalMode)
			if err != nil {
				t.Fatalf("expected server SQL mode, got %v", err)
			}
			globalModes := strings.Split(globalMode, ",")
			for requested := range strings.SplitSeq(mode, ",") {
				if !slices.Contains(globalModes, requested) {
					t.Fatalf("expected server SQL mode %q, got %q", requested, globalMode)
				}
			}
			var expectedModes []string
			for _, mode := range globalModes {
				if mode != "NO_BACKSLASH_ESCAPES" {
					expectedModes = append(expectedModes, mode)
				}
			}
			slices.Sort(expectedModes)
			connector, err := testconnector.NewDatabase[*MySQL]("mysql", settings)
			if err != nil {
				t.Fatalf("expected MySQL connector, got %v", err)
			}
			t.Cleanup(func() {
				err := connector.Close()
				if err != nil {
					t.Errorf("expected connector cleanup, got %v", err)
				}
			})

			t.Run("NewConnections", func(t *testing.T) {
				var firstID uint64
				for range 2 {
					rows, _, err := connector.Query(t.Context(),
						"SELECT CONNECTION_ID() AS connection_id, @@SESSION.sql_mode AS session_mode")
					if err != nil {
						t.Fatalf("expected session query, got %v", err)
					}
					// Keep both connections acquired until the subtest finishes.
					defer rows.Close()
					if !rows.Next() {
						t.Fatalf("expected session result, got %v", rows.Err())
					}
					var id uint64
					var sessionMode string
					if err := rows.Scan(&id, &sessionMode); err != nil {
						t.Fatalf("expected session details, got %v", err)
					}
					if id == firstID {
						t.Fatalf("expected distinct connections, got repeated ID %d", id)
					}
					firstID = id
					actualModes := strings.Split(sessionMode, ",")
					slices.Sort(actualModes)
					if !slices.Equal(actualModes, expectedModes) {
						t.Fatalf("expected session SQL modes %v, got %v", expectedModes, actualModes)
					}
				}
			})

			for _, test := range []struct {
				name   string
				value  any
				typ    types.Type
				stored any
			}{
				{"null", nil, types.String(), nil},
				{"empty", "", types.String(), ""},
				{"single quote", `x' OR TRUE # `, types.String(), `x' OR TRUE # `},
				{"double quote", `x" OR TRUE # `, types.String(), `x" OR TRUE # `},
				{"case and spaces", "AbC ", types.String(), "AbC "},
				{"escapes", "\x00'\"\b\n\r\t\032\\雪🍎", types.String(), "\x00'\"\b\n\r\t\032\\雪🍎"},
				{"json", json.Value(`{"value":"x\" OR TRUE #","quote":"'","slash":"\\"}`), types.JSON(), `{"value":"x\" OR TRUE #","quote":"'","slash":"\\"}`},
				{"json string", `{"value":"x' OR TRUE #","slash":"\\"}`, types.JSON(), `{"value":"x' OR TRUE #","slash":"\\"}`},
				{"json timestamp", "2026'10'23", types.JSON(), "2026'10'23"},
				{"datetime", time.Date(2026, 10, 23, 16, 35, 12, 123456000, time.UTC), types.DateTime(), "2026-10-23 16:35:12.123456"},
				{"date", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), types.Date(), "2026-10-23"},
				{"time", time.Date(1970, 1, 1, 16, 35, 12, 123456000, time.UTC), types.Time(), "16:35:12.123456"},
			} {
				t.Run("SQLLiteral/"+test.name, func(t *testing.T) {
					_, err := admin.db.ExecContext(t.Context(), "DELETE FROM string_values")
					if err != nil {
						t.Fatalf("expected cleared fixtures, got %v", err)
					}
					_, err = admin.db.ExecContext(t.Context(),
						"INSERT INTO string_values VALUES (1, ?), (2, ?)", "safe", test.stored)
					if err != nil {
						t.Fatalf("expected inserted fixtures, got %v", err)
					}
					query := "SELECT @@SESSION.sql_mode AS session_mode, COUNT(*) AS matches_count, " +
						"COALESCE(MIN(id), 0) AS matched_id FROM string_values WHERE CAST(value AS BINARY) <=> " +
						connector.SQLLiteral(test.value, test.typ)
					rows, _, err := connector.Query(t.Context(), query)
					if err != nil {
						t.Fatalf("expected literal query, got %v", err)
					}
					defer rows.Close()
					if !rows.Next() {
						t.Fatalf("expected query result, got %v", rows.Err())
					}
					var sessionMode string
					var count, id int
					if err := rows.Scan(&sessionMode, &count, &id); err != nil {
						t.Fatalf("expected literal query result, got %v", err)
					}
					if count != 1 || id != 2 {
						t.Fatalf("expected one matching row with ID 2, got %d rows with minimum ID %d", count, id)
					}
					actualModes := strings.Split(sessionMode, ",")
					slices.Sort(actualModes)
					if !slices.Equal(actualModes, expectedModes) {
						t.Fatalf("expected session SQL modes %v, got %v", expectedModes, actualModes)
					}
				})
			}

			table := connectors.Table{
				Name: "string_values",
				Columns: []connectors.Column{
					{Name: "id", Type: types.Int(32)},
					{Name: "value", Type: types.String()},
				},
				Keys: []string{"id"},
			}
			for _, test := range []struct {
				name    string
				payload string
			}{
				{"single quote", `x'),(2,0x696e6a6563746564) # `},
				{"double quote", `x"),(2,0x696e6a6563746564) # `},
				{"backslash", `path\to\file`},
				{"backslash quote", `x\' OR TRUE # `},
				{"null byte", "a\x00b"},
				{"unicode", "雪🍎"},
			} {
				t.Run("Merge/"+test.name, func(t *testing.T) {

					_, err := admin.db.ExecContext(t.Context(), "DELETE FROM string_values")
					if err != nil {
						t.Fatalf("expected cleared fixtures, got %v", err)
					}

					// Insert the row, then update the same primary key with a different value.
					for _, payload := range []string{test.payload, "updated " + test.payload} {
						err = connector.Merge(t.Context(), table, [][]any{{1, payload}})
						if err != nil {
							t.Fatalf("expected merged string value, got %v", err)
						}
						var count, id int
						var value string
						err = admin.db.QueryRowContext(t.Context(),
							"SELECT COUNT(*), COALESCE(MIN(id), 0), COALESCE(MIN(value), '') FROM string_values").
							Scan(&count, &id, &value)
						if err != nil {
							t.Fatalf("expected stored string value, got %v", err)
						}
						if count != 1 || id != 1 || value != payload {
							t.Fatalf("expected one row with ID 1 and value %q, got %d rows, ID %d and value %q",
								payload, count, id, value)
						}
					}

				})
			}

			// Connection checks must also work with the server's SQL mode.
			err = testConnection(t.Context(), &settings, connector.env.Dial)
			if err != nil {
				t.Fatalf("expected connection check, got %v", err)
			}

		})
	}

}

// Test_Merge_Query tests the Merge and Query methods on supported types. It
// creates a table, inserts a row, and retrieves the data, verifying that the
// returned columns and values match the expected results.
func Test_Merge_Query(t *testing.T) {

	cols := []struct {
		DriverType    string
		DriverValue   any
		KrenalisType  types.Type
		KrenalisValue any
	}{
		{"TINYINT", int64(-112), types.Int(8), -112},
		{"SMALLINT", int64(1427), types.Int(16), 1427},
		{"MEDIUMINT", int64(5038561), types.Int(24), 5038561},
		{"INT", int64(-105722812), types.Int(32), -105722812},
		{"BIGINT", int64(4946287520337), types.Int(64), 4946287520337},
		{"TINYINT UNSIGNED", int64(213), types.Int(8).Unsigned(), uint(213)},
		{"SMALLINT UNSIGNED", int64(3092), types.Int(16).Unsigned(), uint(3092)},
		{"MEDIUMINT UNSIGNED", int64(5038561), types.Int(24).Unsigned(), uint(5038561)},
		{"INT UNSIGNED", int64(3841006923), types.Int(32).Unsigned(), uint(3841006923)},
		{"BIGINT UNSIGNED", uint64(18192650825384015325), types.Int(64).Unsigned(), uint(18192650825384015325)},
		{"FLOAT", float32(1.123), types.Float(32), float64(float32(1.123))},
		{"DOUBLE", 390.491835234, types.Float(64), 390.491835234},
		{"DECIMAL(10,3)", []byte("1.123"), types.Decimal(10, 3), decimal.MustParse("1.123")},
		{"DATETIME(6)", time.Date(2023, 1, 1, 1, 2, 3, 830511000, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 830511000, time.UTC)},
		{"TIMESTAMP", time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"DATE", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"TIME", []byte("02:03:00"), types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC)},
		{"YEAR", int64(2024), types.Year(), 2024},
		{"JSON", []byte(`{"foo": "bar"}`), types.JSON(), json.Value(`{"foo":"bar"}`)},
		{"VARCHAR(100)", []byte("foo"), types.String(), "foo"},
		{"CHAR(10)", []byte("foo"), types.String(), "foo"},
		{"TEXT", []byte("foo"), types.String(), "foo"},
		{"MEDIUMTEXT", []byte("foo"), types.String(), "foo"},
		{"LONGTEXT", []byte("foo"), types.String(), "foo"},
		{"VARBINARY(100)", []byte("foo"), types.String(), "foo"},
		{"BINARY(10)", []byte("foo\x00\x00\x00\x00\x00\x00\x00"), types.String(), "foo\x00\x00\x00\x00\x00\x00\x00"},
		{"BLOB", []byte("foo"), types.String(), "foo"},
		{"MEDIUMBLOB", []byte("foo"), types.String(), "foo"},
		{"LONGBLOB", []byte("foo"), types.String(), "foo"},
		{"ENUM('x-small','small','medium','large','x-large')", []byte("small"), types.String(), "small"},
		// TODO(marco): SET can be implemented as an array(T), but the driver only returns the first element of the set.
		//{"SET('one','two','three')", []byte("two,tree"), types.Array(types.String()), []any{"two", "tree"}},
	}

	table := connectors.Table{
		Name:    "test_krenalis_query",
		Columns: make([]connectors.Column, len(cols)),
		Keys:    []string{"c0"},
	}
	for i, c := range cols {
		table.Columns[i] = connectors.Column{
			Name:     fmt.Sprintf("c%d", i),
			Type:     c.KrenalisType,
			Nullable: i > 0,
		}
	}

	// Run the MySQL container.
	const (
		database = "krenalis"
		username = "krenalis"
		password = "krenalis"
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
			WaitingFor: wait.ForAll(
				wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(3*time.Minute),
				wait.ForListeningPort("3306/tcp"),
			),
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
	connector, err := testconnector.NewDatabase[*MySQL]("mysql", innerSettings{
		Host:     host,
		Port:     int(port.Num()),
		Username: username,
		Password: password,
		Database: database,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	if err = connector.openDB(ctx); err != nil {
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
		row[i] = c.KrenalisValue
	}
	err = connector.Merge(context.Background(), table, [][]any{row})
	if err != nil {
		t.Fatalf("cannot upsert: %s", err)
	}

	// Execute the query.
	var query strings.Builder
	query.WriteString("SELECT ")
	for i, c := range table.Columns {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(c.Name)
	}
	query.WriteString(" FROM " + table.Name)
	rows, columns, err := connector.Query(context.Background(), query.String())
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
