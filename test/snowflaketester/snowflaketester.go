// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"database/sql"
	"database/sql/driver"
	"os"

	"github.com/snowflakedb/gosnowflake"
)

type SnowflakeTester struct {
	connector driver.Connector
	db        *sql.DB
	settings  Settings
}

type Settings struct {
	Account   string
	User      string
	Password  string
	Database  string
	Role      string
	Schema    string
	Warehouse string
}

func New() (*SnowflakeTester, error) {

	settings := Settings{
		Account:   os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ACCOUNT"),
		User:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_USER"),
		Password:  os.Getenv("KRENALIS_SNOWFLAKE_TESTER_PASSWORD"),
		Database:  "", // will be set later.
		Role:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ROLE"),
		Schema:    os.Getenv("KRENALIS_SNOWFLAKE_TESTER_SCHEMA"),
		Warehouse: os.Getenv("KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE"),
	}

	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:          settings.Account,
		User:             settings.User,
		Password:         settings.Password,
		Role:             settings.Role,
		Schema:           settings.Schema,
		Warehouse:        settings.Warehouse,
		DisableTelemetry: true,
	})

	db := sql.OpenDB(connector)

	// TODO

	return &SnowflakeTester{
		connector: connector,
		db:        db,
		settings:  settings,
	}, nil
}

func (st *SnowflakeTester) Settings() Settings {
	return st.settings
}

func (st *SnowflakeTester) DB() *sql.DB { // TODO: is this necessary?
	return st.db
}

func (st *SnowflakeTester) Teardown() error {

	// TODO

	return nil
}
