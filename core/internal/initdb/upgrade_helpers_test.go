// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/kms"
)

// This file holds the helpers of the test of the database upgrade of a release,
// upgrade_release_test.go, which exists only while a release is prepared. See
// .agents/skills/release-upgrade/SKILL.md.

// schemaSnapshotQuery describes the objects of the current schema regardless
// of how they were created and of the order of the columns.
const schemaSnapshotQuery = `
	SELECT jsonb_build_object(
		'columns', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, a.attname, format_type(a.atttypid, a.atttypmod),
				a.attnotnull, a.attidentity, pg_get_expr(d.adbin, d.adrelid)) ORDER BY c.relname, a.attname)
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
		'triggers', (
			SELECT jsonb_agg(pg_get_triggerdef(tg.oid) ORDER BY c.relname, tg.tgname)
			FROM pg_trigger tg
			JOIN pg_class c ON c.oid = tg.tgrelid
			WHERE c.relnamespace = current_schema()::regnamespace AND NOT tg.tgisinternal),
		'functions', (
			SELECT jsonb_agg(pg_get_functiondef(p.oid) ORDER BY pg_get_functiondef(p.oid))
			FROM pg_proc p
			WHERE p.pronamespace = current_schema()::regnamespace)
	)::text`

// testKMSKey is the key of the local KMS used by loadTestState.
const testKMSKey = "key:DJ0UMRTROH4pjY/Esh3fAErsPbdYmvsnfDCZtc9K4iU"

// TestTestDatabaseFromSQL verifies that newTestDatabaseFromSQL, given the SQL
// files of the current schema, creates the same schema as initialize, and that
// loadTestState loads the state from a newly initialized database.
func TestTestDatabaseFromSQL(t *testing.T) {

	t.Run("schema", func(t *testing.T) {
		t.Parallel()
		files := fstest.MapFS{
			"schema.sql":              {Data: []byte(schema)},
			"rate_limiter_leases.sql": {Data: []byte(createRateLimiterLeasesFunction)},
		}
		expected := schemaSnapshot(t, newInitializedTestDatabase(t))
		got := schemaSnapshot(t, newTestDatabaseFromSQL(t, files, schemaVersion))
		if got != expected {
			t.Fatalf("expected schema %s, got %s", expected, got)
		}
	})

	t.Run("state", func(t *testing.T) {
		t.Parallel()
		loadTestState(t, newInitializedTestDatabase(t))
	})

}

// assertQueryJSON verifies that query returns a single JSON value equal to
// expected, comparing them as jsonb values.
func assertQueryJSON(t *testing.T, database *db.DB, query, expected string) {
	t.Helper()
	var equal bool
	var got string
	err := database.QueryRow(t.Context(), `SELECT q = $1::jsonb, q::text FROM (`+query+`) AS r(q)`, expected).
		Scan(&equal, &got)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

// loadTestState loads the state of Krenalis from the database, as Krenalis does
// at startup, and fails the test if it cannot. Before loading it, it replaces
// the KMS-encrypted keys of the metadata table with keys of a local KMS.
// Connections and pipelines can be loaded only if their connectors are
// included in the test binary.
func loadTestState(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := t.Context()
	keyManager, err := kms.New(ctx, testKMSKey)
	if err != nil {
		t.Fatal(err)
	}
	httpSecretKey, oauthKey, notificationKey, apiKeyPepper, err := generateKmsEncryptedKeys(ctx, keyManager)
	if err != nil {
		t.Fatal(err)
	}
	err = database.Transaction(ctx, func(tx *db.Tx) error {
		return initializeKmsEncryptedKeys(ctx, tx, httpSecretKey, oauthKey, notificationKey, apiKeyPepper)
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := state.New(ctx, database, keyManager, nil, false)
	if err != nil {
		t.Fatalf("expected the state to be loaded, got error %q", err)
	}
	st.Close(ctx)
}

// newBaseTestDatabase returns a test database with the schema of the base
// release, created from its SQL files in testdata/<base> by
// newTestDatabaseFromSQL. If the base has the schema_version table, the
// database is at the previous schema version, finalized.
func newBaseTestDatabase(t *testing.T, base string) *db.DB {
	t.Helper()
	return newTestDatabaseFromSQL(t, os.DirFS(path.Join("testdata", base)), schemaVersion-1)
}

// newTestDatabaseFromSQL returns a test database created from the SQL files of
// fsys as initialize creates it: it executes schema.sql one statement at a
// time, then each other .sql file as a whole, in lexical order. If the schema
// has the schema_version table, it records version, finalized. It does not
// insert the organization and the member that initialize inserts.
func newTestDatabaseFromSQL(t *testing.T, fsys fs.FS, version int) *db.DB {
	t.Helper()
	ctx := t.Context()
	database := newTestDatabase(t)
	data, err := fs.ReadFile(fsys, "schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	for query := range strings.SplitSeq(string(data), ";\n") {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err = database.Exec(ctx, query)
		if err != nil {
			t.Fatal(err)
		}
	}
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(names)
	for _, name := range names {
		if name == "schema.sql" {
			continue
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = database.Exec(ctx, string(data))
		if err != nil {
			t.Fatalf("cannot execute %s: %s", name, err)
		}
	}
	var hasVersion bool
	err = database.QueryRow(ctx, "SELECT to_regclass('schema_version') IS NOT NULL").Scan(&hasVersion)
	if err != nil {
		t.Fatal(err)
	}
	if hasVersion {
		_, err = database.Exec(ctx, "INSERT INTO schema_version (version, finalized) VALUES ($1, true)", version)
		if err != nil {
			t.Fatal(err)
		}
	}
	return database
}

// schemaSnapshot returns a description of the objects of the schema of the
// database, which is equal for two databases with the same objects, regardless
// of how they were created and of the order of the columns.
func schemaSnapshot(t *testing.T, database *db.DB) string {
	t.Helper()
	var snapshot string
	err := database.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
