// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"database/sql"
	"database/sql/driver"
	"os"

	"github.com/krenalis/krenalis/warehouses"
	"github.com/snowflakedb/gosnowflake"
)

type SnowflakeTester struct {
	connector driver.Connector
	db        *sql.DB
}

func New() (*SnowflakeTester, error) {
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:          os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ACCOUNT"),
		User:             os.Getenv("KRENALIS_SNOWFLAKE_TESTER_USER"),
		Password:         os.Getenv("KRENALIS_SNOWFLAKE_TESTER_PASSWORD"),
		Role:             os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ROLE"),
		Schema:           os.Getenv("KRENALIS_SNOWFLAKE_TESTER_SCHEMA"),
		Warehouse:        os.Getenv("KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE"),
		DisableTelemetry: true,
	})

	db := sql.OpenDB(connector)

	// TODO

	return &SnowflakeTester{
		connector: connector,
		db:        db,
	}, nil
}

func (st *SnowflakeTester) WarehouseSettingsLoader() warehouses.SettingsLoader {
	// TODO
	return nil
}

func (st *SnowflakeTester) DB() *sql.DB { // TODO: is this necessary?
	return st.db
}

func (st *SnowflakeTester) Teardown() error {

	// TODO

	return nil
}
