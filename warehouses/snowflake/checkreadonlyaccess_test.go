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

	"github.com/krenalis/krenalis/warehouses"
)

func Test_CheckReadOnlyAccess_rejectsNonReadOnlyPrivileges(t *testing.T) {
	db, queries := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{
		{
			match: `FROM "KRENALIS_PROFILE_SCHEMA_VERSIONS"`,
			cols:  []string{"VERSION"},
			rows:  [][]driver.Value{{int64(1)}},
		},
		{
			match: `FROM "INFORMATION_SCHEMA"."TABLE_PRIVILEGES"`,
			cols:  []string{"TABLE_NAME", "PRIVILEGE_TYPE"},
			rows:  [][]driver.Value{{"KRENALIS_IDENTITIES", "INSERT"}},
		},
	})
	defer db.Close()

	wh := &Snowflake{db: db}
	err := wh.CheckReadOnlyAccess(t.Context())
	assertSettingsNotReadOnly(t, err)

	if got := strings.Join(*queries, "\n"); !strings.Contains(got, `IS_ROLE_IN_SESSION("GRANTEE")`) {
		t.Fatalf("expected role hierarchy check in query, got:\n%s", got)
	}
	if got := strings.Join(*queries, "\n"); !strings.Contains(got, `IS_DATABASE_ROLE_IN_SESSION("GRANTEE")`) {
		t.Fatalf("expected database role hierarchy check in query, got:\n%s", got)
	}
	if !strings.Contains(err.Error(), "KRENALIS_IDENTITIES (INSERT)") {
		t.Fatalf("expected KRENALIS_IDENTITIES INSERT in error, got %q", err.Error())
	}
}

func Test_CheckReadOnlyAccess_acceptsExpectedReadOnlySurface(t *testing.T) {
	db, queries := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{
		{
			match: `FROM "KRENALIS_PROFILE_SCHEMA_VERSIONS"`,
			cols:  []string{"VERSION"},
			rows:  [][]driver.Value{{int64(1)}},
		},
		{
			match: `FROM "INFORMATION_SCHEMA"."TABLE_PRIVILEGES"`,
			cols:  []string{"TABLE_NAME", "PRIVILEGE_TYPE"},
		},
	})
	defer db.Close()

	wh := &Snowflake{db: db}
	err := wh.CheckReadOnlyAccess(t.Context())
	if err != nil {
		t.Fatalf("expected read-only access to be accepted, got %s", err)
	}
	if got := len(*queries); got != 2 {
		t.Fatalf("expected 2 queries, got %d:\n%s", got, strings.Join(*queries, "\n"))
	}

	got := strings.Join(*queries, "\n")
	for _, want := range []string{
		`KRENALIS_IDENTITIES`,
		`KRENALIS_PROFILES_1`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected query to contain %q, got:\n%s", want, got)
		}
	}
}

func assertSettingsNotReadOnly(t *testing.T, err error) {
	t.Helper()
	var target *warehouses.SettingsNotReadOnly
	if !errors.As(err, &target) {
		t.Fatalf("expected warehouses.SettingsNotReadOnly, got %T (%v)", err, err)
	}
}

type checkReadOnlyQuery struct {
	match string
	cols  []string
	rows  [][]driver.Value
	err   error
}

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

type checkReadOnlyConnector struct {
	t         *testing.T
	responses []checkReadOnlyQuery
	queries   *[]string
	next      int
}

func (c *checkReadOnlyConnector) Connect(context.Context) (driver.Conn, error) {
	return &checkReadOnlyConn{connector: c}, nil
}

func (c *checkReadOnlyConnector) Driver() driver.Driver {
	return checkReadOnlyDriver{}
}

type checkReadOnlyDriver struct{}

func (checkReadOnlyDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("checkReadOnlyDriver.Open is not used")
}

type checkReadOnlyConn struct {
	connector *checkReadOnlyConnector
}

func (c *checkReadOnlyConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("checkReadOnlyConn.Prepare is not implemented")
}

func (c *checkReadOnlyConn) Close() error {
	return nil
}

func (c *checkReadOnlyConn) Begin() (driver.Tx, error) {
	return nil, errors.New("checkReadOnlyConn.Begin is not implemented")
}

func (c *checkReadOnlyConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	connector := c.connector
	*connector.queries = append(*connector.queries, query)
	if connector.next >= len(connector.responses) {
		connector.t.Fatalf("unexpected query:\n%s", query)
	}

	response := connector.responses[connector.next]
	connector.next++
	if !strings.Contains(query, response.match) {
		connector.t.Fatalf("query %d: expected to contain %q, got:\n%s", connector.next, response.match, query)
	}
	if response.err != nil {
		return nil, response.err
	}
	return &checkReadOnlyRows{cols: response.cols, rows: response.rows}, nil
}

type checkReadOnlyRows struct {
	cols []string
	rows [][]driver.Value
	next int
}

func (r *checkReadOnlyRows) Columns() []string {
	return r.cols
}

func (r *checkReadOnlyRows) Close() error {
	return nil
}

func (r *checkReadOnlyRows) Next(dest []driver.Value) error {
	if r.next >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.next])
	r.next++
	return nil
}
