//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package clickhouse implements the ClickHouse connector.
// (https://clickhouse.com/docs/)
package clickhouse

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Database and UIHandler interfaces.
var _ interface {
	chichi.Database
	chichi.UIHandler
} = (*ClickHouse)(nil)

func init() {
	chichi.RegisterDatabase(chichi.DatabaseInfo{
		Name:        "ClickHouse",
		SampleQuery: "SELECT * FROM users LIMIT ${limit}",
		Icon:        icon,
	}, New)
}

// New returns a new ClickHouse connector instance.
func New(conf *chichi.DatabaseConfig) (*ClickHouse, error) {
	c := ClickHouse{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of ClickHouse connector")
		}
	}
	return &c, nil
}

type ClickHouse struct {
	conf     *chichi.DatabaseConfig
	settings *Settings
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
func (ch *ClickHouse) Columns(ctx context.Context, table string) ([]types.Property, error) {
	var err error
	table, err = quoteTable(table)
	if err != nil {
		return nil, err
	}
	rows, columns, err := ch.query(ctx, "SELECT * FROM "+table)
	if err != nil {
		return nil, err
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}
	return columns, nil
}

// Query executes the given query and returns the resulting rows and columns.
func (ch *ClickHouse) Query(ctx context.Context, query string) (chichi.Rows, []types.Property, error) {
	return ch.query(ctx, query)
}

// ServeUI serves the connector's user interface.
func (ch *ClickHouse) ServeUI(ctx context.Context, event string, values []byte) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if ch.settings == nil {
			s.Port = 9000
		} else {
			s = *ch.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, ch.saveValues(ctx, values, false)
	case "test":
		return nil, ch.saveValues(ctx, values, true)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&chichi.Input{Name: "Port", Label: "Port", Placeholder: "9000", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&chichi.Input{Name: "Username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 64},
			&chichi.Input{Name: "Password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 100},
			&chichi.Input{Name: "Database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 64},
		},
		Values: values,
		Buttons: []chichi.Button{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return ui, nil
}

// Upsert creates or updates the provided rows in the specified table.
func (ch *ClickHouse) Upsert(ctx context.Context, table string, rows []map[string]any, columns []types.Property) error {

	var err error
	table, err = quoteTable(table)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	for i, column := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(column.Name)
		b.WriteByte('"')
	}
	b.WriteString(") VALUES ")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("(")
		for j, column := range columns {
			if j > 0 {
				b.WriteByte(',')
			}
			if v, ok := row[column.Name]; ok {
				quoteValue(&b, v, column.Type)
			} else {
				b.WriteString(`DEFAULT`)
			}
		}
		b.WriteByte(')')
	}
	query := b.String()

	if err = ch.openDB(); err != nil {
		return err
	}
	err = ch.db.Exec(ctx, query)

	return err
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
func (ch *ClickHouse) query(ctx context.Context, query string) (chichi.Rows, []types.Property, error) {
	if err := ch.openDB(); err != nil {
		return nil, nil, err
	}
	rows, err := ch.db.Query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	columnTypes := rows.ColumnTypes()
	columns := make([]types.Property, len(columnTypes))
	for i, c := range columnTypes {
		typ, nullable, err := propertyType(c)
		if err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		columns[i] = types.Property{
			Name:     c.Name(),
			Type:     typ,
			Nullable: nullable,
		}
	}
	return rows, columns, nil
}

// saveValues saves the user-entered values as settings. If test is true, it
// validates only the values without saving it.
func (ch *ClickHouse) saveValues(ctx context.Context, values []byte, test bool) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return chichi.NewInvalidUIValuesError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return chichi.NewInvalidUIValuesError("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 64 {
		return chichi.NewInvalidUIValuesError("username length in bytes must be in range [1,64]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return chichi.NewInvalidUIValuesError("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 64 {
		return chichi.NewInvalidUIValuesError("database length in bytes must be in range [1,64]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ch.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ch.settings = &s
	return nil
}

type Settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// options returns the connection options, from s.
func (s *Settings) options() *clickhouse.Options {
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
// indicating if it is nullable.
func propertyType(t driver.ColumnType) (types.Type, bool, error) {
	typ, nullable := columnType(t.DatabaseTypeName())
	if !typ.Valid() {
		return types.Type{}, false, chichi.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
	}
	return typ, nullable, nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *Settings) error {
	conn, err := clickhouse.Open(settings.options())
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Ping(ctx)
}
