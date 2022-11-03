//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"chichi/apis"
	"chichi/apis/types"
	"chichi/connector"

	"github.com/go-sql-driver/mysql"
)

// This package is the MySQL connector.
// (https://dev.mysql.com/doc/refman/8.0/en/)

// Make sure it implements the DatabaseConnection interface.
var _ connector.DatabaseConnection = &connection{}

func init() {
	apis.RegisterDatabaseConnector("MySQL", New)
}

// New returns a new MySQL connection.
func New(ctx context.Context, settings []byte, fh connector.Firehose) (connector.DatabaseConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of MySQL connection")
		}
	}
	c.firehose = fh
	return &c, nil
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

// Query executes the given query and returns the resulting rows.
func (c *connection) Query(query string) ([]connector.Column, connector.Rows, error) {
	db, err := sql.Open("mysql", c.settings.dsn())
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxIdleConns(0)
	rows, err := db.QueryContext(c.ctx, query)
	if err != nil {
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		return nil, nil, err
	}
	columns := make([]connector.Column, len(columnTypes))
	for i, c := range columnTypes {
		typ, err := propertyType(c)
		if err != nil {
			_ = rows.Close()
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
func (c *connection) ServeUI(event string, form []byte) (*connector.SettingsUI, error) {

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if c.settings != nil {
			s = *c.settings
		}
	case "test", "save":
		// Test the connection and save the settings if required.
		err := json.Unmarshal(form, &s)
		if err != nil {
			return nil, err
		}
		err = testConnection(c.ctx, &s)
		if err != nil {
			return nil, connector.UIErrorf("connection failed: %s", err)
		}
		if event == "test" {
			return nil, nil
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	}

	ui := &connector.SettingsUI{
		Components: []connector.Component{
			&connector.Input{Name: "host", Value: s.Host, Label: "Host", Placeholder: "DB host", Type: "text"},
			&connector.Input{Name: "username", Value: s.Username, Label: "Username", Placeholder: "DB username", Type: "text"},
			&connector.Input{Name: "password", Value: s.Password, Label: "Password", Placeholder: "DB password", Type: "password"},
			&connector.Input{Name: "port", Value: s.Port, Label: "Port", Placeholder: "DB port", Type: "number", MaxLength: 5},
			&connector.Input{Name: "database", Value: s.Database, Label: "Database name", Placeholder: "DB name", Type: "text"},
		},
		Actions: []connector.Action{
			{Event: "test", Text: "Test Connection", Variant: "primary"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return ui, nil
}

type settings struct {
	Host     string
	Username string
	Password string
	Port     int
	Database string
}

func (s *settings) dsn() string {
	c := mysql.NewConfig()
	c.User = s.Username
	c.Passwd = s.Password
	c.DBName = s.Database
	c.AllowOldPasswords = true
	c.ParseTime = true
	return c.FormatDSN()
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *settings) error {
	db, err := sql.Open("mysql", settings.dsn())
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
	case "BIT":
		return types.Boolean(), nil
	case "TEXT", "BLOB":
		return types.Text(types.Chars(65535)), nil
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
		return types.Double(), nil
	case "ENUM", "SET":
		return types.Text(), nil
	case "FLOAT":
		return types.Real(), nil
	case "GEOMETRY":
		return types.Type{}, errors.New("MySQL geometry type is not supported")
	case "UNSIGNED MEDIUMINT":
		return types.UnsignedMediumInt(), nil
	case "MEDIUMINT":
		return types.MediumInt(), nil
	case "JSON":
		return types.JSON(), nil
	case "UNSIGNED INT":
		return types.UnsignedInt(), nil
	case "INT":
		return types.Int(), nil
	case "LONGTEXT", "LONGBLOB":
		return types.Text(types.Chars(4294967295)), nil
	case "UNSIGNED BIGINT":
		return types.UnsignedBigInt(), nil
	case "BIGINT":
		return types.BigInt(), nil
	case "MEDIUMTEXT", "MEDIUMBLOB":
		return types.Text(types.Chars(16777216)), nil
	case "UNSIGNED SMALLINT":
		return types.UnsignedSmallInt(), nil
	case "SMALLINT":
		return types.SmallInt(), nil
	case "VARCHAR", "CHAR", "VARBINARY", "BINARY":
		length, ok := t.Length()
		if !ok {
			return types.Type{}, errors.New("cannot get length")
		}
		return types.Text(types.Chars(length)), nil
	case "TIME":
		return types.Time(), nil
	case "TIMESTAMP":
		return types.DateTime(), nil
	case "UNSIGNED TINYINT":
		return types.UnsignedTinyInt(), nil
	case "TINYINT":
		return types.TinyInt(), nil
	case "TINYTEXT", "TINYBLOB":
		return types.Text(types.Chars(255)), nil
	case "YEAR":
		return types.Year(), nil
	}
	return types.Type{}, fmt.Errorf("unknown MySQL type: %s", t.DatabaseTypeName())
}
