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
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"

	"github.com/snowflakedb/gosnowflake"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Database and UIHandler interfaces.
var _ interface {
	meergo.Database
	meergo.UIHandler
} = (*Snowflake)(nil)

func init() {
	meergo.RegisterDatabase(meergo.DatabaseInfo{
		Name:        "Snowflake",
		SampleQuery: "SELECT *\nFROM users\nWHERE ${last_change_time}\nLIMIT ${limit}\n",
		Icon:        icon,
	}, New)
}

// New returns a new Snowflake connector instance.
func New(conf *meergo.DatabaseConfig) (*Snowflake, error) {
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
	conf     *meergo.DatabaseConfig
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

// LastChangeTimeCondition returns the query condition used for the
// last_change_time placeholder in the form "column >= value" or, if column is
// empty, a true value.
func (sf *Snowflake) LastChangeTimeCondition(column string, typ types.Type, value any) string {
	if column == "" {
		return "TRUE"
	}
	b := strings.Builder{}
	b.WriteString(quoteColumn(column))
	b.WriteString(` >= `)
	quoteValue(&b, value, typ)
	return b.String()
}

// Query executes the given query and returns the resulting rows and columns.
func (sf *Snowflake) Query(ctx context.Context, query string) (meergo.Rows, []types.Property, error) {
	return sf.query(ctx, query)
}

// ServeUI serves the connector's user interface.
func (sf *Snowflake) ServeUI(ctx context.Context, event string, values []byte, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s Settings
		if sf.settings != nil {
			s = *sf.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, sf.saveValues(ctx, values, false)
	case "test":
		return nil, sf.saveValues(ctx, values, true)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "Account", Label: "Account", Placeholder: "ABCDEFG-TUVWXYZ", Type: "text", MinLength: 1, MaxLength: 255},
			&meergo.Input{Name: "Username", Label: "Username", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&meergo.Input{Name: "Password", Label: "Password", Placeholder: "", Type: "password", MinLength: 1, MaxLength: 255},
			&meergo.Input{Name: "Database", Label: "Database", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&meergo.Input{Name: "Schema", Label: "Schema", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&meergo.Input{Name: "Warehouse", Label: "Warehouse", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
			&meergo.Input{Name: "Role", Label: "Role", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Values: values,
		Buttons: []meergo.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

// Upsert inserts or updates the rows provided in the specified table.
func (sf *Snowflake) Upsert(ctx context.Context, table meergo.Table, rows []map[string]any) error {
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
func (sf *Snowflake) query(ctx context.Context, query string) (meergo.Rows, []types.Property, error) {
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

// saveValues saves the user-entered values as settings. If test is true, it
// validates only the values without saving it.
func (sf *Snowflake) saveValues(ctx context.Context, values []byte, test bool) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate Account.
	if n := utf8.RuneCountInString(s.Account); n < 1 || n > 255 {
		return meergo.NewInvalidUIValuesError("account length must be in range [1,255]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 255 {
		return meergo.NewInvalidUIValuesError("username length must be in range [1,255]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 255 {
		return meergo.NewInvalidUIValuesError("password length must be in range [1,255]")
	}
	// Validate Warehouse.
	if n := utf8.RuneCountInString(s.Warehouse); n < 1 || n > 255 {
		return meergo.NewInvalidUIValuesError("warehouse length must be in range [1,255]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 255 {
		return meergo.NewInvalidUIValuesError("database length must be in range [1,255]")
	}
	// Validate Schema.
	if n := utf8.RuneCountInString(s.Schema); n < 1 || n > 255 {
		return meergo.NewInvalidUIValuesError("schema length must be in range [1,255]")
	}
	// Validate Role.
	if n := utf8.RuneCountInString(s.Role); n < 1 || n > 255 {
		return meergo.NewInvalidUIValuesError("role length must be in range [1,255]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = sf.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	sf.settings = &s
	return nil
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
	return types.Type{}, meergo.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
}
