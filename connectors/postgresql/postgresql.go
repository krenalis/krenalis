// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package postgresql provides a connector for PostgreSQL.
// (https://www.postgresql.org/docs/16/)
package postgresql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterDatabase(connectors.DatabaseSpec{
		Code:        "postgresql",
		Label:       "PostgreSQL",
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

// New returns a new connector instance for PostgreSQL.
func New(env *connectors.DatabaseEnv) (*PostgreSQL, error) {
	c := PostgreSQL{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for PostgreSQL")
		}
	}
	return &c, nil
}

type PostgreSQL struct {
	env      *connectors.DatabaseEnv
	settings *innerSettings
	pool     *pgxpool.Pool
}

// Close closes the database.
func (ps *PostgreSQL) Close() error {
	if ps.pool == nil {
		return nil
	}
	ps.pool.Close()
	return nil
}

// Columns returns the columns of the given table.
func (ps *PostgreSQL) Columns(ctx context.Context, table string) ([]connectors.Column, error) {
	columns, err := ps.columns(ctx, ps.settings.Schema, table)
	if err != nil {
		return nil, err
	}
	return columns, nil
}

// Merge performs batch insert and update operations on the specified table,
// basing on the table keys.
func (ps *PostgreSQL) Merge(ctx context.Context, table connectors.Table, rows [][]any) error {
	if err := ps.openDB(ctx); err != nil {
		return err
	}
	// Acquire a connection.
	conn, err := ps.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	// Merge rows.
	return merge(ctx, conn, table, rows, nil)
}

// Query executes the given query and returns the resulting rows and columns.
func (ps *PostgreSQL) Query(ctx context.Context, query string) (connectors.Rows, []connectors.Column, error) {
	if err := ps.openDB(ctx); err != nil {
		return nil, nil, err
	}
	rows, err := ps.pool.Query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]connectors.Column, len(fieldDescriptions))
	for i, c := range fieldDescriptions {
		typ, issue, err := ps.propertyType(ctx, c)
		if err != nil {
			rows.Close()
			return nil, nil, err
		}
		if !typ.Valid() {
			columns[i].Issue = issue
			continue
		}
		if !types.IsValidPropertyPath(c.Name) {
			columns[i].Issue = fmt.Sprintf("Column %q does not have a valid property name. Valid names start with a letter or underscore, followed by only letters, numbers, or underscores.", c.Name)
			continue
		}
		columns[i].Name = c.Name
		columns[i].Type = typ
		// Nullable is always considered true, as the PostgreSQL driver does
		// not have information about nullability of returned columns.
		columns[i].Nullable = true
	}
	return withCloseError{rows}, columns, nil
}

// QuoteTime returns a quoted time value for the specified type or "NULL" if the
// value is nil.
func (ps *PostgreSQL) QuoteTime(value any, typ types.Type) string {
	if value == nil {
		return "NULL"
	}
	var b strings.Builder
	quoteValue(&b, value, typ)
	return b.String()
}

type withCloseError struct {
	pgx.Rows
}

func (rows withCloseError) Close() error {
	rows.Rows.Close()
	return nil
}

// ServeUI serves the connector's user interface.
func (ps *PostgreSQL) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if ps.settings == nil {
			s.Port = 5432
			s.Schema = "public"
		} else {
			s = *ps.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ps.saveSettings(ctx, settings, false)
	case "test":
		return nil, ps.saveSettings(ctx, settings, true)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&connectors.Input{Name: "Port", Label: "Port", Placeholder: "5432", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&connectors.Input{Name: "Username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 63},
			&connectors.Input{Name: "Password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 100},
			&connectors.Input{Name: "Database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 63},
			&connectors.Input{Name: "Schema", Label: "Schema name", Placeholder: "public", Type: "text", MinLength: 1, MaxLength: 63},
		},
		Settings: settings,
		Buttons: []connectors.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

type innerSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

// dsn returns the connection string, from s, in the URL format.
func (s *innerSettings) dsn() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(s.Username, s.Password),
		Host:     net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:     "/" + url.PathEscape(s.Database),
		RawQuery: "search_path=" + url.QueryEscape(s.Schema),
	}
	return u.String()
}

// openDB opens the database. If the database is already open it does nothing.
func (ps *PostgreSQL) openDB(ctx context.Context) error {
	if ps.pool != nil {
		return nil
	}
	config, err := pgxpool.ParseConfig(ps.settings.dsn())
	if err != nil {
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	ps.pool = pool
	return nil
}

// saveSettings validates and saves the settings. If test is true, it validates
// only the settings without saving it.
func (ps *PostgreSQL) saveSettings(ctx context.Context, settings json.Value, test bool) error {
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
	if n := len(s.Username); n < 1 || n > 63 {
		return connectors.NewInvalidSettingsError("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return connectors.NewInvalidSettingsError("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return connectors.NewInvalidSettingsError("database length in bytes must be in range [1,63]")
	}
	// Validate Schema.
	if n := len(s.Schema); n < 1 || n > 63 {
		return connectors.NewInvalidSettingsError("schema length in bytes must be in range [1,63]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ps.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ps.settings = &s
	return nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *innerSettings) error {
	db, err := sql.Open("pgx", settings.dsn())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column with type t.
// If the column type is not supported, it returns an invalid type and an issue
// message.
func (ps *PostgreSQL) propertyType(ctx context.Context, fd pgconn.FieldDescription) (types.Type, string, error) {
	switch fd.DataTypeOID {
	case pgtype.BoolOID:
		return types.Boolean(), "", nil
	case pgtype.Int8OID:
		return types.Int(64), "", nil
	case pgtype.Int2OID:
		return types.Int(16), "", nil
	case pgtype.Int4OID:
		return types.Int(32), "", nil
	case pgtype.Float4OID:
		return types.Float(32), "", nil
	case pgtype.Float8OID:
		return types.Float(64), "", nil
	case pgtype.NumericOID:
		mod := fd.TypeModifier - 4
		precision, scale := int((mod>>16)&0xffff), int(mod&0xffff)
		if precision < 1 || scale < 0 || scale > precision {
			return types.Type{}, "", fmt.Errorf("precision and scale (%d, %d) are invalid", precision, scale)
		}
		if precision > types.MaxDecimalPrecision {
			issue := fmt.Sprintf("Column %q has a precision of %d, which exceeds the maximum supported precision of %d.", fd.Name, precision, types.MaxDecimalPrecision)
			return types.Type{}, issue, nil
		}
		if scale > types.MaxDecimalScale {
			issue := fmt.Sprintf("Column %q has a scale of %d, which exceeds the maximum supported scale of %d.", fd.Name, scale, types.MaxDecimalScale)
			return types.Type{}, issue, nil
		}
		return types.Decimal(precision, scale), "", nil
	case pgtype.TimestampOID, pgtype.TimestamptzOID:
		return types.DateTime(), "", nil
	case pgtype.DateOID:
		return types.Date(), "", nil
	case pgtype.TimeOID, pgtype.TimetzOID:
		return types.Time(), "", nil
	case pgtype.UUIDOID:
		return types.UUID(), "", nil
	case pgtype.JSONOID, pgtype.JSONBOID:
		return types.JSON(), "", nil
	case pgtype.InetOID:
		return types.Inet(), "", nil
	case pgtype.BPCharOID, pgtype.VarcharOID:
		length := int(fd.TypeModifier - 4)
		if 1 <= length && length <= types.MaxStringLen {
			return types.String().WithCharLen(length), "", nil
		}
		return types.String(), "", nil
	case pgtype.TextOID, pgtype.ByteaOID:
		return types.String(), "", nil
	}
	conn, err := ps.pool.Acquire(ctx)
	if err != nil {
		return types.Type{}, "", err
	}
	defer conn.Release()
	var typ string
	if t, ok := conn.Conn().TypeMap().TypeForOID(fd.DataTypeOID); ok {
		typ = strings.ToUpper(t.Name)
		if strings.HasPrefix(typ, "_") {
			typ = "array"
		}
	} else {
		typ = strconv.FormatUint(uint64(fd.DataTypeOID), 10)
	}
	issue := fmt.Sprintf("Column %q has an unsupported type %q.", fd.Name, typ)
	return types.Type{}, issue, nil
}
