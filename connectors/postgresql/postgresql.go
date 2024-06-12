//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package postgresql implements the PostgreSQL connector.
// (https://www.postgresql.org/docs/15/)
package postgresql

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"

	"github.com/jackc/pgx/v5/stdlib"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Database and UIHandler interfaces.
var _ interface {
	chichi.Database
	chichi.UIHandler
} = (*PostgreSQL)(nil)

func init() {
	chichi.RegisterDatabase(chichi.DatabaseInfo{
		Name:        "PostgreSQL",
		SampleQuery: "SELECT *\nFROM users\nLIMIT ${limit}\n",
		Icon:        icon,
	}, New)
}

// New returns a new PostgreSQL connector instance.
func New(conf *chichi.DatabaseConfig) (*PostgreSQL, error) {
	c := PostgreSQL{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of PostgreSQL connector")
		}
	}
	return &c, nil
}

type PostgreSQL struct {
	conf     *chichi.DatabaseConfig
	settings *Settings
	db       *sql.DB
}

// Close closes the database.
func (ps *PostgreSQL) Close() error {
	if ps.db == nil {
		return nil
	}
	return ps.db.Close()
}

// Columns returns the columns of the given table.
func (ps *PostgreSQL) Columns(ctx context.Context, table string) ([]types.Property, error) {
	if err := ps.openDB(); err != nil {
		return nil, err
	}
	conn, err := ps.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	var columns []types.Property
	err = conn.Raw(func(driverConn any) error {
		conn := driverConn.(*stdlib.Conn)
		tx, err := conn.Conn().Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		tables, err := tablesSchemas(ctx, tx, "public", []string{table})
		if err != nil {
			return err
		}
		if len(tables) == 1 {
			columns = tables[0].columns
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table '%s' does not exist", table)
	}
	return columns, nil
}

// Query executes the given query and returns the resulting rows and columns.
func (ps *PostgreSQL) Query(ctx context.Context, query string) (chichi.Rows, []types.Property, error) {
	if err := ps.openDB(); err != nil {
		return nil, nil, err
	}
	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		return nil, nil, err
	}
	columns := make([]types.Property, len(columnTypes))
	for i, column := range columnTypes {
		typ, err := propertyType(column)
		if err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		columns[i] = types.Property{
			Name: column.Name(),
			Type: typ,
			// Nullable is always considered true, as the PostgreSQL driver does
			// not have information about nullability of returned columns.
			Nullable: true,
		}
	}
	return rows, columns, nil
}

// ServeUI serves the connector's user interface.
func (ps *PostgreSQL) ServeUI(ctx context.Context, event string, values []byte, role chichi.Role) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if ps.settings == nil {
			s.Port = 5432
		} else {
			s = *ps.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, ps.saveValues(ctx, values, false)
	case "test":
		return nil, ps.saveValues(ctx, values, true)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&chichi.Input{Name: "Port", Label: "Port", Placeholder: "5432", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&chichi.Input{Name: "Username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 63},
			&chichi.Input{Name: "Password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 100},
			&chichi.Input{Name: "Database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 63},
		},
		Values: values,
		Buttons: []chichi.Button{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

// Upsert creates or updates the provided rows in the specified table.
func (ps *PostgreSQL) Upsert(ctx context.Context, table string, rows []map[string]any, columns []types.Property) error {

	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(quoteTable(table))
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
	b.WriteString(` ON CONFLICT ("id") DO UPDATE SET `)
	i := 0
	for _, column := range columns {
		if column.Name == "id" {
			continue
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('"')
		b.WriteString(column.Name)
		b.WriteString(`" = EXCLUDED."`)
		b.WriteString(column.Name)
		b.WriteByte('"')
		i++
	}
	query := b.String()

	if err := ps.openDB(); err != nil {
		return err
	}
	_, err := ps.db.ExecContext(ctx, query)

	return err
}

type Settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// dsn returns the connection string, from s, in the URL format.
func (s *Settings) dsn() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(s.Username, s.Password),
		Host:   net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:   "/" + url.PathEscape(s.Database),
	}
	return u.String()
}

// openDB opens the database. If the database is already open it does nothing.
func (ps *PostgreSQL) openDB() error {
	if ps.db != nil {
		return nil
	}
	db, err := sql.Open("pgx", ps.settings.dsn())
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(0)
	ps.db = db
	return nil
}

// saveValues saves the user-entered values as settings. If test is true, it
// validates only the values without saving it.
func (ps *PostgreSQL) saveValues(ctx context.Context, values []byte, test bool) error {
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
	if n := len(s.Username); n < 1 || n > 63 {
		return chichi.NewInvalidUIValuesError("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return chichi.NewInvalidUIValuesError("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return chichi.NewInvalidUIValuesError("database length in bytes must be in range [1,63]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ps.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ps.settings = &s
	return nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *Settings) error {
	db, err := sql.Open("pgx", settings.dsn())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column type t.
func propertyType(t *sql.ColumnType) (types.Type, error) {
	switch t.DatabaseTypeName() {
	case "BOOL":
		return types.Boolean(), nil
	case "BYTEA", "TEXT":
		return types.Text(), nil
	case "CHAR", "VARCHAR":
		length, ok := t.Length()
		if !ok {
			return types.Type{}, errors.New("cannot get length")
		}
		if length > 0 {
			return types.Text().WithCharLen(int(length)), nil
		}
		return types.Text(), nil
	case "DATE":
		return types.Date(), nil
	case "FLOAT4":
		return types.Float(32), nil
	case "FLOAT8":
		return types.Float(64), nil
	case "INET":
		return types.Inet(), nil
	case "INT2":
		return types.Int(16), nil
	case "INT4":
		return types.Int(32), nil
	case "INT8":
		return types.Int(64), nil
	case "JSON", "JSONB":
		return types.JSON(), nil
	case "NUMERIC":
		precision, scale, ok := t.DecimalSize()
		if !ok {
			return types.Type{}, errors.New("cannot get decimal size")
		}
		if precision > types.MaxDecimalPrecision || scale > types.MaxDecimalScale {
			return types.Type{}, fmt.Errorf("PostgreSQL type %s(%d,%d) is not supported",
				t.DatabaseTypeName(), precision, scale)
		}
		return types.Decimal(int(precision), int(scale)), nil
	case "TIME", "1266":
		// 1266: time with time zone.
		return types.Time(), nil
	case "TIMESTAMP", "TIMESTAMPTZ":
		return types.DateTime(), nil
	case "UUID":
		return types.UUID(), nil
	}
	return types.Type{}, chichi.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
}
