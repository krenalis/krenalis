//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"chichi/apis/postgres"
	"chichi/server"
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
	cancel                 func()
	t                      *testing.T
	done                   chan struct{}
	transformationsTempDir string
	httpClient             *http.Client
	workspace              int
}

var chichiAlreadyLaunched bool
var chichiAlreadyBuilt bool

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

	c := Chichi{
		t:         t,
		done:      make(chan struct{}),
		workspace: 1,
	}

	// Create the HTTP client.
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cannot create a cookie jar: %s", err)
	}
	c.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("redirect")
		},
		Jar: jar,
	}

	// In case of an error during the test initialization, stop Chichi.
	var initOk bool
	defer func() {
		if !initOk {
			c.Stop()
		}
	}()

	// Create a temporary directory that will hold the transformation files.
	transformationsTempDir, err := os.MkdirTemp("", "chichi-tests-python-transformation-*")
	if err != nil {
		t.Fatalf("cannot create temporary directory for Python transformation files: %s", err)
	}
	c.transformationsTempDir = transformationsTempDir

	// Create an admin session key.
	var sessionKey string
	{
		key := make([]byte, 64)
		_, err = rand.Read(key)
		if err != nil {
			t.Fatalf("cannot generate an admin session key: %s", err)
		}
		sessionKey = base64.StdEncoding.EncodeToString(key)
	}

	setts := server.Settings{}
	setts.Main.Host = testsSettings.ChichiHost
	setts.Admin.SessionKey = sessionKey
	setts.PostgreSQL.Host = testsSettings.Database.Host
	setts.PostgreSQL.Port = testsSettings.Database.Port
	setts.PostgreSQL.Username = testsSettings.Database.Username
	setts.PostgreSQL.Password = testsSettings.Database.Password
	setts.PostgreSQL.Database = testsSettings.Database.Database
	setts.PostgreSQL.Schema = testsSettings.Database.Schema
	setts.Transformer.Local.PythonExecutable = testsSettings.PythonExecutable
	setts.Transformer.Local.FunctionsDir = transformationsTempDir

	// Launch Chichi.
	ctxWithCancel, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	if launchChichiExternally {
		// Create, if necessary, the directory that will hold the Chichi
		// executable (as well as the other files, eg. config.yaml, needed for
		// the execution).
		repo, err := filepath.Abs("../")
		if err != nil {
			t.Fatal(err)
		}
		_, err = os.Stat(filepath.Join(repo, "go.work"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Fatal("file 'go.work' not found, cannot determine root directory where to build Chichi")
			}
			t.Fatal(err)
		}
		chichiDir := filepath.Join(repo, "test", "chichi-executable-for-tests")
		err = os.Mkdir(chichiDir, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			t.Fatal(err)
		}
		// Write the config YAML file.
		err = writeConfigYAMLFile(chichiDir, &setts)
		if err != nil {
			t.Fatalf("cannot write the YAML config file: %s", err)
		}
		if !chichiAlreadyBuilt {
			err := buildChichi(repo, chichiDir, ctx, &setts)
			if err != nil {
				t.Fatalf("cannot build Chichi: %s", err)
			}
			chichiAlreadyBuilt = true
		}
		go func() {
			err := launchChichi(ctxWithCancel)
			if err != nil {
				select {
				case <-ctxWithCancel.Done():
				default:
					log.Printf("[error] %s", err)
				}
			}
			close(c.done)
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
			close(c.done)
		}()
	}

	// Wait a second for Chichi to load.
	time.Sleep(1 * time.Second)

	// Wait until Chichi starts listening.
	attempts := 0
	for {
		select {
		case <-c.done:
			t.Fatalf("ChiChi has exited")
		default:
		}
		conn, err := net.DialTimeout("tcp", testsSettings.ChichiHost, 500*time.Millisecond)
		if err != nil {
			attempts++
			if attempts >= 30 {
				t.Fatalf("cannot connect to Chichi on %q. No response after %d connections attempts, aborting test", testsSettings.ChichiHost, attempts)
			}
			// Use an exponential backoff timeout.
			timeout := time.Duration(math.Exp(float64(attempts-1))*5) * time.Millisecond
			time.Sleep(timeout)
			continue
		}
		_ = conn.Close()
		break
	}

	// Login.
	err = c.login()
	if err != nil {
		t.Fatalf("cannot log in into the API: %s", err)
	}

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

	// Add columns to the tables in the data warehouse.
	err = addColumnsToWarehouseTables(ctx, testsSettings.Warehouse)
	if err != nil {
		t.Fatalf("cannot add columns to data warehouse: %s", err)
	}

	// Wait some time for the leader election.
	time.Sleep(3 * time.Second)

	initOk = true

	return &c
}

// CountEventsInWarehouse returns the counts of events stored in the "events"
// table of the testing data warehouse.
func (c *Chichi) CountEventsInWarehouse(ctx context.Context) int {
	db, err := postgres.Open(&postgres.Options{
		Host:     testsSettings.Warehouse.Host,
		Port:     testsSettings.Warehouse.Port,
		Username: testsSettings.Warehouse.Username,
		Password: testsSettings.Warehouse.Password,
		Database: testsSettings.Warehouse.Database,
		Schema:   testsSettings.Warehouse.Schema,
	})
	if err != nil {
		c.t.Fatalf("cannot open warehouse for executing query tests: %s", err)
	}
	defer db.Close()
	var count int
	err = db.QueryRow(ctx, `SELECT COUNT(*) AS "count" FROM "events"`).Scan(&count)
	if err != nil {
		c.t.Fatalf("cannot execute count query: %s", err)
	}
	return count
}

// Stop stops the execution of Chichi.
func (c *Chichi) Stop() {
	c.cancel()
	<-c.done
	if c.transformationsTempDir == "" {
		panic("BUG: transformationsTempDir not set")
	}
	err := os.RemoveAll(c.transformationsTempDir)
	if err != nil {
		log.Printf("cannot remove transformations temporary directory: %s", err)
		return
	}
}

// UseWorkspace uses the given workspace in the next calls perfomed using the
// support methods exposed by Chichi.
// The default workspace, used when UseWorkspace is never called, is 1.
func (c *Chichi) UseWorkspace(workspace int) {
	c.workspace = workspace
}

func (c *Chichi) connectWarehouse(whType string, whSettings *DBSettings) error {
	body := map[string]any{
		"Type":     whType,
		"Settings": whSettings,
	}
	_, err := c.call("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connect-warehouse", body)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chichi) initWarehouse() error {
	_, err := c.call("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/init-warehouse", nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chichi) login() error {
	body := map[string]any{
		"Email":    "acme@open2b.com",
		"Password": "foopass2",
	}
	_, err := c.call("POST", "/api/members/login", body)
	return err
}

func resetDatabase(ctx context.Context, dbSetts *DBSettings) error {
	err := validDatabaseNameForTests(dbSetts.Database)
	if err != nil {
		return err
	}
	err = recreateDatabase(ctx, dbSetts.Host, dbSetts.Port, dbSetts.Username, dbSetts.Password, dbSetts.Database)
	if err != nil {
		return fmt.Errorf("cannot recreate database: %s", err)
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

// ExecQueryTestDatabase executes a query on the test database.
func (c *Chichi) ExecQueryTestDatabase(ctx context.Context, query string, args ...any) {
	db, err := postgres.Open(&postgres.Options{
		Host:     testsSettings.Database.Host,
		Port:     testsSettings.Database.Port,
		Username: testsSettings.Database.Username,
		Password: testsSettings.Database.Password,
		Database: testsSettings.Database.Database,
		Schema:   testsSettings.Database.Schema,
	})
	if err != nil {
		c.t.Fatalf("cannot open database for executing query tests: %s", err)
	}
	_, err = db.Exec(ctx, query, args...)
	if err != nil {
		c.t.Fatalf("query %q failed: %s", query, err)
	}
	db.Close()
}

// QueryRowTestDatabase queries a row on the test database, scanning it into
// dest, which must be a pointer.
func (c *Chichi) QueryRowTestDatabase(ctx context.Context, dest any, query string, args ...any) {
	if reflect.TypeOf(dest).Kind() != reflect.Pointer {
		panic("dest must be a pointer")
	}
	db, err := postgres.Open(&postgres.Options{
		Host:     testsSettings.Database.Host,
		Port:     testsSettings.Database.Port,
		Username: testsSettings.Database.Username,
		Password: testsSettings.Database.Password,
		Database: testsSettings.Database.Database,
		Schema:   testsSettings.Database.Schema,
	})
	if err != nil {
		c.t.Fatalf("cannot open database for executing query tests: %s", err)
	}
	err = db.QueryRow(ctx, query, args...).Scan(dest)
	if err != nil {
		c.t.Fatalf("cannot scan result of QueryRow: %s", err)
	}
	db.Close()
}

func addColumnsToWarehouseTables(ctx context.Context, warehouse *DBSettings) error {
	err := validDatabaseNameForTests(warehouse.Database)
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

func resetWarehouse(ctx context.Context, warehouse *DBSettings) error {
	err := validDatabaseNameForTests(warehouse.Database)
	if err != nil {
		return err
	}
	err = recreateDatabase(ctx, warehouse.Host, warehouse.Port, warehouse.Username, warehouse.Password, warehouse.Database)
	if err != nil {
		return fmt.Errorf("cannot recreate database: %s", err)
	}
	return nil
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
