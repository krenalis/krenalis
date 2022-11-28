//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package mysql

// This package is the MySQL connector.
// (https://dev.mysql.com/doc/refman/8.0/en/)

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"

	"github.com/go-sql-driver/mysql"
)

// Connector icon.
var icon []byte

// Make sure it implements the DatabaseConnection interface.
var _ connector.DatabaseConnection = &connection{}

func init() {
	connector.RegisterDatabase("MySQL", New)
}

// New returns a new MySQL connection.
func New(ctx context.Context, conf *connector.DatabaseConfig) (connector.DatabaseConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of MySQL connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "MySQL",
		Type: connector.DatabaseType,
		Icon: icon,
	}
}

// Query executes the given query and returns the resulting rows.
func (c *connection) Query(query string) ([]connector.Column, connector.Rows, error) {
	mysqlConnector, err := mysql.NewConnector(c.settings.config())
	if err != nil {
		return nil, nil, err
	}
	db := sql.OpenDB(mysqlConnector)
	db.SetMaxIdleConns(0)
	rows, err := db.QueryContext(c.ctx, query)
	if err != nil {
		_ = db.Close()
		if err, ok := err.(*mysql.MySQLError); ok {
			return nil, nil, connector.NewDatabaseQueryError(err.Message)
		}
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		_ = db.Close()
		return nil, nil, err
	}
	columns := make([]connector.Column, len(columnTypes))
	for i, c := range columnTypes {
		typ, err := propertyType(c)
		if err != nil {
			_ = rows.Close()
			_ = db.Close()
			return nil, nil, err
		}
		columns[i] = connector.Column{
			Name: c.Name(),
			Type: typ,
		}
	}
	return columns, rows, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, error) {

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if c.settings == nil {
			s.Port = 3306
		} else {
			s = *c.settings
		}
	case "test", "save":
		// Test the connection and save the settings if required.
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
		err = testConnection(c.ctx, &s)
		if err != nil {
			return nil, ui.Errorf("connection failed: %s", err)
		}
		if event == "test" {
			return nil, nil
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	default:
		return nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Value: s.Host, Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Value: s.Port, Label: "Port", Placeholder: "3306", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Value: s.Username, Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 16},
			&ui.Input{Name: "password", Value: s.Password, Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
			&ui.Input{Name: "database", Value: s.Database, Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 64},
		},
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil
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

// propertyType returns the property type of the column with name c and type t.
func propertyType(t *sql.ColumnType) (types.Type, error) {
	switch t.DatabaseTypeName() {
	case "BIT":
		return types.Boolean(), nil
	case "TEXT", "BLOB":
		return types.Text(types.Bytes(65535)), nil
	case "DATE":
		return types.Date("2006-01-02"), nil
	case "DATETIME":
		return types.DateTime(time.RFC3339), nil
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
		return types.JSON(), nil
	case "UNSIGNED INT":
		return types.UInt(), nil
	case "INT":
		return types.Int(), nil
	case "LONGTEXT", "LONGBLOB":
		return types.Text(types.Bytes(4294967295)), nil
	case "UNSIGNED BIGINT":
		return types.UInt64(), nil
	case "BIGINT":
		return types.Int64(), nil
	case "MEDIUMTEXT", "MEDIUMBLOB":
		return types.Text(types.Bytes(16777216)), nil
	case "UNSIGNED SMALLINT":
		return types.UInt16(), nil
	case "SMALLINT":
		return types.Int16(), nil
	case "VARCHAR", "CHAR":
		length, ok := t.Length()
		if !ok {
			return types.Type{}, errors.New("cannot get length")
		}
		return types.Text(types.Chars(length)), nil
	case "VARBINARY", "BINARY":
		length, ok := t.Length()
		if !ok {
			return types.Type{}, errors.New("cannot get length")
		}
		return types.Text(types.Bytes(length)), nil
	case "TIME":
		return types.Text(types.Bytes(10)), nil
	case "TIMESTAMP":
		return types.DateTime(time.RFC3339), nil
	case "UNSIGNED TINYINT":
		return types.UInt8(), nil
	case "TINYINT":
		return types.Int8(), nil
	case "TINYTEXT", "TINYBLOB":
		return types.Text(types.Bytes(255)), nil
	case "YEAR":
		return types.Year(), nil
	}
	return types.Type{}, connector.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
}
