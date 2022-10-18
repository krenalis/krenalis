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
	"net/http"
	"time"

	"chichi/connectors"

	"github.com/go-sql-driver/mysql"
)

// This package is the MySQL connector.
// (https://dev.mysql.com/doc/refman/8.0/en/)

// Make sure it implements the Connector interface.
var _ connectors.QueryConnecter = &Connector{}

type Connector struct {
	Settings *settings
}

func init() {
	connectors.RegisterConnector("MySQL", (*Connector)(nil))
}

// Query executes the given query and returns the resulting rows.
func (c *Connector) Query(ctx context.Context, query string, timestamp time.Time, limit int) ([]connectors.Column, connectors.Rows, error) {

	err := c.setContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open("mysql", c.Settings.dsn())
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxIdleConns(0)

	var args []any
	if !timestamp.IsZero() {
		args = []any{timestamp}
	} else if limit > 0 {
		args = []any{limit}
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}

	return nil, rows, nil
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
	return json.Unmarshal(settings, &c.Settings)
}

//// rowsCloser embeds *sql.Rows and closes the database when Close is called.
//type rowsCloser struct {
//	*sql.Rows
//	db *sql.DB
//}
//
//// Close closes the rows and the database.
//func (r rowsCloser) Close() error {
//	defer func() {
//		_ = r.db.Close()
//	}()
//	return r.Close()
//}
