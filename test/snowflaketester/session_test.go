// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/snowflakedb/gosnowflake/v2"
)

type sessionTestConnector struct {
	columns                       []string
	values                        [][]driver.Value
	connectErr, queryErr, nextErr error
	rowsCloseErr                  error
	connections                   []*sessionTestConn
}

// Connect creates a connection with the configured session response.
func (c *sessionTestConnector) Connect(context.Context) (driver.Conn, error) {
	if c.connectErr != nil {
		return nil, c.connectErr
	}
	conn := &sessionTestConn{connector: c, rows: &sessionTestRows{columns: c.columns, values: c.values,
		nextErr: c.nextErr, closeErr: c.rowsCloseErr}}
	c.connections = append(c.connections, conn)
	return conn, nil
}

// Driver returns the synthetic driver's DSN entry point.
func (*sessionTestConnector) Driver() driver.Driver {
	return sessionTestDriver{}
}

type sessionTestConn struct {
	queryContext context.Context
	connector    *sessionTestConnector
	rows         *sessionTestRows
	queries      int
	closed       int
}

// Begin rejects unexpected transactions.
func (*sessionTestConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("test/snowflaketester: unexpected transaction")
}

// Close records physical connection cleanup.
func (c *sessionTestConn) Close() error {
	c.closed++
	return nil
}

// Prepare rejects unexpected prepared statements.
func (*sessionTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("test/snowflaketester: unexpected prepared statement")
}

// QueryContext returns the synthetic SHOW PARAMETERS response.
func (c *sessionTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.queries++
	c.queryContext = ctx
	if query != "SHOW PARAMETERS IN SESSION" || len(args) != 0 || ctx == nil {
		return nil, fmt.Errorf("test/snowflaketester: unexpected session query")
	}
	if c.connector.queryErr != nil {
		return nil, c.connector.queryErr
	}
	return c.rows, nil
}

type sessionTestDriver struct{}

// Open rejects DSN connections, since the test uses sql.OpenDB.
func (sessionTestDriver) Open(string) (driver.Conn, error) {
	return nil, fmt.Errorf("test/snowflaketester: unexpected DSN connection")
}

type sessionTestRows struct {
	columns           []string
	values            [][]driver.Value
	nextErr, closeErr error
	index, closed     int
}

// Close records result cleanup and returns the configured failure.
func (r *sessionTestRows) Close() error {
	r.closed++
	return r.closeErr
}

// Columns returns the configured metadata.
func (r *sessionTestRows) Columns() []string {
	return r.columns
}

// Next returns parameter rows followed by EOF or the configured failure.
func (r *sessionTestRows) Next(values []driver.Value) error {
	if r.index == len(r.values) {
		if r.nextErr != nil {
			return r.nextErr
		}
		return io.EOF
	}
	copy(values, r.values[r.index])
	r.index++
	return nil
}

// TestSessionConnector checks failures before a connection can enter the pool.
func TestSessionConnector(t *testing.T) {
	failure := errors.New("synthetic driver failure")
	for _, test := range []struct {
		name       string
		change     func(*sessionTestConnector)
		wantDriver bool
	}{
		{"connect", func(c *sessionTestConnector) { c.connectErr = failure }, true},
		{"query", func(c *sessionTestConnector) { c.queryErr = failure }, true},
		{"iteration", func(c *sessionTestConnector) { c.nextErr = failure }, true},
		{"rows close", func(c *sessionTestConnector) { c.rowsCloseErr = failure }, true},
		{"missing", func(c *sessionTestConnector) { c.values = c.values[:3] }, false},
		{"duplicate", func(c *sessionTestConnector) { c.values = append(c.values, c.values[0]) }, false},
		{"key type", func(c *sessionTestConnector) { c.values[0][0] = nil }, false},
		{"value type", func(c *sessionTestConnector) { c.values[0][1] = int64(60) }, false},
		{"missing column", func(c *sessionTestConnector) { c.columns = []string{"key", "other"} }, false},
		{"duplicate column", func(c *sessionTestConnector) { c.columns = []string{"key", "key"} }, false},
		{"wide result", func(c *sessionTestConnector) { c.columns = make([]string, 17) }, false},
		{"many rows", func(c *sessionTestConnector) {
			for range 1025 {
				c.values = append(c.values, []driver.Value{"OTHER", "ignored"})
			}
		}, false},
		{"timezone", func(c *sessionTestConnector) { c.values[0][1] = "UTC" }, false},
		{"mapping", func(c *sessionTestConnector) { c.values[1][1] = "TIMESTAMP_LTZ" }, false},
		{"input", func(c *sessionTestConnector) { c.values[2][1] = "YYYY" }, false},
		{"timeout", func(c *sessionTestConnector) { c.values[3][1] = "0" }, false},
		{"unsafe value", func(c *sessionTestConnector) { c.values[0][1] = strings.Repeat("sensitive", 1024) }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			underlying := &sessionTestConnector{columns: []string{"key", "value"},
				values: sessionTestValues("America/Los_Angeles")}
			test.change(underlying)
			checked := SessionConnector{Connector: underlying, Timezone: "America/Los_Angeles"}
			conn, err := checked.Connect(t.Context())
			if err != nil {
				if test.wantDriver && !errors.Is(err, failure) || !test.wantDriver && len(err.Error()) > 150 {
					t.Fatalf("expected bounded verification error or driver failure, got %v", err)
				}
				if conn != nil {
					t.Fatalf("expected no connection on failure, got %T", conn)
				}
				for _, opened := range underlying.connections {
					if opened.closed != 1 || opened.queries != 1 {
						t.Fatalf("expected one query and connection close, got %+v", opened)
					}
					if underlying.queryErr == nil && opened.rows.closed != 1 {
						t.Fatalf("expected one result close, got %d", opened.rows.closed)
					}
				}
				return
			}
			t.Fatalf("expected session verification failure, got nil")
		})
	}
}

// TestSessionConnectorInvalid rejects invalid overlay configuration locally.
func TestSessionConnectorInvalid(t *testing.T) {
	for _, checked := range []SessionConnector{{}, {Connector: &sessionTestConnector{}, Timezone: "Europe/Rome"}} {
		conn, err := checked.Connect(t.Context())
		if err != nil {
			if conn != nil {
				t.Fatalf("expected no connection on invalid configuration, got %T", conn)
			}
			continue
		}
		t.Fatalf("expected invalid session connector error, got nil")
	}
}

// TestSessionConnectorPool checks every new connection, including replacements.
func TestSessionConnectorPool(t *testing.T) {
	for _, timezone := range []string{"UTC", "America/Los_Angeles"} {
		t.Run(timezone, func(t *testing.T) {

			underlying := &sessionTestConnector{columns: []string{"key", "value"}, values: sessionTestValues(timezone)}
			underlying.values = append(underlying.values, []driver.Value{"OTHER", "ignored"})
			checked := SessionConnector{Connector: underlying, Timezone: timezone}
			db := sql.OpenDB(checked)
			t.Cleanup(func() {
				err := db.Close()
				if err != nil {
					t.Errorf("expected pool cleanup, got %v", err)
				}
			})
			db.SetMaxOpenConns(3)
			db.SetMaxIdleConns(0)

			var held []*sql.Conn
			for range 3 {
				conn, err := db.Conn(t.Context())
				if err != nil {
					t.Fatalf("expected verified pool connection, got %v", err)
				}
				held = append(held, conn)
			}
			for _, conn := range held {
				err := conn.Close()
				if err != nil {
					t.Fatalf("expected idle connection cleanup, got %v", err)
				}
			}

			replacement, err := db.Conn(t.Context())
			if err != nil {
				t.Fatalf("expected verified replacement connection, got %v", err)
			}
			err = replacement.Raw(func(conn any) error {
				if _, ok := conn.(*sessionTestConn); !ok {
					return fmt.Errorf("expected original driver connection, got %T", conn)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			err = replacement.Close()
			if err != nil {
				t.Fatalf("expected replacement cleanup, got %v", err)
			}

			if len(underlying.connections) != 4 {
				t.Fatalf("expected four physical connections, got %d", len(underlying.connections))
			}
			for _, conn := range underlying.connections {
				if conn.closed != 1 || conn.queries != 1 || conn.rows.closed != 1 {
					t.Fatalf("expected verified and closed physical connection, got %+v", conn)
				}
			}

		})
	}
}

// TestSessionConnectorProfiles checks the timeout and cancellation contracts.
func TestSessionConnectorProfiles(t *testing.T) {
	for _, profile := range []string{"standard", "timeout", "cancellation"} {
		for _, change := range []string{"none", "timeout", "cache", "profile"} {
			t.Run(profile+"/"+change, func(t *testing.T) {
				underlying := &sessionTestConnector{columns: []string{"key", "value"}, values: sessionTestValues("UTC")}
				if profile == "timeout" {
					underlying.values[3][1] = "10"
				}
				underlying.values = append(underlying.values, []driver.Value{"USE_CACHED_RESULT", "false"})
				checked := SessionConnector{Connector: underlying, Timezone: "UTC", Profile: profile}
				switch change {
				case "none":
					// Keep the requested session profile.
				case "timeout":
					underlying.values[3][1] = "0"
				case "cache":
					underlying.values[4][1] = "true"
				case "profile":
					checked.Profile = "unknown"
				default:
					t.Fatalf("expected known mutation, got %s", change)
				}
				valid := change == "none" || profile == "standard" && change == "cache"
				conn, err := checked.Connect(t.Context())
				if err != nil {
					if valid || conn != nil {
						t.Fatalf("expected valid=%t and no failed connection, got %T, %v", valid, conn, err)
					}
					for _, opened := range underlying.connections {
						if opened.closed != 1 || opened.rows.closed != 1 {
							t.Fatal("expected cleanup after rejected profile, got open resources")
						}
					}
					return
				}
				if !valid || conn == nil {
					t.Fatalf("expected rejected profile=%t, got successful connection", !valid)
				}
				err = conn.Close()
				if err != nil {
					t.Fatalf("expected verified connection cleanup, got %v", err)
				}
			})
		}
	}
}

// TestSessionConnectorQueryTag separates verification from the target query.
func TestSessionConnectorQueryTag(t *testing.T) {

	ctx, cancel := context.WithTimeout(gosnowflake.WithQueryTag(t.Context(), "target-query"), time.Minute)
	defer cancel()
	underlying := &sessionTestConnector{columns: []string{"key", "value"}, values: sessionTestValues("UTC")}
	checked := SessionConnector{Connector: underlying, Timezone: "UTC"}
	conn, err := checked.Connect(ctx)
	if err != nil {
		t.Fatalf("expected verified connection, got %v", err)
	}
	err = conn.Close()
	if err != nil {
		t.Fatalf("expected connection cleanup, got %v", err)
	}
	verification := underlying.connections[0].queryContext
	// This is the key read by the vendored driver's request builder.
	key := gosnowflake.ContextKey("QUERY_TAG")
	if verification.Value(key) != "krenalis-session-verification" || ctx.Value(key) != "target-query" {
		t.Fatal("expected distinct verification and target tags, got mixed tags")
	}
	expected, _ := ctx.Deadline()
	deadline, ok := verification.Deadline()
	if !ok || deadline != expected {
		t.Fatalf("expected inherited verification deadline %v, got %v, %t", expected, deadline, ok)
	}
	cancel()
	err = verification.Err()
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected inherited verification cancellation, got %v", err)
		}
		return
	}

	t.Fatal("expected verification cancellation, got nil")
}

func sessionTestValues(timezone string) [][]driver.Value {
	return [][]driver.Value{{"TIMEZONE", timezone}, {"TIMESTAMP_TYPE_MAPPING", "TIMESTAMP_NTZ"},
		{"TIMESTAMP_INPUT_FORMAT", "AUTO"}, {"STATEMENT_TIMEOUT_IN_SECONDS", "60"}}
}
