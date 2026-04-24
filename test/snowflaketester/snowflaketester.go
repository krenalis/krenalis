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
	return &SnowflakeTester{
		connector: connector,
		db:        db,
	}, nil
}

func (st *SnowflakeTester) Teardown()
