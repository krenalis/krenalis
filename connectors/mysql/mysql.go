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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"

	"github.com/go-sql-driver/mysql"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI interface.
var _ connector.UI = (*connection)(nil)

func init() {
	connector.RegisterDatabase(connector.Database{
		Name:                   "MySQL",
		SourceDescription:      "import users and groups from a MySQL database",
		DestinationDescription: "export users and groups to a MySQL database",
		SampleQuery:            "SELECT * FROM users LIMIT ${limit}",
		Icon:                   icon,
	}, new)
}

// new returns a new MySQL connection.
func new(conf *connector.DatabaseConfig) (*connection, error) {
	c := connection{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of MySQL connection")
		}
	}
	return &c, nil
}

type connection struct {
	conf     *connector.DatabaseConfig
	settings *settings
	db       *sql.DB
}

// Close closes the database. When Close is called, no other calls to connection
// methods are in progress and no more will be made.
func (c *connection) Close() error {
	if c.db == nil {
		return nil
	}
	return c.db.Close()
}

// Columns returns the columns of the given table.
func (c *connection) Columns(ctx context.Context, table string) ([]types.Property, error) {
	var err error
	table, err = quoteTable(table)
	if err != nil {
		return nil, err
	}
	rows, columns, err := c.query(ctx, "SELECT * FROM "+table+" LIMIT 0")
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
func (c *connection) Query(ctx context.Context, query string) (connector.Rows, []types.Property, error) {
	return c.query(ctx, query)
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the UI.
		var s settings
		if c.settings == nil {
			s.Port = 3306
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		s, err := c.ValidateSettings(ctx, values)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, nil, nil
		}
		err = c.conf.SetSettings(ctx, s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Label: "Port", Placeholder: "3306", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 16},
			&ui.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
			&ui.Input{Name: "database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 64},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral", Confirm: true},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// Upsert creates or updates the provided rows in the specified table.
// The columns parameter specifies the columns of the rows, including a column
// named "id" that serves as the table's key.
func (c *connection) Upsert(ctx context.Context, table string, rows [][]any, columns []types.Property) error {

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
		b.WriteByte('`')
		b.WriteString(column.Name)
		b.WriteByte('`')
	}
	b.WriteString(") VALUES ")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("(")
		for j, v := range row {
			if j > 0 {
				b.WriteByte(',')
			}
			pt := columns[j].Type.PhysicalType()
			quoteValue(&b, v, pt)
		}
		b.WriteByte(')')
	}
	b.WriteString(` ON DUPLICATE KEY UPDATE `)
	for i, column := range columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('`')
		b.WriteString(column.Name)
		b.WriteString("` = VALUES(`")
		b.WriteString(column.Name)
		b.WriteString("`)")
	}
	query := b.String()

	if err = c.openDB(); err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, query)

	return err
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, ui.Errorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, ui.Errorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 16 {
		return nil, ui.Errorf("username length must be in range [1,16]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 200 {
		return nil, ui.Errorf("password length must be in range [1,200]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 64 {
		return nil, ui.Errorf("database length must be in range [1,64]")
	}
	err = testConnection(ctx, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

// openDB opens the database. If the database is already open it does nothing.
func (c *connection) openDB() error {
	if c.db != nil {
		return nil
	}
	mysqlConnector, err := mysql.NewConnector(c.settings.config())
	if err != nil {
		return err
	}
	db := sql.OpenDB(mysqlConnector)
	db.SetMaxIdleConns(0)
	c.db = db
	return nil
}

// query executes the given query and returns the resulting rows and columns.
func (c *connection) query(ctx context.Context, query string) (connector.Rows, []types.Property, error) {
	if err := c.openDB(); err != nil {
		return nil, nil, err
	}
	rows, err := c.db.QueryContext(ctx, query)
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
			return nil, nil, fmt.Errorf("cannot get type for property %q: %s", column.Name(), err)
		}
		// Unlike what happens with PostgreSQL, the MySQL driver is able to
		// determine whether a column returned by the query is nullable or not.
		nullable, ok := column.Nullable()
		columns[i] = types.Property{
			Name:     column.Name(),
			Type:     typ,
			Nullable: nullable || !ok,
		}
	}
	return rows, columns, nil
}

type settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func (s *settings) config() *mysql.Config {
	c := mysql.NewConfig()
	c.User = s.Username
	c.Passwd = s.Password
	c.DBName = s.Database
	c.AllowOldPasswords = true
	c.ParseTime = true
	return c
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *settings) error {
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
		return types.Text().WithByteLen(65535), nil
	case "DATE":
		return types.Date(), nil
	case "DATETIME":
		return types.DateTime(), nil
	case "DECIMAL":
		precision, scale, ok := t.DecimalSize()
		if !ok {
			return types.Type{}, errors.New("cannot get decimal size")
		}
		return types.Decimal(int(precision), int(scale)), nil
	case "DOUBLE":
		return types.Float(), nil
	case "ENUM", "SET":
		return types.Text(), nil
	case "FLOAT":
		return types.Float32(), nil
	case "UNSIGNED MEDIUMINT":
		return types.UInt24(), nil
	case "MEDIUMINT":
		return types.Int24(), nil
	case "JSON":
		// The driver seems to return the json type as VARCHAR instead of JSON.
		return types.JSON(), nil
	case "UNSIGNED INT":
		return types.UInt(), nil
	case "INT":
		return types.Int(), nil
	case "LONGBLOB":
		return types.Text().WithByteLen(math.MaxUint32), nil
	case "LONGTEXT":
		return types.Text().WithCharLen(math.MaxUint32), nil
	case "UNSIGNED BIGINT":
		return types.UInt64(), nil
	case "BIGINT":
		return types.Int64(), nil
	case "MEDIUMTEXT", "MEDIUMBLOB":
		return types.Text().WithByteLen(16777216), nil
	case "UNSIGNED SMALLINT":
		return types.UInt16(), nil
	case "SMALLINT":
		return types.Int16(), nil
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
		return types.Text().WithCharLen(65535), nil
	case "TIME":
		return types.Time(), nil
	case "TIMESTAMP":
		return types.DateTime(), nil
	case "UNSIGNED TINYINT":
		return types.UInt8(), nil
	case "TINYINT":
		return types.Int8(), nil
	case "TINYBLOB":
		return types.Text().WithByteLen(255), nil
	case "TINYTEXT":
		return types.Text().WithCharLen(255), nil
	case "YEAR":
		return types.Year(), nil
	}
	return types.Type{}, connector.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
}
