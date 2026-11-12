// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
)

// checkReadOnlyQuery describes one expected SQL query and the rows or error
// returned by the fake warehouse database connection.
type checkReadOnlyQuery struct {
	match    string
	suffix   string
	cols     []string
	rows     [][]driver.Value
	maxRows  int
	err      error
	closeErr error
}

// newCheckReadOnlyTestDB returns a database backed by a fake driver that
// validates query order and records every query it receives.
func newCheckReadOnlyTestDB(t *testing.T, responses []checkReadOnlyQuery) (*sql.DB, *[]string) {
	t.Helper()

	queries := make([]string, 0, len(responses))
	connector := &checkReadOnlyConnector{
		t:         t,
		responses: responses,
		queries:   &queries,
	}
	return sql.OpenDB(connector), &queries
}

// checkReadOnlyConnector implements driver.Connector for the fake
// warehouse database.
type checkReadOnlyConnector struct {
	t         *testing.T
	responses []checkReadOnlyQuery
	queries   *[]string
	next      int
}

// Connect returns a new fake connection sharing c's response cursor.
func (c *checkReadOnlyConnector) Connect(context.Context) (driver.Conn, error) {
	return &checkReadOnlyConn{connector: c}, nil
}

// Driver returns the fake driver associated with c.
func (c *checkReadOnlyConnector) Driver() driver.Driver {
	return checkReadOnlyDriver{}
}

// checkReadOnlyDriver implements driver.Driver for sql.OpenDB compatibility.
type checkReadOnlyDriver struct{}

// Open is unused because tests construct databases through sql.OpenDB.
func (checkReadOnlyDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("checkReadOnlyDriver.Open is not used")
}

// checkReadOnlyConn implements the query-capable subset of driver.Conn used by
// warehouse queries.
type checkReadOnlyConn struct {
	connector *checkReadOnlyConnector
}

// Prepare is not implemented because the tests only exercise QueryContext.
func (c *checkReadOnlyConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("checkReadOnlyConn.Prepare is not implemented")
}

// Close closes the fake connection.
func (c *checkReadOnlyConn) Close() error {
	return nil
}

// Begin is not implemented by the query-only test connection.
func (c *checkReadOnlyConn) Begin() (driver.Tx, error) {
	return nil, errors.New("checkReadOnlyConn.Begin is not implemented")
}

// QueryContext validates query against the next expected response and returns
// its configured rows or error.
func (c *checkReadOnlyConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	connector := c.connector
	*connector.queries = append(*connector.queries, query)
	if connector.next >= len(connector.responses) {
		connector.t.Fatalf("unexpected query:\n%s", query)
	}

	response := connector.responses[connector.next]
	connector.next++
	if response.match != "" && !strings.Contains(query, response.match) {
		connector.t.Fatalf("query %d: expected to contain %q, got:\n%s", connector.next, response.match, query)
	}
	if response.suffix != "" && !strings.HasSuffix(query, response.suffix) {
		connector.t.Fatalf("query %d: expected suffix %q, got:\n%s", connector.next, response.suffix, query)
	}
	if response.err != nil {
		return nil, response.err
	}
	rows := response.rows
	if response.maxRows > 0 && len(rows) > response.maxRows {
		rows = rows[:response.maxRows]
	}
	return &checkReadOnlyRows{cols: response.cols, rows: rows, closeErr: response.closeErr}, nil
}

// checkReadOnlyRows implements driver.Rows over an in-memory row slice.
type checkReadOnlyRows struct {
	cols     []string
	rows     [][]driver.Value
	next     int
	closeErr error
}

// Columns returns the configured result column names.
func (r *checkReadOnlyRows) Columns() []string {
	return r.cols
}

// Close closes the fake result set.
func (r *checkReadOnlyRows) Close() error {
	return r.closeErr
}

// Next copies the next configured row into dest.
func (r *checkReadOnlyRows) Next(dest []driver.Value) error {
	if r.next >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.next])
	r.next++
	return nil
}
