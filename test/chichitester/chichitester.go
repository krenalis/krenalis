//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"chichi/apis/postgres"
	"chichi/server"

	"github.com/redis/go-redis/v9"
)

// launchChichiExternally determines if Chichi should be launched externally
// when testing.
//
//   - Set this to true when testing Chichi using 'go run commit/commit.go' or
//     'go test'.
//
//   - Set this to false when debugging a single Chichi test.
const launchChichiExternally = true

// Chichi represents an instance of Chichi which responds to HTTP requests and
// exposes methods to make calls to the APIs.
type Chichi struct {
	cancel func()
	t      *testing.T
	done   chan struct{}
}

var chichiAlreadyLaunched bool

// InitAndLaunch initializes and launches an instance of Chichi in a separate
// goroutine.
// After calling InitAndLaunch, the "Stop" method must be called on the returned
// instance of Chichi to stop the instance and shutdown the server.
func InitAndLaunch(t *testing.T) *Chichi {

	if !launchChichiExternally {
		if chichiAlreadyLaunched {
			msg := "aborting tests: you are executing more than one test, consequently the 'launchChichiExternally'" +
				" constant cannot be false. It can be false only when executing a single test"
			t.Fatal(msg)
		}
		chichiAlreadyLaunched = true
	}

	err := loadTestConfig()
	if err != nil {
		t.Fatalf("cannot load tests configuration: %s", err)
	}

	ctx := context.Background()

	err = resetDatabase(ctx, testsSettings.Database)
	if err != nil {
		t.Fatalf("cannot reset database: %s", err)
	}

	err = resetWarehouse(ctx, testsSettings.Warehouse)
	if err != nil {
		t.Fatalf("cannot reset warehouse: %s", err)
	}

	err = resetRedis(ctx, testsSettings.Redis)
	if err != nil {
		t.Fatalf("cannot reset Redis: %s", err)
	}

	c := Chichi{
		t:    t,
		done: make(chan struct{}, 1),
	}

	setts := server.Settings{}
	setts.Main.Host = testsSettings.ChichiHost
	setts.PostgreSQL.Host = testsSettings.Database.Host
	setts.PostgreSQL.Port = testsSettings.Database.Port
	setts.PostgreSQL.Username = testsSettings.Database.Username
	setts.PostgreSQL.Password = testsSettings.Database.Password
	setts.PostgreSQL.Database = testsSettings.Database.Database
	setts.PostgreSQL.Schema = testsSettings.Database.Schema
	setts.Redis.Network = testsSettings.Redis.Network
	setts.Redis.Addr = testsSettings.Redis.Addr
	setts.Redis.Username = testsSettings.Redis.Username
	setts.Redis.Password = testsSettings.Redis.Password
	setts.Redis.DB = testsSettings.Redis.DB

	// Launch Chichi.
	ctxWithCancel, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	if launchChichiExternally {
		err := buildChichi(ctxWithCancel, &setts)
		if err != nil {
			t.Fatalf("cannot build Chichi: %s", err)
		}
		go func() {
			err := launchChichi(ctxWithCancel)
			if err != nil && !isSignalKilledError(err) {
				log.Printf("[error] %s", err)
			}
			c.done <- struct{}{}
		}()
	} else {
		err = validDatabaseNameForTests(setts.PostgreSQL.Database)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			err := server.Run(ctxWithCancel, &setts)
			if err != nil {
				log.Printf("[error] %s", err)
				return
			}
			c.done <- struct{}{}
		}()
	}

	// Wait some time for Chichi to load.
	time.Sleep(1 * time.Second)

	// Connect the data warehouse.
	err = c.connectWarehouse(testsSettings.WarehouseType, testsSettings.Warehouse)
	if err != nil {
		t.Fatalf("cannot connect warehouse: %s", err)
	}

	// Initialize the data warehouse.
	err = c.initWarehouse()
	if err != nil {
		t.Fatalf("cannot init warehouse: %s", err)
	}

	// Reload the schemas.
	err = c.reloadSchemas()
	if err != nil {
		t.Fatalf("cannot reload schemas: %s", err)
	}

	// Wait some time for the leader election.
	time.Sleep(3 * time.Second)

	return &c
}

// Stop stops the execution of Chichi.
func (c *Chichi) Stop() {
	c.cancel()
	<-c.done
}

func (c *Chichi) connectWarehouse(whType string, whSettings *DBSettings) error {
	body := map[string]any{
		"Type":     whType,
		"Settings": whSettings,
	}
	_, err := c.call("POST", "/api/workspace/connect-warehouse", body)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chichi) initWarehouse() error {
	_, err := c.call("POST", "/api/workspace/init-warehouse", nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chichi) reloadSchemas() error {
	_, err := c.call("POST", "/api/workspace/reload-schemas", nil)
	if err != nil {
		return err
	}
	return nil
}

func resetDatabase(ctx context.Context, dbSetts *DBSettings) error {
	err := recreateDatabase(ctx, dbSetts.Host, dbSetts.Port, dbSetts.Username, dbSetts.Password, dbSetts.Database)
	if err != nil {
		return fmt.Errorf("cannot recreate database: %s", err)
	}
	err = validDatabaseNameForTests(dbSetts.Database)
	if err != nil {
		return err
	}
	db, err := postgres.Open(&postgres.Options{
		Host:     dbSetts.Host,
		Port:     dbSetts.Port,
		Username: dbSetts.Username,
		Password: dbSetts.Password,
		Database: dbSetts.Database,
		Schema:   dbSetts.Schema,
	})
	if err != nil {
		return fmt.Errorf("cannot establish connection to database: %s", err)
	}
	defer db.Close()
	err = execQueries(ctx, db, "../database/PostgreSQL.sql")
	if err != nil {
		return err
	}
	return nil
}

func resetWarehouse(ctx context.Context, warehouse *DBSettings) error {
	err := recreateDatabase(ctx, warehouse.Host, warehouse.Port, warehouse.Username, warehouse.Password, warehouse.Database)
	if err != nil {
		return fmt.Errorf("cannot recreate database: %s", err)
	}
	err = validDatabaseNameForTests(warehouse.Database)
	if err != nil {
		return err
	}
	db, err := postgres.Open(&postgres.Options{
		Host:     warehouse.Host,
		Port:     warehouse.Port,
		Username: warehouse.Username,
		Password: warehouse.Password,
		Database: warehouse.Database,
		Schema:   warehouse.Schema,
	})
	if err != nil {
		return fmt.Errorf("cannot establish connection to warehouse: %s", err)
	}
	defer db.Close()
	err = execQueries(ctx, db, "../database/warehouses/postgresql.sql")
	if err != nil {
		return err
	}
	return nil
}

func resetRedis(ctx context.Context, conf *RedisSettings) error {
	c := redis.NewClient(&redis.Options{
		Network:  conf.Network,
		Addr:     conf.Addr,
		Username: conf.Username,
		Password: conf.Password,
		DB:       conf.DB,
	})
	err := c.FlushAll(ctx).Err()
	return err
}

func recreateDatabase(ctx context.Context, host string, port int, username, password, database string) error {
	db, err := postgres.Open(&postgres.Options{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Database: "postgres",
	})
	if err != nil {
		return fmt.Errorf("cannot connect to database: %s", err)
	}
	defer db.Close()
	err = validDatabaseNameForTests(database)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "DROP DATABASE IF EXISTS "+database)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "CREATE DATABASE "+database)
	if err != nil {
		return err
	}
	return nil
}

// execQueries executes on db the queries read from queriesFile, separated by a
// ";" character and a newline.
func execQueries(ctx context.Context, db *postgres.DB, queriesFile string) error {

	content, err := os.ReadFile(queriesFile)
	if err != nil {
		abs, err2 := filepath.Abs(queriesFile)
		if err2 != nil {
			log.Panic(err)
		}
		return fmt.Errorf("cannot read %q: %s", abs, err)
	}

	queries := strings.Split(string(content), ";\n")

	// Recreate the schema from "PostgreSQL.sql".
	for _, query := range queries {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err := db.Exec(ctx, query)
		if err != nil {
			return err
		}
		cancel()
	}

	return nil

}

var databaseNameRegexp = regexp.MustCompile(`^test_[a-zA-Z0-9_]+$`)

// validDatabaseNameForTests returns an error if name is not a valid database
// name for tests.
func validDatabaseNameForTests(name string) error {
	valid := databaseNameRegexp.MatchString(name)
	if !valid {
		return fmt.Errorf("invalid database name %q, it must match the regexp: ^test_[a-zA-Z0-9_]+$", name)
	}
	return nil
}

func isSignalKilledError(err error) bool {
	ee, ok := err.(*exec.ExitError)
	return ok && ee.Error() == "signal: killed"
}
