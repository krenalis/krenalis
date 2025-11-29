// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package snowflake provides a connector for Snowflake.
// (https://docs.snowflake.com/)
//
// Snowflake is a trademark of Snowflake Inc.
// This connector is not affiliated with or endorsed by Snowflake Inc.
package snowflake

import (
	"context"
	"database/sql"
	"database/sql/driver"
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/snowflakedb/gosnowflake"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterDatabase(connectors.DatabaseSpec{
		Code:        "snowflake",
		Label:       "Snowflake",
		Categories:  connectors.CategoryDatabase,
		SampleQuery: "SELECT *\nFROM \"USERS\"\n",
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

// New returns a new connector instance for Snowflake.
func New(env *connectors.DatabaseEnv) (*Snowflake, error) {
	c := Snowflake{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Snowflake")
		}
	}
	return &c, nil
}

type Snowflake struct {
	env      *connectors.DatabaseEnv
	settings *innerSettings
	db       *sql.DB
}

// Close closes the database.
func (sf *Snowflake) Close() error {
	if sf.db == nil {
		return nil
	}
	return sf.db.Close()
}

// Columns returns the columns of the given table.
func (sf *Snowflake) Columns(ctx context.Context, table string) ([]connectors.Column, error) {
	rows, columns, err := sf.query(ctx, "SELECT * FROM "+quoteTable(table)+" LIMIT 0", true)
	if err != nil {
		return nil, err
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}
	return columns, nil
}

// Merge performs batch insert, update, and delete operations on the specified
// table.
func (sf *Snowflake) Merge(ctx context.Context, table connectors.Table, rows [][]any) error {
	if err := sf.openDB(); err != nil {
		return err
	}
	// Acquire a connection.
	conn, err := sf.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Merge rows.
	return merge(ctx, conn, table, rows, nil)
}

// Query executes the given query and returns the resulting rows and columns.
func (sf *Snowflake) Query(ctx context.Context, query string) (connectors.Rows, []connectors.Column, error) {
	return sf.query(ctx, query, false)
}

// QuoteTime returns a quoted time value for the specified type or "NULL" if the
// value is nil.
func (sf *Snowflake) QuoteTime(value any, typ types.Type) string {
	if value == nil {
		return "NULL"
	}
	var b strings.Builder
	quoteValue(&b, value, typ)
	return b.String()
}

// ServeUI serves the connector's user interface.
func (sf *Snowflake) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if sf.settings != nil {
			s = *sf.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, sf.saveSettings(ctx, settings, false)
	case "test":
		return nil, sf.saveSettings(ctx, settings, true)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "Account", Label: "Account Identifier", Placeholder: "ABCDEFG-TUVWXYZ", Type: "text", MinLength: 3, MaxLength: 255},
			&connectors.Input{Name: "Username", Label: "User Name", Placeholder: "USERNAME", Type: "text", MinLength: 1, MaxLength: 255},
			&connectors.Input{Name: "Password", Label: "Password", Placeholder: "", Type: "password", MinLength: 1, MaxLength: 255},
			&connectors.Input{Name: "Role", Label: "Role", Placeholder: "CUSTOM_ROLE", Type: "text", MinLength: 1, MaxLength: 255},
			&connectors.Input{Name: "Database", Label: "Database", Placeholder: "MY_DATABASE", Type: "text", MinLength: 1, MaxLength: 255},
			&connectors.Input{Name: "Schema", Label: "Schema", Placeholder: "PUBLIC", Type: "text", MinLength: 1, MaxLength: 255},
			&connectors.Input{Name: "Warehouse", Label: "Warehouse", Placeholder: "COMPUTE_WH", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Settings: settings,
		Buttons: []connectors.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

type innerSettings struct {
	Account   string
	Username  string
	Password  string
	Role      string
	Database  string
	Schema    string
	Warehouse string
}

// connector returns a driver.Connector from the settings.
func (s *innerSettings) connector() driver.Connector {
	account := s.Account
	if i := strings.IndexByte(account, '.'); i > 0 {
		account = account[:i] + "-" + account[i+1:]
	}
	return gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:          account,
		User:             s.Username,
		Password:         s.Password,
		Role:             s.Role,
		Database:         s.Database,
		Schema:           s.Schema,
		Warehouse:        s.Warehouse,
		Params:           make(map[string]*string),
		DisableTelemetry: true,
	})
}

// openDB opens the database. If the database is already open it does nothing.
func (sf *Snowflake) openDB() error {
	if sf.db != nil {
		return nil
	}
	db := sql.OpenDB(sf.settings.connector())
	db.SetMaxIdleConns(0)
	sf.db = db
	return nil
}

// query executes the given query and returns the resulting rows and columns.
// writable indicates whether the resulting columns should be marked as
// writable.
func (sf *Snowflake) query(ctx context.Context, query string, writable bool) (connectors.Rows, []connectors.Column, error) {
	if err := sf.openDB(); err != nil {
		return nil, nil, err
	}
	rows, err := sf.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		return nil, nil, err
	}
	columns := make([]connectors.Column, len(columnTypes))
	for i, c := range columnTypes {
		typ, issue, err := propertyType(c)
		if err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		if !typ.Valid() {
			columns[i].Issue = issue
			continue
		}
		if !types.IsValidPropertyPath(c.Name()) {
			columns[i].Issue = fmt.Sprintf("Column %q does not have a valid property name. Valid names start with a letter or underscore, followed by only letters, numbers, or underscores.", c.Name())
			continue
		}
		nullable, ok := c.Nullable()
		columns[i].Name = c.Name()
		columns[i].Type = typ
		columns[i].Nullable = nullable || !ok
		columns[i].Writable = writable
	}
	return rows, columns, nil
}

// accountFormat is the format of the account identifier in the settings.
var accountFormat = regexp.MustCompile(`^[a-zA-Z0-9]+[.-][a-zA-Z0-9]+$`)

// saveSettings validates and saves the settings. If test is true, it validates
// only the settings without saving it.
func (sf *Snowflake) saveSettings(ctx context.Context, options json.Value, test bool) error {
	var s innerSettings
	err := options.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Account.
	if n := utf8.RuneCountInString(s.Account); n < 3 || n > 255 {
		return connectors.NewInvalidSettingsError("account identifier length must be in range [3,255]")
	}
	if !accountFormat.MatchString(s.Account) {
		return connectors.NewInvalidSettingsError("account identifier must be in the <organization>.<account> or <organization>-<account> format")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 255 {
		return connectors.NewInvalidSettingsError("username length must be in range [1,255]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 255 {
		return connectors.NewInvalidSettingsError("password length must be in range [1,255]")
	}
	// Validate Role.
	if n := utf8.RuneCountInString(s.Role); n < 1 || n > 255 {
		return connectors.NewInvalidSettingsError("role length must be in range [1,255]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 255 {
		return connectors.NewInvalidSettingsError("database length must be in range [1,255]")
	}
	// Validate Schema.
	if n := utf8.RuneCountInString(s.Schema); n < 1 || n > 255 {
		return connectors.NewInvalidSettingsError("schema length must be in range [1,255]")
	}
	// Validate Warehouse.
	if n := utf8.RuneCountInString(s.Warehouse); n < 1 || n > 255 {
		return connectors.NewInvalidSettingsError("warehouse length must be in range [1,255]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = sf.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	sf.settings = &s
	return nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *innerSettings) error {
	db := sql.OpenDB(settings.connector())
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column with type t.
// If the column type is not supported, it returns an invalid type and an issue
// message.
func propertyType(t *sql.ColumnType) (types.Type, string, error) {
	switch t.DatabaseTypeName() {
	case "ARRAY":
		return types.Array(types.JSON()), "", nil
	case "BOOLEAN":
		return types.Boolean(), "", nil
	case "DATE":
		return types.Date(), "", nil
	case "FIXED":
		precision, scale, ok := t.DecimalSize()
		if !ok {
			return types.Type{}, "", errors.New("cannot get decimal size")
		}
		if precision < 1 || scale < 0 || scale > precision {
			return types.Type{}, "", fmt.Errorf("precision and scale (%d, %d) are invalid", precision, scale)
		}
		if precision > types.MaxDecimalPrecision {
			issue := fmt.Sprintf("Column %q has a precision of %d, which exceeds the maximum supported precision of %d.", t.Name(), precision, types.MaxDecimalPrecision)
			return types.Type{}, issue, nil
		}
		if scale > types.MaxDecimalScale {
			issue := fmt.Sprintf("Column %q has a scale of %d, which exceeds the maximum supported scale of %d.", t.Name(), scale, types.MaxDecimalScale)
			return types.Type{}, issue, nil
		}
		return types.Decimal(int(precision), int(scale)), "", nil
	case "OBJECT":
		return types.Map(types.JSON()), "", nil
	case "REAL":
		return types.Float(64), "", nil
	case "TEXT":
		length, ok := t.Length()
		if !ok {
			return types.Type{}, "", errors.New("cannot get length")
		}
		if length < 0 {
			return types.Type{}, "", errors.New("invalid TEXT length")
		}
		if length > types.MaxTextLen {
			issue := fmt.Sprintf("Column %q is not available because its %d characters exceed the maximum length of %d", t.Name(), length, types.MaxTextLen)
			return types.Type{}, issue, nil
		}
		t := types.Text().WithCharLen(int(length))
		const maxBytesLen = 16_777_216
		if length > maxBytesLen/4 {
			t = t.WithByteLen(min(int(length*4), maxBytesLen))
		}
		return t, "", nil
	case "TIME":
		return types.Time(), "", nil
	case "TIMESTAMP_NTZ":
		return types.DateTime(), "", nil
	case "VARIANT":
		return types.JSON(), "", nil
	}
	issue := fmt.Sprintf("Column %q has an unsupported type %q.", t.Name(), t.DatabaseTypeName())
	return types.Type{}, issue, nil
}
