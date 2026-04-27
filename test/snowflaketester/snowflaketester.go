// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

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

func CreateTestDatabase() (*SnowflakeTester, error) {

	// Read the Snowflake settings from the environment.
	settings := Settings{
		Account:   os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ACCOUNT"),
		User:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_USER"),
		Password:  os.Getenv("KRENALIS_SNOWFLAKE_TESTER_PASSWORD"),
		Database:  "", // will be set later.
		Role:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ROLE"),
		Schema:    os.Getenv("KRENALIS_SNOWFLAKE_TESTER_SCHEMA"),
		Warehouse: os.Getenv("KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE"),
	}

	// Instantiate a Snowflake connector.
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:          settings.Account,
		User:             settings.User,
		Password:         settings.Password,
		Role:             settings.Role,
		Schema:           settings.Schema,
		Warehouse:        settings.Warehouse,
		DisableTelemetry: true,
	})

	// Generate the name and create the test database.
	db := sql.OpenDB(connector)
	dbName, err := generateTestDatabaseName()
	if err != nil {
		return nil, fmt.Errorf("cannot generate test database name: %s", err)
	}
	_, err = db.Exec("CREATE DATABASE \"%s\"", dbName)
	if err != nil {
		return nil, fmt.Errorf("cannot create test database on Snowflake: %s", err)
	}
	slog.Info("test Snowflake database created", "dbName", dbName)

	settings.Database = dbName
	return &SnowflakeTester{
		connector: connector,
		db:        db,
		settings:  settings,
	}, nil
}

func (st *SnowflakeTester) Settings() Settings {
	return st.settings
}

// JSONSettings returns the settings as JSON, in the form:
//
//	{
//	    "username": "...",
//	    "password": "...",
//	    "account": "...",
//	    "warehouse": "...",
//	    "database": "...",
//	    "schema": "...",
//	    "role": "..."
//	}
func (st *SnowflakeTester) JSONSettings() []byte {
	settings, err := json.Marshal(map[string]any{
		"username":  st.settings.User,
		"password":  st.settings.Password,
		"account":   st.settings.Account,
		"warehouse": st.settings.Warehouse,
		"database":  st.settings.Database,
		"schema":    st.settings.Schema,
		"role":      st.settings.Role,
	})
	if err != nil {
		panic(err)
	}
	return settings
}

func (st *SnowflakeTester) Teardown() error {
	_, err := st.db.Exec("DROP DATABASE \"%s\"", st.settings.Database)
	if err != nil {
		return fmt.Errorf("cannot drop test Snowflake database %q: %s", st.settings.Database, err)
	}
	slog.Info("test Snowflake database dropped", "dbName", st.settings.Database)
	return nil
}

// generateTestDatabaseName generates the name of a Snowflake database to use
// for testing.
// The returned name has the form:
//
//	KRENALIS_TEST_1777297109_e1ddc97e0b7b9d71005affc2325c10b3
//
// and it is not quoted by this function.
func generateTestDatabaseName() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("KRENALIS_TEST_%d_%s",
		time.Now().UTC().Unix(),
		hex.EncodeToString(b),
	), nil
}
