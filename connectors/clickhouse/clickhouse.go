// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package clickhouse provides a connector for ClickHouse.
// (https://clickhouse.com/docs/)
//
// ClickHouse is a trademark of ClickHouse, Inc.
// This connector is not affiliated with or endorsed by ClickHouse, Inc.
package clickhouse

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterDatabase(connectors.DatabaseSpec{
		Code:        "clickhouse",
		Label:       "ClickHouse",
		Categories:  connectors.CategoryDatabase,
		SampleQuery: "SELECT *\nFROM users\n",
		Documentation: connectors.Documentation{
			Source: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
			Destination: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new connector instance for ClickHouse.
func New(env *connectors.DatabaseEnv) (*ClickHouse, error) {
	c := ClickHouse{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for ClickHouse")
		}
	}
	return &c, nil
}

type ClickHouse struct {
	env      *connectors.DatabaseEnv
	settings *innerSettings
	db       driver.Conn
}

// Close closes the database.
func (ch *ClickHouse) Close() error {
	if ch.db == nil {
		return nil
	}
	return ch.db.Close()
}

// Columns returns the columns of the given table.
func (ch *ClickHouse) Columns(ctx context.Context, table string) ([]connectors.Column, error) {
	var err error
	table, err = quoteTable(table)
	if err != nil {
		return nil, err
	}
	// The "SELECT * FROM table" query does not return MATERIALIZED columns.
	// See issue https://github.com/krenalis/krenalis/issues/1417.
	rows, columns, err := ch.query(ctx, "SELECT * FROM "+table+" LIMIT 0", true)
	if err != nil {
		return nil, err
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}
	return columns, nil
}

// Merge performs batch insert and update operations on the specified table,
// basing on the table keys.
func (ch *ClickHouse) Merge(ctx context.Context, table connectors.Table, rows [][]any) error {
	if err := ch.openDB(); err != nil {
		return err
	}
	// Merge rows.
	return merge(ctx, ch.db, table, rows)
}

// Query executes the given query and returns the resulting rows and columns.
func (ch *ClickHouse) Query(ctx context.Context, query string) (connectors.Rows, []connectors.Column, error) {
	return ch.query(ctx, query, false)
}

// SQLLiteral returns the SQL literal representation of v according to the
// provided Meergo type t. It supports nil values and the following Meergo
// types: string, datetime, date, and json.
func (ch *ClickHouse) SQLLiteral(value any, typ types.Type) string {
	if value == nil {
		return "NULL"
	}
	var b strings.Builder
	quoteValue(&b, value, typ)
	return b.String()
}

// ServeUI serves the connector's user interface.
func (ch *ClickHouse) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if ch.settings == nil {
			s.Port = 9000
		} else {
			s = *ch.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ch.saveSettings(ctx, settings, false)
	case "test":
		return nil, ch.saveSettings(ctx, settings, true)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "host", Label: "Host", Placeholder: "localhost", Type: "text", MinLength: 1, MaxLength: 253},
			&connectors.Input{Name: "port", Label: "Port", Placeholder: "9000", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5, HelpText: "Native ClickHouse protocol port (9000 default; 9440 with TLS)"},
			&connectors.Input{Name: "username", Label: "Username", Placeholder: "default", Type: "text", MinLength: 1, MaxLength: 64},
			&connectors.Input{Name: "password", Label: "Password", Placeholder: "", Type: "password", MinLength: 1, MaxLength: 100},
			&connectors.Input{Name: "database", Label: "Database name", Placeholder: "default", Type: "text", MinLength: 1, MaxLength: 64},
		},
		Settings: settings,
		Buttons: []connectors.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

// openDB opens the database. If the database is already open it does nothing.
func (ch *ClickHouse) openDB() error {
	if ch.db != nil {
		return nil
	}
	db, err := clickhouse.Open(ch.settings.options())
	if err != nil {
		return err
	}
	ch.db = db
	return nil
}

// query executes the given query and returns the resulting rows and columns.
// writable indicates whether the resulting columns should be marked as
// writable.
func (ch *ClickHouse) query(ctx context.Context, query string, writable bool) (connectors.Rows, []connectors.Column, error) {
	if err := ch.openDB(); err != nil {
		return nil, nil, err
	}
	rows, err := ch.db.Query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	columnTypes := rows.ColumnTypes()
	columns := make([]connectors.Column, len(columnTypes))
	for i, c := range columnTypes {
		typ, nullable, issue := propertyType(c)
		if !typ.Valid() {
			columns[i].Issue = issue
			continue
		}
		if !types.IsValidPropertyPath(c.Name()) {
			columns[i].Issue = fmt.Sprintf("Column %q does not have a valid property name. Valid names start with a letter or underscore, followed by only letters, numbers, or underscores.", c.Name())
			continue
		}
		columns[i].Name = c.Name()
		columns[i].Type = typ
		columns[i].Nullable = nullable
		columns[i].Writable = writable
	}
	return rows, columns, nil
}

// saveSettings saves the settings. If test is true, it validates only the
// options without saving it.
func (ch *ClickHouse) saveSettings(ctx context.Context, settings json.Value, test bool) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return connectors.NewInvalidSettingsError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65535 {
		return connectors.NewInvalidSettingsError("port must be in range [1,65535]")
	}
	// Validate Username.
	if n := len(s.Username); n > 64 {
		return connectors.NewInvalidSettingsError("username length in bytes must be in range [0,64]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n > 100 {
		return connectors.NewInvalidSettingsError("password length must be in range [0,100]")
	}
	// Validate Database.
	if n := len(s.Database); n > 64 {
		return connectors.NewInvalidSettingsError("database length in bytes must be in range [0,64]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ch.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ch.settings = &s
	return nil
}

type innerSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

// options returns the connection options, from s.
func (s *innerSettings) options() *clickhouse.Options {
	return &clickhouse.Options{
		Addr: []string{net.JoinHostPort(s.Host, strconv.Itoa(s.Port))},
		Auth: clickhouse.Auth{
			Database: s.Database,
			Username: s.Username,
			Password: s.Password,
		},
	}
}

// propertyType returns the property type of the column type and a boolean
// indicating if it is nullable. If the type is not supported, it returns an
// invalid type and the issue message.
func propertyType(t driver.ColumnType) (types.Type, bool, string) {
	typ, nullable := columnType(t.DatabaseTypeName())
	if !typ.Valid() {
		issue := fmt.Sprintf("Column %q has an unsupported type %q.", t.Name(), t.DatabaseTypeName())
		return types.Type{}, false, issue
	}
	return typ, nullable, ""
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *innerSettings) error {
	conn, err := clickhouse.Open(settings.options())
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Ping(ctx)
}
