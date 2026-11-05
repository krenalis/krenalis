// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mysql

import (
	"bytes"
	"context"
	"database/sql"
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

// TestStringValuesAcrossSQLModes verifies that string values are preserved
// in queries and merge operations across different SQL modes.
func TestStringValuesAcrossSQLModes(t *testing.T) {

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
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container cleanup, got %v", err)
		}
	})
	if err != nil {
		t.Fatalf("expected MySQL container, got %v", err)
	}
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
	err = admin.openDB(t.Context(), false)
	if err != nil {
		t.Fatalf("expected administrative database, got %v", err)
	}
	_, err = admin.db.ExecContext(t.Context(), "CREATE TABLE string_values (id INT PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("expected fixture table, got %v", err)
	}
	_, err = admin.db.ExecContext(t.Context(),
		"CREATE TABLE typed_values (id INT PRIMARY KEY, "+
			"text_value VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci, document JSON, "+
			"datetime_value DATETIME(6), date_value DATE, time_value TIME(6))")
	if err != nil {
		t.Fatalf("expected typed fixture table, got %v", err)
	}
	settings.Username = username
	settings.Password = password
	newConnector := func(t *testing.T) *MySQL {
		t.Helper()
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
		return connector
	}

	const document = `{"value":"x\" OR TRUE #","quote":"'","slash":"\\","text":"雪🍎","escapes":"\b\n\r\t\u0000\u001a"}`
	values := []struct {
		name   string
		value  any
		typ    types.Type
		stored any
	}{
		{"null", nil, types.String(), nil},
		{"empty", "", types.String(), ""},
		{"single quote", `x' OR TRUE # `, types.String(), `x' OR TRUE # `},
		{"double quote", `x" OR TRUE # `, types.String(), `x" OR TRUE # `},
		{"extra row single quote", `x'),(2,0x696e6a6563746564) # `, types.String(), `x'),(2,0x696e6a6563746564) # `},
		{"extra row double quote", `x"),(2,0x696e6a6563746564) # `, types.String(), `x"),(2,0x696e6a6563746564) # `},
		{"case and spaces", "AbC ", types.String(), "AbC "},
		{"backslash", `path\to\file`, types.String(), `path\to\file`},
		{"backslash quote", `x\' OR TRUE # `, types.String(), `x\' OR TRUE # `},
		{"single quote boundaries", "'x'", types.String(), "'x'"},
		{"double quote boundaries", `"x"`, types.String(), `"x"`},
		{"backslash boundaries", `\x\`, types.String(), `\x\`},
		{"null byte", "\x00x\x00", types.String(), "\x00x\x00"},
		{"backspace", "\bx\b", types.String(), "\bx\b"},
		{"newline", "\nx\n", types.String(), "\nx\n"},
		{"carriage return", "\rx\r", types.String(), "\rx\r"},
		{"tab", "\tx\t", types.String(), "\tx\t"},
		{"control Z", "\x1ax\x1a", types.String(), "\x1ax\x1a"},
		{"escapes", "\x00'\"\b\n\r\t\032\\雪🍎", types.String(), "\x00'\"\b\n\r\t\032\\雪🍎"},
		{"unicode", "雪🍎", types.String(), "雪🍎"},
		{"json", json.Value(document), types.JSON(), document},
		{"json string", `{"value":"x' OR TRUE #","slash":"\\"}`, types.JSON(), `{"value":"x' OR TRUE #","slash":"\\"}`},
		{"json timestamp", "2026'10'23", types.JSON(), "2026'10'23"},
		{"datetime", time.Date(2026, 10, 23, 16, 35, 12, 123456000, time.UTC), types.DateTime(), "2026-10-23 16:35:12.123456"},
		{"datetime seconds", time.Date(2026, 10, 23, 16, 35, 12, 0, time.UTC), types.DateTime(), "2026-10-23 16:35:12"},
		{"datetime nanoseconds", time.Date(2026, 10, 23, 16, 35, 12, 123456789, time.UTC), types.DateTime(), "2026-10-23 16:35:12.123456"},
		{"date", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), types.Date(), "2026-10-23"},
		{"time", time.Date(1970, 1, 1, 16, 35, 12, 123456000, time.UTC), types.Time(), "16:35:12.123456"},
		{"time seconds", time.Date(1970, 1, 1, 16, 35, 12, 0, time.UTC), types.Time(), "16:35:12"},
		{"time nanoseconds", time.Date(1970, 1, 1, 16, 35, 12, 123456789, time.UTC), types.Time(), "16:35:12.123456"},
	}
	table := connectors.Table{
		Name: "string_values",
		Columns: []connectors.Column{
			{Name: "id", Type: types.Int(32)},
			{Name: "value", Type: types.String(), Nullable: true},
		},
		Keys: []string{"id"},
	}
	operations := []struct {
		name    string
		isQuery bool
		run     func(context.Context, *MySQL) error
	}{
		{"Query", true, func(ctx context.Context, connector *MySQL) error {
			rows, _, err := connector.Query(ctx, "SELECT 1 AS value")
			if err != nil {
				return err
			}
			return rows.Close()
		}},
		{"Columns", false, func(ctx context.Context, connector *MySQL) error {
			_, err := connector.Columns(ctx, table.Name)
			return err
		}},
		{"Merge", false, func(ctx context.Context, connector *MySQL) error {
			return connector.Merge(ctx, table, [][]any{{1, `x\' OR TRUE # `}})
		}},
	}

	for _, mode := range []string{
		"",
		"NO_BACKSLASH_ESCAPES",
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
			globalModes := strings.FieldsFunc(globalMode, func(r rune) bool { return r == ',' })
			for requested := range strings.SplitSeq(mode, ",") {
				if requested != "" && !slices.Contains(globalModes, requested) {
					t.Fatalf("expected server SQL mode %q, got %q", requested, globalMode)
				}
			}
			expectedModes := slices.Clone(globalModes)
			slices.Sort(expectedModes)
			writeModes := slices.DeleteFunc(slices.Clone(expectedModes), func(mode string) bool {
				return mode == "NO_BACKSLASH_ESCAPES"
			})
			connector := newConnector(t)

			for _, operation := range operations {
				t.Run("NewConnections/"+operation.name, func(t *testing.T) {

					ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
					defer cancel()
					connector := newConnector(t)
					err := operation.run(ctx, connector)
					if err != nil {
						t.Fatalf("expected initial %s, got %v", operation.name, err)
					}
					wantModes := expectedModes
					if !operation.isQuery {
						wantModes = writeModes
					}

					var firstID uint64
					for range 2 {
						query := "SELECT CONNECTION_ID() AS connection_id, " +
							"@@SESSION.sql_mode AS session_mode, @@GLOBAL.sql_mode AS global_mode"
						var rows connectors.Rows
						if operation.isQuery {
							rows, _, err = connector.Query(ctx, query)
						} else {
							// Inspect write sessions without mixing connector roles.
							rows, err = connector.db.QueryContext(ctx, query)
						}
						if err != nil {
							t.Fatalf("expected session query, got %v", err)
						}
						// Keep both connections acquired until the subtest finishes.
						defer rows.Close()
						if !rows.Next() {
							t.Fatalf("expected session result, got %v", rows.Err())
						}
						var id uint64
						var sessionMode, actualGlobalMode string
						if err := rows.Scan(&id, &sessionMode, &actualGlobalMode); err != nil {
							t.Fatalf("expected session details, got %v", err)
						}
						if id == firstID {
							t.Fatalf("expected distinct connections, got repeated ID %d", id)
						}
						firstID = id
						actualModes := strings.FieldsFunc(sessionMode, func(r rune) bool { return r == ',' })
						slices.Sort(actualModes)
						if !slices.Equal(actualModes, wantModes) {
							t.Fatalf("expected session SQL modes %v, got %v", wantModes, actualModes)
						}
						if actualGlobalMode != globalMode {
							t.Fatalf("expected global SQL mode %q, got %q", globalMode, actualGlobalMode)
						}
					}

				})
			}

			for _, test := range values {
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

					literal := connector.SQLLiteral(test.value, test.typ)
					query := "SELECT @@SESSION.sql_mode AS session_mode, " +
						"CHARSET(" + literal + ") AS literal_charset, " +
						"COUNT(*) AS matches_count, " +
						"COALESCE(MIN(id), 0) AS matched_id FROM string_values WHERE CAST(value AS BINARY) <=> CAST(" +
						literal + " AS BINARY)"
					rows, _, err := connector.Query(t.Context(), query)
					if err != nil {
						t.Fatalf("expected literal query, got %v", err)
					}
					defer rows.Close()
					if !rows.Next() {
						t.Fatalf("expected query result, got %v", rows.Err())
					}
					var sessionMode, charset string
					var count, id int
					if err := rows.Scan(&sessionMode, &charset, &count, &id); err != nil {
						t.Fatalf("expected literal query result, got %v", err)
					}
					if count != 1 || id != 2 {
						t.Fatalf("expected one matching row with ID 2, got %d rows with minimum ID %d", count, id)
					}
					isText := test.typ.Kind() == types.StringKind || test.typ.Kind() == types.JSONKind
					if test.value != nil && isText && charset != "utf8mb4" {
						t.Fatalf("expected literal charset utf8mb4, got %q", charset)
					}
					actualModes := strings.FieldsFunc(sessionMode, func(r rune) bool { return r == ',' })
					slices.Sort(actualModes)
					if !slices.Equal(actualModes, expectedModes) {
						t.Fatalf("expected session SQL modes %v, got %v", expectedModes, actualModes)
					}

				})

				t.Run("Merge/"+test.name, func(t *testing.T) {

					_, err := admin.db.ExecContext(t.Context(), "DELETE FROM string_values")
					if err != nil {
						t.Fatalf("expected cleared fixtures, got %v", err)
					}
					_, err = admin.db.ExecContext(t.Context(), "INSERT INTO string_values VALUES (1, ?)", "safe")
					if err != nil {
						t.Fatalf("expected inserted fixture, got %v", err)
					}

					destination := newConnector(t)
					mergeTable := table
					mergeTable.Columns = slices.Clone(table.Columns)
					mergeTable.Columns[1].Type = test.typ
					// Update the existing row and insert a new row in the same batch.
					err = destination.Merge(t.Context(), mergeTable, [][]any{{1, test.value}, {2, test.value}})
					if err != nil {
						t.Fatalf("expected merged values, got %v", err)
					}

					rows, err := admin.db.QueryContext(t.Context(), "SELECT id, value FROM string_values ORDER BY id")
					if err != nil {
						t.Fatalf("expected stored values, got %v", err)
					}
					defer rows.Close()
					var expected sql.NullString
					if test.stored != nil {
						expected = sql.NullString{String: test.stored.(string), Valid: true}
					}
					count := 0
					for rows.Next() {
						count++
						var id int
						var value sql.NullString
						if err := rows.Scan(&id, &value); err != nil {
							t.Fatalf("expected stored row, got %v", err)
						}
						if id != count || value != expected {
							t.Fatalf("expected row %d with value %#v, got ID %d and value %#v",
								count, expected, id, value)
						}
					}
					err = rows.Err()
					if err != nil {
						t.Fatalf("expected complete stored rows, got %v", err)
					}
					if count != 2 {
						t.Fatalf("expected two stored rows, got %d", count)
					}

				})
			}

			t.Run("SQLLiteralTypes", func(t *testing.T) {

				_, err := admin.db.ExecContext(t.Context(), "DELETE FROM typed_values")
				if err != nil {
					t.Fatalf("expected cleared typed fixtures, got %v", err)
				}
				_, err = admin.db.ExecContext(t.Context(),
					"INSERT INTO typed_values VALUES (1, ?, ?, ?, ?, ?), (2, ?, ?, ?, ?, ?)",
					"safe", "{}", "2025-01-01 00:00:00", "2025-01-01", "00:00:00",
					"AbC 雪🍎", document, "2026-10-23 16:35:12.123456", "2026-10-23", "16:35:12.123456")
				if err != nil {
					t.Fatalf("expected inserted typed fixtures, got %v", err)
				}

				for _, test := range []struct {
					column string
					value  any
					typ    types.Type
				}{
					{"text_value", "abc 雪🍎", types.String()},
					{"document", json.Value(document), types.JSON()},
					{"datetime_value", time.Date(2026, 10, 23, 16, 35, 12, 123456000, time.UTC), types.DateTime()},
					{"date_value", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), types.Date()},
					{"time_value", time.Date(1970, 1, 1, 16, 35, 12, 123456000, time.UTC), types.Time()},
				} {
					t.Run(test.column, func(t *testing.T) {
						literal := connector.SQLLiteral(test.value, test.typ)
						if test.typ.Kind() == types.JSONKind {
							literal = "CAST(" + literal + " AS JSON)"
						}
						query := "SELECT COUNT(*) AS matches_count, COALESCE(MIN(id), 0) AS matched_id " +
							"FROM typed_values WHERE " + test.column + " = " + literal
						rows, _, err := connector.Query(t.Context(), query)
						if err != nil {
							t.Fatalf("expected typed literal query, got %v", err)
						}
						defer rows.Close()
						if !rows.Next() {
							t.Fatalf("expected typed literal result, got %v", rows.Err())
						}
						var count, id int
						if err := rows.Scan(&count, &id); err != nil {
							t.Fatalf("expected typed literal query result, got %v", err)
						}
						if count != 1 || id != 2 {
							t.Fatalf("expected one matching row with ID 2, got %d rows with minimum ID %d", count, id)
						}
					})
				}

			})

			const mixedOperationsError = "connectors/mysql: cannot mix Query with Columns or Merge on the same instance"
			for _, first := range operations {
				for _, second := range operations {
					t.Run("Operations/"+first.name+"Then"+second.name, func(t *testing.T) {

						connector := newConnector(t)
						err := first.run(t.Context(), connector)
						if err != nil {
							t.Fatalf("expected initial %s, got %v", first.name, err)
						}

						shouldFail := first.isQuery != second.isQuery
						err = second.run(t.Context(), connector)
						if err != nil {
							if !shouldFail {
								t.Fatalf("expected compatible operations, got %v", err)
							}
							if err.Error() != mixedOperationsError {
								t.Fatalf("expected %q, got %q", mixedOperationsError, err.Error())
							}
							return
						}
						if shouldFail {
							t.Fatalf("expected %q, got no error", mixedOperationsError)
						}

					})
				}
			}

			t.Run("TestConnection", func(t *testing.T) {
				err := testConnection(t.Context(), &settings, connector.env.Dial)
				if err != nil {
					t.Fatalf("expected connection check, got %v", err)
				}
			})

			var actualGlobalMode string
			err = admin.db.QueryRowContext(t.Context(), "SELECT @@GLOBAL.sql_mode").Scan(&actualGlobalMode)
			if err != nil {
				t.Fatalf("expected server SQL mode after connector operations, got %v", err)
			}
			if actualGlobalMode != globalMode {
				t.Fatalf("expected unchanged global SQL mode %q, got %q", globalMode, actualGlobalMode)
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
		{"FLOAT", float32(1.123), types.Float(32).Real(), float64(float32(1.123))},
		{"DOUBLE", 390.491835234, types.Float(64).Real(), 390.491835234},
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

	// Open the destination connector.
	settings := innerSettings{
		Host:     host,
		Port:     int(port.Num()),
		Username: username,
		Password: password,
		Database: database,
	}
	connector, err := testconnector.NewDatabase[*MySQL]("mysql", settings)
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	err = connector.openDB(t.Context(), true)
	if err != nil {
		t.Fatalf("expected open destination database, got %v", err)
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
	source, err := testconnector.NewDatabase[*MySQL]("mysql", settings)
	if err != nil {
		t.Fatalf("expected source connector, got %v", err)
	}
	defer source.Close()
	rows, columns, err := source.Query(t.Context(), query.String())
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
