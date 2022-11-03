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

	"chichi/apis"
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
		return nil, nil, err
	}
	columns := make([]connector.Column, len(columnTypes))
	for i, c := range columnTypes {
		columns[i] = connector.Column{
			Name: c.Name(),
			Type: c.DatabaseTypeName(),
		}
	}
	return columns, rows, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, form []byte) (*connector.SettingsUI, error) {

	settings := settings{}
	if c.settings != nil {
		settings = *c.settings
	}

	Components := []connector.Component{
		&connector.Input{Name: "host", Value: settings.Host, Label: "Host", Placeholder: "DB host", Type: "text"},
		&connector.Input{Name: "username", Value: settings.Username, Label: "Username", Placeholder: "DB username", Type: "text"},
		&connector.Input{Name: "password", Value: settings.Password, Label: "Password", Placeholder: "DB password", Type: "password"},
		&connector.Input{Name: "port", Value: settings.Port, Label: "Port", Placeholder: "DB port", Type: "number", MaxLength: 5},
		&connector.Input{Name: "database", Value: settings.Database, Label: "Database name", Placeholder: "DB name", Type: "text"},
	}

	UI := connector.SettingsUI{
		Components: Components,
		Actions: []connector.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}
	switch event {
	case "load":
		return &UI, nil
	case "save":
		err := c.firehose.SetSettings(form)
		return nil, err
	default:
		return nil, nil
	}
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
