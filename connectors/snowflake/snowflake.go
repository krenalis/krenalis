//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package snowflake implements the Snowflake connector.
// (https://docs.snowflake.com/)
package snowflake

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"

	"github.com/snowflakedb/gosnowflake"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Database and UIHandler interfaces.
var _ interface {
	chichi.Database
	chichi.UIHandler
} = (*Snowflake)(nil)

func init() {
	chichi.RegisterDatabase(chichi.DatabaseInfo{
		Name:        "Snowflake",
		SampleQuery: "SELECT * FROM users LIMIT ${limit}",
		Icon:        icon,
	}, New)
}

// New returns a new Snowflake connector instance.
func New(conf *chichi.DatabaseConfig) (*Snowflake, error) {
	c := Snowflake{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Snowflake connector")
		}
	}
	return &c, nil
}

type Snowflake struct {
	conf     *chichi.DatabaseConfig
	settings *Settings
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
func (sf *Snowflake) Columns(ctx context.Context, table string) ([]types.Property, error) {
	rows, columns, err := sf.query(ctx, "SELECT * FROM "+quoteTable(table))
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
func (sf *Snowflake) Query(ctx context.Context, query string) (chichi.Rows, []types.Property, error) {
	return sf.query(ctx, query)
}

// ServeUI serves the connector's user interface.
func (sf *Snowflake) ServeUI(ctx context.Context, event string, values []byte) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if sf.settings != nil {
			s = *sf.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		s, err := validateValues(ctx, values)
		if err != nil {
			return nil, err
		}
		if event == "test" {
			return nil, nil
		}
		return nil, sf.conf.SetSettings(ctx, s)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "Account", Label: "Account", Placeholder: "ABCDEFG-TUVWXYZ", Type: "text", MinLength: 1, MaxLength: 255},
			&chichi.Input{Name: "Username", Label: "Username", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&chichi.Input{Name: "Password", Label: "Password", Placeholder: "", Type: "password", MinLength: 1, MaxLength: 255},
			&chichi.Input{Name: "Database", Label: "Database", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&chichi.Input{Name: "Schema", Label: "Schema", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&chichi.Input{Name: "Warehouse", Label: "Warehouse", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&chichi.Input{Name: "Role", Label: "Role", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
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
func (sf *Snowflake) Upsert(ctx context.Context, table string, rows []map[string]any, columns []types.Property) error {
	return errors.New("not implemented")
}

type Settings struct {
	Account   string
	Username  string
	Password  string
	Warehouse string
	Database  string
	Schema    string
	Role      string
}

// connector returns a driver.Connector from the settings.
func (s *Settings) connector() gosnowflake.Connector {
	return gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:   s.Account,
		User:      s.Username,
		Password:  s.Password,
		Database:  s.Database,
		Schema:    s.Schema,
		Warehouse: s.Warehouse,
		Role:      s.Role,
		Params:    make(map[string]*string),
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
func (sf *Snowflake) query(ctx context.Context, query string) (chichi.Rows, []types.Property, error) {
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
	columns := make([]types.Property, len(columnTypes))
	for i, column := range columnTypes {
		typ, err := propertyType(column)
		if err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		nullable, ok := column.Nullable()
		columns[i] = types.Property{
			Name:     column.Name(),
			Type:     typ,
			Nullable: nullable || !ok,
		}
	}
	return rows, columns, nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *Settings) error {
	db := sql.OpenDB(settings.connector())
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column type t.
func propertyType(t *sql.ColumnType) (types.Type, error) {
	switch t.DatabaseTypeName() {
	case "ARRAY":
		return types.Array(types.JSON()), nil
	case "BOOLEAN":
		return types.Boolean(), nil
	case "DATE":
		return types.Date(), nil
	case "FIXED":
		precision, scale, ok := t.DecimalSize()
		if !ok {
			return types.Type{}, errors.New("cannot get decimal size")
		}
		if precision > types.MaxDecimalPrecision || scale > types.MaxDecimalScale {
			return types.Type{}, fmt.Errorf("Snowflake type %s(%d,%d) is not supported",
				t.DatabaseTypeName(), precision, scale)
		}
		return types.Decimal(int(precision), int(scale)), nil
	case "OBJECT":
		return types.Map(types.JSON()), nil
	case "REAL":
		return types.Float(64), nil
	case "TEXT":
		length, ok := t.Length()
		if !ok {
			return types.Type{}, errors.New("cannot get length")
		}
		if length < 0 {
			return types.Type{}, errors.New("invalid TEXT length")
		}
		const maxBytesLen = 16_777_216
		return types.Text().WithByteLen(maxBytesLen).WithCharLen(int(length)), nil
	case "TIME":
		return types.Time(), nil
	case "TIMESTAMP_NTZ":
		return types.DateTime(), nil
	case "VARIANT":
		return types.JSON(), nil
	}
	return types.Type{}, chichi.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
}

// validateValues validates the user-entered values and returns the settings.
func validateValues(ctx context.Context, values []byte) ([]byte, error) {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate Account.
	if n := utf8.RuneCountInString(s.Account); n < 1 || n > 255 {
		return nil, chichi.NewInvalidUIValuesError("account length must be in range [1,255]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 255 {
		return nil, chichi.NewInvalidUIValuesError("username length must be in range [1,255]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 255 {
		return nil, chichi.NewInvalidUIValuesError("password length must be in range [1,255]")
	}
	// Validate Warehouse.
	if n := utf8.RuneCountInString(s.Warehouse); n < 1 || n > 255 {
		return nil, chichi.NewInvalidUIValuesError("warehouse length must be in range [1,255]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 255 {
		return nil, chichi.NewInvalidUIValuesError("database length must be in range [1,255]")
	}
	// Validate Schema.
	if n := utf8.RuneCountInString(s.Schema); n < 1 || n > 255 {
		return nil, chichi.NewInvalidUIValuesError("schema length must be in range [1,255]")
	}
	// Validate Role.
	if n := utf8.RuneCountInString(s.Role); n < 1 || n > 255 {
		return nil, chichi.NewInvalidUIValuesError("role length must be in range [1,255]")
	}
	err = testConnection(ctx, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}
