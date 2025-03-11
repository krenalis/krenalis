//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package mysql implements the MySQL connector.
// (https://dev.mysql.com/doc/refman/8.0/en/)
package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/go-sql-driver/mysql"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterDatabase(meergo.DatabaseInfo{
		Name:        "MySQL",
		SampleQuery: "SELECT *\nFROM users\n",
		Icon:        icon,
	}, New)
}

// New returns a new MySQL connector instance.
func New(conf *meergo.DatabaseConfig) (*MySQL, error) {
	c := MySQL{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of MySQL connector")
		}
	}
	return &c, nil
}

type MySQL struct {
	conf     *meergo.DatabaseConfig
	settings *innerSettings
	db       *sql.DB
}

// Close closes the database.
func (my *MySQL) Close() error {
	if my.db == nil {
		return nil
	}
	return my.db.Close()
}

// Columns returns the columns of the given table.
func (my *MySQL) Columns(ctx context.Context, table string) ([]meergo.Column, error) {
	var err error
	table, err = quoteTable(table)
	if err != nil {
		return nil, err
	}
	rows, columns, err := my.query(ctx, "SELECT * FROM "+table+" LIMIT 0", true)
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
func (my *MySQL) Merge(ctx context.Context, table meergo.Table, rows [][]any) error {
	if err := my.openDB(); err != nil {
		return err
	}
	// Acquire a connection.
	conn, err := my.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Merge rows.
	return merge(ctx, conn, table, rows)
}

// Query executes the given query and returns the resulting rows and columns.
func (my *MySQL) Query(ctx context.Context, query string) (meergo.Rows, []meergo.Column, error) {
	return my.query(ctx, query, false)
}

// QuoteTime returns a quoted time value for the specified type or "NULL" if the
// value is nil.
func (my *MySQL) QuoteTime(value any, typ types.Type) string {
	if value == nil {
		return "NULL"
	}
	var b strings.Builder
	_ = quoteValue(&b, value, typ)
	return b.String()
}

// ServeUI serves the connector's user interface.
func (my *MySQL) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if my.settings == nil {
			s.Port = 3306
		} else {
			s = *my.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, my.saveSettings(ctx, settings, false)
	case "test":
		return nil, my.saveSettings(ctx, settings, true)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&meergo.Input{Name: "Port", Label: "Port", Placeholder: "3306", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&meergo.Input{Name: "Username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 16},
			&meergo.Input{Name: "Password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
			&meergo.Input{Name: "Database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 64},
		},
		Settings: settings,
		Buttons: []meergo.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

// openDB opens the database. If the database is already open it does nothing.
func (my *MySQL) openDB() error {
	if my.db != nil {
		return nil
	}
	mysqlConnector, err := mysql.NewConnector(my.settings.config())
	if err != nil {
		return err
	}
	db := sql.OpenDB(mysqlConnector)
	db.SetMaxIdleConns(0)
	my.db = db
	return nil
}

// query executes the given query and returns the resulting rows and columns.
// writable indicates whether the resulting columns should be marked as
// writable.
func (my *MySQL) query(ctx context.Context, query string, writable bool) (meergo.Rows, []meergo.Column, error) {
	if err := my.openDB(); err != nil {
		return nil, nil, err
	}
	rows, err := my.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		return nil, nil, err
	}
	columns := make([]meergo.Column, len(columnTypes))
	for i, column := range columnTypes {
		typ, err := propertyType(column)
		if err != nil {
			_ = rows.Close()
			return nil, nil, fmt.Errorf("cannot get type for column %q: %s", column.Name(), err)
		}
		// Unlike what happens with PostgreSQL, the MySQL driver is able to
		// determine whether a column returned by the query is nullable or not.
		nullable, ok := column.Nullable()
		columns[i] = meergo.Column{
			Name:     column.Name(),
			Type:     typ,
			Nullable: nullable || !ok,
			Writable: writable,
		}
	}
	return rows, columns, nil
}

// saveSettings validates and saves the settings. If test is true, it validates
// only the settings without saving it.
func (my *MySQL) saveSettings(ctx context.Context, settings json.Value, test bool) error {
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
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 16 {
		return meergo.NewInvalidsettingsError("username length must be in range [1,16]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 200 {
		return meergo.NewInvalidsettingsError("password length must be in range [1,200]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 64 {
		return meergo.NewInvalidsettingsError("database length must be in range [1,64]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = my.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	my.settings = &s
	return nil
}

type innerSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func (s *innerSettings) config() *mysql.Config {
	c := mysql.NewConfig()
	c.User = s.Username
	c.Passwd = s.Password
	c.Addr = s.Host + ":" + strconv.Itoa(s.Port)
	c.DBName = s.Database
	c.AllowOldPasswords = true
	c.ParseTime = true
	return c
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *innerSettings) error {
	mysqlConnector, err := mysql.NewConnector(settings.config())
	if err != nil {
		return err
	}
	db := sql.OpenDB(mysqlConnector)
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column with type t.
func propertyType(t *sql.ColumnType) (types.Type, error) {
	switch t.DatabaseTypeName() {
	case "BLOB":
		return types.Text(), nil
	case "DATE":
		return types.Date(), nil
	case "DATETIME":
		return types.DateTime(), nil
	case "DECIMAL":
		precision, scale, ok := t.DecimalSize()
		if !ok {
			return types.Type{}, errors.New("cannot get decimal size")
		}
		if precision < 1 || scale < 0 || scale > precision {
			return types.Type{}, fmt.Errorf("precision and scale (%d,%d) are invalid", precision, scale)
		}
		if precision > types.MaxDecimalPrecision {
			return types.Type{}, fmt.Errorf("precision %d exceeds the maximum supported precision of %d", precision, types.MaxDecimalPrecision)
		}
		if scale > types.MaxDecimalScale {
			return types.Type{}, fmt.Errorf("scale %d exceeds the maximum supported scale of %d", scale, types.MaxDecimalScale)
		}
		return types.Decimal(int(precision), int(scale)), nil
	case "DOUBLE":
		return types.Float(64), nil
	case "ENUM":
		return types.Text(), nil
	// TODO(marco): SET can be implemented as an array(T), but the driver only returns the first element of the set.
	//case "SET":
	//return types.Array(types.Text()), nil
	case "FLOAT":
		return types.Float(32), nil
	case "UNSIGNED MEDIUMINT":
		return types.Uint(24), nil
	case "MEDIUMINT":
		return types.Int(24), nil
	case "JSON":
		return types.JSON(), nil
	case "UNSIGNED INT":
		return types.Uint(32), nil
	case "INT":
		return types.Int(32), nil
	case "UNSIGNED BIGINT":
		return types.Uint(64), nil
	case "BIGINT":
		return types.Int(64), nil
	case "UNSIGNED SMALLINT":
		return types.Uint(16), nil
	case "SMALLINT":
		return types.Int(16), nil
	case "VARCHAR", "CHAR":
		length, ok := t.Length()
		if !ok {
			return types.Text(), nil
		}
		return types.Text().WithCharLen(int(length)), nil
	case "VARBINARY", "BINARY":
		length, ok := t.Length()
		if !ok {
			return types.Text(), nil
		}
		return types.Text().WithByteLen(int(length)), nil
	case "TEXT":
		return types.Text(), nil
	case "TIME":
		return types.Time(), nil
	case "TIMESTAMP":
		return types.DateTime(), nil
	case "UNSIGNED TINYINT":
		return types.Uint(8), nil
	case "TINYINT":
		return types.Int(8), nil
	case "YEAR":
		return types.Year(), nil
	}
	return types.Type{}, meergo.NewUnsupportedColumnTypeError(t.Name(), t.DatabaseTypeName())
}
