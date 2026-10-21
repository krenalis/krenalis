// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"os"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/db"
)

// schemaSnapshotQuery describes the objects of the current schema regardless
// of how they were created.
const schemaSnapshotQuery = `
	SELECT jsonb_build_object(
		'columns', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, a.attname, format_type(a.atttypid, a.atttypmod),
				a.attnotnull, pg_get_expr(d.adbin, d.adrelid)) ORDER BY c.relname, a.attnum)
			FROM pg_attribute a
			JOIN pg_class c ON c.oid = a.attrelid
			LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
			WHERE c.relnamespace = current_schema()::regnamespace AND a.attnum > 0 AND NOT a.attisdropped),
		'constraints', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, con.conname, pg_get_constraintdef(con.oid))
				ORDER BY c.relname, con.conname)
			FROM pg_constraint con
			JOIN pg_class c ON c.oid = con.conrelid
			WHERE c.relnamespace = current_schema()::regnamespace),
		'indexes', (
			SELECT jsonb_agg(pg_get_indexdef(i.indexrelid) ORDER BY c.relname)
			FROM pg_index i
			JOIN pg_class c ON c.oid = i.indexrelid
			WHERE c.relnamespace = current_schema()::regnamespace),
		'enums', (
			SELECT jsonb_agg(jsonb_build_array(t.typname, e.enumlabel) ORDER BY t.typname, e.enumsortorder)
			FROM pg_enum e
			JOIN pg_type t ON t.oid = e.enumtypid
			WHERE t.typnamespace = current_schema()::regnamespace),
		'views', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, pg_get_viewdef(c.oid)) ORDER BY c.relname)
			FROM pg_class c
			WHERE c.relnamespace = current_schema()::regnamespace AND c.relkind = 'v'),
		'functions', (
			SELECT jsonb_agg(pg_get_functiondef(p.oid) ORDER BY pg_get_functiondef(p.oid))
			FROM pg_proc p
			WHERE p.pronamespace = current_schema()::regnamespace)
	)::text`

// TestUpgrade verifies that upgrading a v0.44.0 database produces the schema
// of a newly initialized database, preserves its data, and is idempotent.
func TestUpgrade(t *testing.T) {

	fresh := newInitializedTestDatabase(t)
	var expected string
	err := fresh.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&expected)
	if err != nil {
		t.Fatal(err)
	}

	// Upgrading a newly initialized database must not change it.
	err = Upgrade(t.Context(), fresh)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	err = fresh.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("expected unchanged schema %s, got %s", expected, got)
	}

	database := newV0_44_0TestDatabase(t)

	for range 2 {
		err = Upgrade(t.Context(), database)
		if err != nil {
			t.Fatal(err)
		}
		err = database.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got != expected {
			t.Fatalf("expected schema %s, got %s", expected, got)
		}
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(jsonb_build_array(number, leader, date))
			FROM election`, `[[2147483647, "", "2023-01-01T00:00:00"]]`)
	}

}

// assertUpgradedData verifies that query returns a single JSON value equal to
// expected.
func assertUpgradedData(t *testing.T, database *db.DB, query, expected string) {
	t.Helper()
	var equal bool
	var got string
	err := database.QueryRow(t.Context(), `SELECT q = $1::jsonb, q::text FROM (`+query+`) AS r(q)`, expected).Scan(&equal, &got)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

// newV0_44_0TestDatabase returns a test database with the schema of release
// v0.44.0 and data covering every upgrade step.
func newV0_44_0TestDatabase(t *testing.T) *db.DB {

	t.Helper()

	schema, err := os.ReadFile("testdata/schema_v0.44.0.sql")
	if err != nil {
		t.Fatal(err)
	}
	database := newTestDatabase(t)
	for query := range strings.SplitSeq(string(schema), ";\n") {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err = database.Exec(t.Context(), query)
		if err != nil {
			t.Fatal(err)
		}
	}
	// The rate-limit lease function has not changed since release v0.44.0.
	_, err = database.Exec(t.Context(), createRateLimiterLeasesFunction)
	if err != nil {
		t.Fatal(err)
	}

	// The largest election number representable before the upgrade.
	_, err = database.Exec(t.Context(), `UPDATE election SET number = 2147483647`)
	if err != nil {
		t.Fatal(err)
	}

	return database
}
