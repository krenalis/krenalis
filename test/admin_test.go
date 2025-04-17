//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/testimages"

	"github.com/testcontainers/testcontainers-go"
	_postgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// showUI controls whether to display the Playwright UI or not, that is, whether
// to pass the "--ui" option to the "playwright test" command.
//
// Setting this constant to true could be particularly useful in the case of
// tests that do not pass, allowing for more selective execution of various
// tests while simultaneously checking the output of the browser window and
// other useful information for debugging.
const showUI = false

func TestAdmin(t *testing.T) {
	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Create and instantiate the PostgreSQL database referenced in actions.
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
		{
			// TODO(Gianluca): this is a workaround for
			// https://github.com/meergo/meergo/issues/1172.
			if runtime.GOOS == "windows" && postgresHost == "localhost" {
				postgresHost = "127.0.0.1"
			}
		}
		dbHost = postgresHost
		postgresPort, err := container.MappedPort(ctx, "5432/tcp")
		if err != nil {
			t.Fatal(err)
		}
		dbPort = postgresPort.Int()
	}

	// Initialize the PostgreSQL database referenced in actions.
	{
		db, err := db.Open(&db.Options{
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
		err = db.Ping(pingCtx)
		if err != nil {
			t.Fatalf("cannot connect to PostgreSQL: %s", err)
		}
		t.Logf("connection to PostgreSQL database %q established", dbDatabase)
		const query = `CREATE TABLE users (
			email VARCHAR(300),
			first_name VARCHAR(300),
			last_name VARCHAR(300)
		);`
		_, err = db.Exec(context.Background(), query)
		if err != nil {
			t.Fatalf("error while executing query on PostgreSQL database: %s", err)
		}
		t.Logf("table 'users' created on PostgreSQL database %q", dbDatabase)
	}

	// Write the "test-config.json" file.
	testConfig := map[string]any{
		"baseURL":     "http://" + c.Host(),
		"workspaceID": c.WorkspaceID(),
		"dbHost":      dbHost,
		"dbPort":      dbPort,
		"dbUsername":  dbUsername,
		"dbPassword":  dbPassword,
		"dbName":      dbDatabase,
		"dbSchema":    dbSchema,
	}
	testConfigJSONPath := filepath.Join("..", "assets", "tests", "test-config.json")
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

	// Prepare and run the admin tests.
	assetsDir := filepath.Join("..", "assets")
	run(t, "npm", []string{"install"}, assetsDir)
	run(t, "npx", []string{"playwright", "install", "chromium"}, assetsDir)
	if showUI {
		run(t, "npx", []string{"playwright", "test", "--ui"}, assetsDir)
		t.Fatal("the tests were launched through the UI, which means that the Go test" +
			" cannot know the result of the individual tests, and therefore this test" +
			" is considered failed as a precaution")
	} else {
		run(t, "npx", []string{"playwright", "test"}, assetsDir)
	}

}

func run(t *testing.T, name string, args []string, directory string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("error while executing %s: %s", name, err)
	}
}
