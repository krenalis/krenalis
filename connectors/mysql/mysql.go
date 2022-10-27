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

	"chichi/connectors"

	"github.com/go-sql-driver/mysql"
)

// This package is the MySQL connector.
// (https://dev.mysql.com/doc/refman/8.0/en/)

// Make sure it implements the DatabaseConnection interface.
var _ connectors.DatabaseConnection = &connection{}

func init() {
	connectors.RegisterDatabaseConnector("MySQL", New)
}

// New returns a new MySQL connection.
func New(ctx context.Context, settings []byte, fh connectors.Firehose) (connectors.DatabaseConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of MySQL connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	settings *settings
}

// Query executes the given query and returns the resulting rows.
func (c *connection) Query(query string) ([]connectors.Column, connectors.Rows, error) {
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
	columns := make([]connectors.Column, len(columnTypes))
	for i, c := range columnTypes {
		columns[i] = connectors.Column{
			Name: c.Name(),
			Type: c.DatabaseTypeName(),
		}
	}
	return columns, rows, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, form []byte) (*connectors.SettingsUI, error) {
	return nil, nil
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
