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
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterDatabase(meergo.DatabaseInfo{
		Name:        "PostgreSQL",
		SampleQuery: "SELECT *\nFROM users\nWHERE ${last_change_time}\nLIMIT ${limit}\n",
		Icon:        icon,
	}, New)
}

// New returns a new PostgreSQL connector instance.
func New(conf *meergo.DatabaseConfig) (*PostgreSQL, error) {
	c := PostgreSQL{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of PostgreSQL connector")
		}
	}
	return &c, nil
}

type PostgreSQL struct {
	conf     *meergo.DatabaseConfig
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
func (ps *PostgreSQL) Columns(ctx context.Context, table string) ([]meergo.Column, error) {
	tables, err := ps.tablesSchemas(ctx, "public", []string{table})
	if err != nil {
		return nil, err
	}
	var columns []meergo.Column
	if len(tables) == 1 {
		columns = tables[0].columns
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table '%s' does not exist", table)
	}
	return columns, nil
}

// LastChangeTimeCondition returns the query condition used for the
// last_change_time placeholder in the form "column >= value" or, if column is
// empty, a true value.
func (ps *PostgreSQL) LastChangeTimeCondition(column string, typ types.Type, value any) string {
	if column == "" {
		return "TRUE"
	}
	b := strings.Builder{}
	b.WriteString(quoteIdent(column))
	b.WriteString(` >= `)
	quoteValue(&b, value, typ)
	return b.String()
}

// Merge performs batch insert and update operations on the specified table,
// basing on the table keys.
func (ps *PostgreSQL) Merge(ctx context.Context, table meergo.Table, rows [][]any) error {
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
func (ps *PostgreSQL) Query(ctx context.Context, query string) (meergo.Rows, []meergo.Column, error) {
	if err := ps.openDB(ctx); err != nil {
		return nil, nil, err
	}
	rows, err := ps.pool.Query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]meergo.Column, len(fieldDescriptions))
	for i, field := range fieldDescriptions {
		typ, err := ps.propertyType(ctx, field)
		if err != nil {
			rows.Close()
			return nil, nil, err
		}
		columns[i] = meergo.Column{
			Name: field.Name,
			Type: typ,
			// Nullable is always considered true, as the PostgreSQL driver does
			// not have information about nullability of returned columns.
			Nullable: true,
		}
	}
	return withCloseError{rows}, columns, nil
}

type withCloseError struct {
	pgx.Rows
}

func (rows withCloseError) Close() error {
	rows.Rows.Close()
	return nil
}

// ServeUI serves the connector's user interface.
func (ps *PostgreSQL) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if ps.settings == nil {
			s.Port = 5432
		} else {
			s = *ps.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ps.saveSettings(ctx, settings, false)
	case "test":
		return nil, ps.saveSettings(ctx, settings, true)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&meergo.Input{Name: "Port", Label: "Port", Placeholder: "5432", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&meergo.Input{Name: "Username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 63},
			&meergo.Input{Name: "Password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 100},
			&meergo.Input{Name: "Database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 63},
		},
		Settings: settings,
		Buttons: []meergo.Button{
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
}

// dsn returns the connection string, from s, in the URL format.
func (s *innerSettings) dsn() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(s.Username, s.Password),
		Host:   net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:   "/" + url.PathEscape(s.Database),
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
		return meergo.NewInvalidsettingsError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return meergo.NewInvalidsettingsError("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 63 {
		return meergo.NewInvalidsettingsError("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return meergo.NewInvalidsettingsError("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return meergo.NewInvalidsettingsError("database length in bytes must be in range [1,63]")
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
func testConnection(ctx context.Context, settings *innerSettings) error {
	db, err := sql.Open("pgx", settings.dsn())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column type t.
func (ps *PostgreSQL) propertyType(ctx context.Context, fd pgconn.FieldDescription) (types.Type, error) {
	switch fd.DataTypeOID {
	case pgtype.BoolOID:
		return types.Boolean(), nil
	case pgtype.Int8OID:
		return types.Int(64), nil
	case pgtype.Int2OID:
		return types.Int(16), nil
	case pgtype.Int4OID:
		return types.Int(32), nil
	case pgtype.Float4OID:
		return types.Float(32), nil
	case pgtype.Float8OID:
		return types.Float(64), nil
	case pgtype.NumericOID:
		mod := fd.TypeModifier - 4
		precision, scale := int((mod>>16)&0xffff), int(mod&0xffff)
		if 1 <= precision && precision <= types.MaxDecimalPrecision && 0 <= scale && scale <= types.MaxDecimalScale && scale <= precision {
			return types.Decimal(precision, scale), nil
		}
	case pgtype.TimestampOID, pgtype.TimestamptzOID:
		return types.DateTime(), nil
	case pgtype.DateOID:
		return types.Date(), nil
	case pgtype.TimeOID, pgtype.TimetzOID:
		return types.Time(), nil
	case pgtype.UUIDOID:
		return types.UUID(), nil
	case pgtype.JSONOID, pgtype.JSONBOID:
		return types.JSON(), nil
	case pgtype.InetOID:
		return types.Inet(), nil
	case pgtype.BPCharOID, pgtype.VarcharOID:
		length := int(fd.TypeModifier - 4)
		if 1 <= length && length <= types.MaxTextLen {
			return types.Text().WithCharLen(length), nil
		}
		return types.Text(), nil
	case pgtype.TextOID, pgtype.ByteaOID:
		return types.Text(), nil
	}
	conn, err := ps.pool.Acquire(ctx)
	if err != nil {
		return types.Type{}, err
	}
	defer conn.Release()
	var name string
	if t, ok := conn.Conn().TypeMap().TypeForOID(fd.DataTypeOID); ok {
		name = strings.ToUpper(t.Name)
		if strings.HasPrefix(name, "_") {
			name = "array"
		}
	} else {
		name = strconv.FormatUint(uint64(fd.DataTypeOID), 10)
	}
	return types.Type{}, meergo.NewUnsupportedColumnTypeError(fd.Name, name)
}
