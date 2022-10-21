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
	"net/http"

	"chichi/connectors"

	"github.com/go-sql-driver/mysql"
)

// This package is the MySQL connector.
// (https://dev.mysql.com/doc/refman/8.0/en/)

// Make sure it implements the Connector interface.
var _ connectors.DatabaseConnecter = &Connector{}

type Connector struct {
	Settings *settings
}

func init() {
	connectors.RegisterConnector("MySQL", (*Connector)(nil))
}

// Query executes the given query and returns the resulting rows.
func (c *Connector) Query(ctx context.Context, query string) ([]connectors.Column, connectors.Rows, error) {
	err := c.setContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	db, err := sql.Open("mysql", c.Settings.dsn())
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxIdleConns(0)
	rows, err := db.QueryContext(ctx, query)
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

// ServeUserInterface serves the connector's user interface.
func (c *Connector) ServeUserInterface(w http.ResponseWriter, r *http.Request) {
	return
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

// setContext sets ctx as the context for c.
func (c *Connector) setContext(ctx context.Context) error {
	settings, _ := ctx.Value(connectors.SettingsContextKey{}).([]byte)
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.Settings)
		if err != nil {
			return errors.New("cannot unmarshal settings of the MySQL connector")
		}
	}
	return nil
}
