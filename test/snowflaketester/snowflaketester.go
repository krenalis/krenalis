// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// snowflaketester provides an interface for creating temporary databases on
// Snowflake, which can be used for testing.
//
// # Environment variables
//
// The credentials for accessing Snowflake are read from these environment
// variables:
//
//	KRENALIS_SNOWFLAKE_TESTER_ACCOUNT
//	KRENALIS_SNOWFLAKE_TESTER_PASSWORD
//	KRENALIS_SNOWFLAKE_TESTER_ROLE
//	KRENALIS_SNOWFLAKE_TESTER_SCHEMA
//	KRENALIS_SNOWFLAKE_TESTER_USER
//	KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE
//
// # Creating a test database on Snowflake
//
// See the function [CreateTestDatabase].
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

// CreateTestDatabase creates a test database on Snowflake with an unique name.
//
// Once created, you need to call the [TestDB.Teardown] method to
// delete it.
//
// The configuration for Snowflake access is read from these environment
// variables:
//
//	KRENALIS_SNOWFLAKE_TESTER_ACCOUNT
//	KRENALIS_SNOWFLAKE_TESTER_PASSWORD
//	KRENALIS_SNOWFLAKE_TESTER_ROLE
//	KRENALIS_SNOWFLAKE_TESTER_SCHEMA
//	KRENALIS_SNOWFLAKE_TESTER_USER
//	KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE
func CreateTestDatabase() (*TestDB, error) {

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
	return &TestDB{
		connector: connector,
		db:        db,
		settings:  settings,
	}, nil
}

// TestDB represents an instance of a test database on Snowflake.
type TestDB struct {
	connector driver.Connector
	db        *sql.DB
	settings  Settings
}

// Settings represents the settings for accessing a test database on Snowflake.
type Settings struct {
	Account   string
	User      string
	Password  string
	Database  string // something like: "KRENALIS_TEST_1777297109_e1ddc97e0b7b9d71005affc2325c10b3"
	Role      string
	Schema    string
	Warehouse string
}

// Settings returns the settings of the test database.
func (testDB *TestDB) Settings() Settings {
	return testDB.settings
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
func (testDB *TestDB) JSONSettings() []byte {
	settings, err := json.Marshal(map[string]any{
		"username":  testDB.settings.User,
		"password":  testDB.settings.Password,
		"account":   testDB.settings.Account,
		"warehouse": testDB.settings.Warehouse,
		"database":  testDB.settings.Database,
		"schema":    testDB.settings.Schema,
		"role":      testDB.settings.Role,
	})
	if err != nil {
		panic(err)
	}
	return settings
}

// Teardown deletes the Snowflake test database. This method must be called for
// any database initialized with [CreateTestDatabase]. Once this method is
// called, the test database can no longer be used.
func (testDB *TestDB) Teardown() error {
	_, err := testDB.db.Exec("DROP DATABASE \"%s\"", testDB.settings.Database)
	if err != nil {
		return fmt.Errorf("cannot drop test Snowflake database %q: %s", testDB.settings.Database, err)
	}
	slog.Info("test Snowflake database dropped", "dbName", testDB.settings.Database)
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
