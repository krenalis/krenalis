// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// snowflaketester provides an interface for creating temporary environments on
// Snowflake, which can be used for testing.
//
// Specifically, this package creates temporary schemas within the provided
// Snowflake database, which are then deleted at the end of the tests.
//
// # Environment variables
//
// The credentials for accessing Snowflake are read from these environment
// variables:
//
//	KRENALIS_SNOWFLAKE_TESTER_ACCOUNT
//	KRENALIS_SNOWFLAKE_TESTER_DATABASE
//	KRENALIS_SNOWFLAKE_TESTER_PASSWORD
//	KRENALIS_SNOWFLAKE_TESTER_ROLE
//	KRENALIS_SNOWFLAKE_TESTER_USER
//	KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE
//
// # Creating a test environment on Snowflake
//
// See the function [CreateTestEnvironment].
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
	"strings"
	"time"

	"github.com/snowflakedb/gosnowflake"
)

// CreateTestEnvironment creates a test environment on Snowflake.
//
// Once created, you need to call the [TestEnvironment.Teardown] method to
// delete it.
//
// The configuration for Snowflake access is read from these environment
// variables:
//
//	KRENALIS_SNOWFLAKE_TESTER_ACCOUNT
//	KRENALIS_SNOWFLAKE_TESTER_DATABASE
//	KRENALIS_SNOWFLAKE_TESTER_PASSWORD
//	KRENALIS_SNOWFLAKE_TESTER_ROLE
//	KRENALIS_SNOWFLAKE_TESTER_USER
//	KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE
func CreateTestEnvironment() (*TestEnvironment, error) {

	// Return a clear error if env vars are not passed.
	found := false
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, "KRENALIS_SNOWFLAKE_TESTER_") {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("creating a Snowflake test environment requires passing environment variables with your Snowflake credentials")
	}

	// Read the Snowflake settings from the environment.
	settings := Settings{
		Account:   os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ACCOUNT"),
		Database:  os.Getenv("KRENALIS_SNOWFLAKE_TESTER_DATABASE"),
		Password:  os.Getenv("KRENALIS_SNOWFLAKE_TESTER_PASSWORD"),
		Role:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ROLE"),
		Schema:    "", // will be set later.
		User:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_USER"),
		Warehouse: os.Getenv("KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE"),
	}

	// Instantiate a Snowflake connector.
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, gosnowflake.Config{
		Account:          settings.Account,
		Database:         settings.Database,
		Password:         settings.Password,
		Role:             settings.Role,
		User:             settings.User,
		Warehouse:        settings.Warehouse,
		DisableTelemetry: true,
	})

	// Generate a random schema name and create it in the given database.
	db := sql.OpenDB(connector)
	schema, err := generateTestSchemaName()
	if err != nil {
		return nil, fmt.Errorf("cannot generate test schema name: %s", err)
	}
	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema))
	if err != nil {
		return nil, fmt.Errorf("CREATE SCHEMA query failed: %s", err)
	}
	slog.Info("Snowflake test schema created", "schema", schema)

	settings.Schema = schema
	return &TestEnvironment{
		connector: connector,
		db:        db,
		settings:  settings,
	}, nil
}

// TestEnvironment represents an instance of a test environment on Snowflake.
type TestEnvironment struct {
	connector driver.Connector
	db        *sql.DB
	settings  Settings
}

// Settings represents the settings for accessing a test environment on Snowflake.
type Settings struct {
	Account   string
	User      string
	Password  string
	Database  string
	Role      string
	Schema    string // something like: "KRENALIS_TEST_SCHEMA_1777459231_ef9618291974b866473c6abe66acc29c"
	Warehouse string
}

// Settings returns the settings of the test environment.
func (testEnv *TestEnvironment) Settings() Settings {
	return testEnv.settings
}

// JSON returns the settings as JSON, in the form:
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
func (settings Settings) JSON() []byte {
	encoded, err := json.Marshal(map[string]any{
		"username":  settings.User,
		"password":  settings.Password,
		"account":   settings.Account,
		"warehouse": settings.Warehouse,
		"database":  settings.Database,
		"schema":    settings.Schema,
		"role":      settings.Role,
	})
	if err != nil {
		panic(err)
	}
	return encoded
}

// Teardown deletes the Snowflake test environment. This method must be called
// for any environment initialized with [CreateTestEnvironment]. Once this
// method is called, the test environment can no longer be used.
func (testEnv *TestEnvironment) Teardown() error {
	_, err := testEnv.db.Exec(fmt.Sprintf("DROP SCHEMA %s", testEnv.settings.Schema))
	if err != nil {
		return fmt.Errorf("cannot drop Snowflake test schema %q: %s", testEnv.settings.Database, err)
	}
	slog.Info("Snowflake test schema dropped", "schema", testEnv.settings.Schema)
	err = testEnv.db.Close()
	if err != nil {
		return fmt.Errorf("cannot close Snowflake db: %s", err)
	}
	return nil
}

// generateTestSchemaName generates the name of a Snowflake schema to use for
// testing.
// The returned name has the form:
//
//	KRENALIS_TEST_SCHEMA_1777459231_ef9618291974b866473c6abe66acc29c
//
// and it is returned unquoted.
func generateTestSchemaName() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("KRENALIS_TEST_SCHEMA_%d_%s",
		time.Now().UTC().Unix(),
		hex.EncodeToString(b),
	), nil
}
