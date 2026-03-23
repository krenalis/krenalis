// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/test/testimages"

	"github.com/testcontainers/testcontainers-go"
	_postgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// passUIFlagToPlaywright, when set to true, passes the '--ui' flag to the
// Playwright command, when it is executed for the Admin tests.
//
// To understand what the '--ui' flag does, see the Playwright UI Mode
// documentation: https://playwright.dev/docs/test-ui-mode.
//
// NOTE: this is intended for advanced usage and debugging purposes only, so if
// the Admin test is run with this constant enabled, the Go test will always
// fail, as there is no way to determine whether the test set launched via the
// UI is complete and whether all tests ran correctly and successfully.
const passUIFlagToPlaywright = false

func TestAdmin(t *testing.T) {

	// See https://github.com/krenalis/krenalis/issues/2116.
	if os.Getenv("MEERGO_TEST_SKIP_ADMIN_TESTS") == "true" {
		t.Skip("Admin test skipped as MEERGO_TEST_SKIP_ADMIN_TESTS is set to true")
	}

	fsTempDir := meergotester.NewTempStorage(t)

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(fsTempDir.Root())
	c.Start()
	defer c.Stop()

	// Create and instantiate the PostgreSQL database referenced in pipelines.
	const (
		dbUsername = "db_test_user"
		dbPassword = "tXALDfgwZP3"
		dbDatabase = "db_for_import"
		dbSchema   = "public"
	)
	var dbHost string
	var dbPort int
	{
		ctx := context.Background()
		container, err := _postgres.Run(ctx,
			testimages.PostgreSQL,
			_postgres.WithDatabase(dbDatabase),
			_postgres.WithUsername(dbUsername),
			_postgres.WithPassword(dbPassword),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second)),
		)
		defer func() {
			err := testcontainers.TerminateContainer(container)
			if err != nil {
				t.Errorf("cannot terminate container: %s", err)
			}
		}()
		if err != nil {
			t.Fatalf("cannot start the PostgreSQL container: %s", err)
		}
		postgresHost, err := container.Host(ctx)
		if err != nil {
			t.Fatal(err)
		}
		dbHost = postgresHost
		postgresPort, err := container.MappedPort(ctx, "5432/tcp")
		if err != nil {
			t.Fatal(err)
		}
		dbPort = postgresPort.Int()
	}

	// Initialize the PostgreSQL database referenced in pipelines.
	{
		pool, err := meergotester.ConnectionPool(t.Context(), &meergotester.DBSettings{
			Host:     dbHost,
			Port:     dbPort,
			Username: dbUsername,
			Password: dbPassword,
			Database: dbDatabase,
			Schema:   "public",
		})
		if err != nil {
			t.Fatalf("cannot connect to PostgreSQL: %s", err)
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err = pool.Ping(pingCtx)
		if err != nil {
			t.Fatalf("cannot connect to PostgreSQL: %s", err)
		}
		t.Logf("connection to PostgreSQL database %q established", dbDatabase)
		const query = `CREATE TABLE users (
			email VARCHAR(300),
			first_name VARCHAR(300),
			last_name VARCHAR(300)
		);`
		_, err = pool.Exec(context.Background(), query)
		if err != nil {
			t.Fatalf("error while executing query on PostgreSQL database: %s", err)
		}
		t.Logf("table 'users' created on PostgreSQL database %q", dbDatabase)
	}

	// Write the "test-config.json" file.
	testConfig := map[string]any{
		"baseURL":     "http://" + c.Addr(),
		"workspaceID": c.WorkspaceID(),
		"dbHost":      dbHost,
		"dbPort":      dbPort,
		"dbUsername":  dbUsername,
		"dbPassword":  dbPassword,
		"dbName":      dbDatabase,
		"dbSchema":    dbSchema,
	}
	testConfigJSONPath := filepath.Join("..", "admin", "tests", "test-config.json")
	f, err := os.OpenFile(testConfigJSONPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	err = enc.Encode(testConfig)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Logf("configuration file %q created", testConfigJSONPath)

	// Prepare and run the Admin tests.
	adminDir := filepath.Join("..", "admin")
	run(t, "npm", []string{"install"}, adminDir, fsTempDir.Root())
	run(t, "npx", []string{"playwright", "install", "chromium"}, adminDir, fsTempDir.Root())
	if passUIFlagToPlaywright {
		run(t, "npx", []string{"playwright", "test", "--ui"}, adminDir, fsTempDir.Root())
		t.Fatal("The Admin test was run with the constant 'passUIFlagToPlaywright' set to true," +
			" so the test is considered to have failed as a precaution." +
			" For more details, see the documentation for the constant 'passUIFlagToPlaywright'.")
	} else {
		run(t, "npx", []string{"playwright", "test"}, adminDir, fsTempDir.Root())
	}

	// The tests have been run, so the temporary directory used by File System
	// can be deleted.
	fsTempDir.Remove()
}

func run(t *testing.T, name string, args []string, directory, fsTempDir string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "MEERGO_TEST_FS_TEMP_DIR="+fsTempDir)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("error while executing %s: %s", name, err)
	}
}
