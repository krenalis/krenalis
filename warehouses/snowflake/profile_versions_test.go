// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/warehouses"
)

// profileVersionConnector adds transaction support to the query test connector.
type profileVersionConnector struct {
	*checkReadOnlyConnector
}

// newProfileVersionTestDB returns a database that records statements issued by profile version operations.
func newProfileVersionTestDB(t *testing.T, responses []checkReadOnlyQuery) (*sql.DB, *[]string) {
	t.Helper()
	queries := make([]string, 0, len(responses))
	connector := &profileVersionConnector{&checkReadOnlyConnector{
		t: t, responses: responses, queries: &queries,
	}}
	return sql.OpenDB(connector), &queries
}

// Connect returns a connection that supports transactions for profile version operations.
func (c *profileVersionConnector) Connect(context.Context) (driver.Conn, error) {
	return &profileVersionConn{&checkReadOnlyConn{connector: c.checkReadOnlyConnector}}, nil
}

// profileVersionConn adds transaction and write support to the query test connection.
type profileVersionConn struct {
	*checkReadOnlyConn
}

// Begin starts a transaction over the configured responses.
func (*profileVersionConn) Begin() (driver.Tx, error) {
	return &profileVersionTx{}, nil
}

// ExecContext validates and records a statement using the configured responses.
func (c *profileVersionConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	rows, err := c.QueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}
	return driver.RowsAffected(1), nil
}

// profileVersionTx completes test transactions without executing SQL.
type profileVersionTx struct{}

// Commit completes the test transaction.
func (*profileVersionTx) Commit() error { return nil }

// Rollback completes the test transaction.
func (*profileVersionTx) Rollback() error { return nil }

// TestFinalizePublishedProfilesDeletesBoundedBatch verifies that finalization deletes a bounded batch before reporting
// that more obsolete profile tables remain.
func TestFinalizePublishedProfilesDeletesBoundedBatch(t *testing.T) {

	const publishedVersion = maxObsoleteProfilesTableVersions + 1
	rows := make([][]driver.Value, maxObsoleteProfilesTableVersions+1)
	for version := range rows {
		rows[version] = []driver.Value{int64(version)}
	}
	responses := []checkReadOnlyQuery{
		{match: `MAX("VERSION")`, cols: []string{"VERSION"}, rows: [][]driver.Value{{int64(publishedVersion)}}},
		{match: `MAX("V"."VERSION")`, cols: []string{"VERSION"}, rows: [][]driver.Value{{int64(publishedVersion)}}},
		{match: `SELECT EXISTS`, cols: []string{"EXISTS"}, rows: [][]driver.Value{{true}}},
		{match: `CREATE OR REPLACE VIEW`},
		{
			match: "LIMIT " + strconv.Itoa(maxObsoleteProfilesTableVersions+1),
			cols:  []string{"VERSION"},
			rows:  rows,
		},
	}
	for version := range maxObsoleteProfilesTableVersions {
		responses = append(responses, checkReadOnlyQuery{
			match: `DROP TABLE IF EXISTS "KRENALIS_PROFILES_` + strconv.Itoa(version) + `"`,
		})
	}
	db, queries := newProfileVersionTestDB(t, responses)
	defer db.Close()
	err := (&Snowflake{db: db}).finalizePublishedProfiles(t.Context(), db, "published-operation", nil)
	if err == nil {
		t.Fatal("expected an error for the remaining obsolete profile table version")
	}
	if err.Error() != "warehouse returned too many obsolete profile table versions" {
		t.Fatalf("unexpected cleanup error: %v", err)
	}
	if len(*queries) != len(responses) {
		t.Fatalf("expected %d statements, got %d", len(responses), len(*queries))
	}

}

// TestObsoleteProfilesTableVersionsResultValidation verifies that invalid
// results are rejected and that too many results produce a bounded cleanup
// list and an error.
func TestObsoleteProfilesTableVersionsResultValidation(t *testing.T) {

	const publishedProfilesVersion = maxObsoleteProfilesTableVersions + 10
	limitValues := make([]driver.Value, maxObsoleteProfilesTableVersions)
	for i := range limitValues {
		limitValues[i] = int64(i)
	}
	surplusValues := append([]driver.Value(nil), limitValues...)
	surplusValues = append(surplusValues, "must not be scanned")
	for _, tc := range []struct {
		name         string
		values       []driver.Value
		wantError    string
		wantAnyError bool
		wantPartial  bool
	}{
		{name: "empty"},
		{name: "valid", values: []driver.Value{int64(0), int64(1)}},
		{name: "last valid", values: []driver.Value{int64(publishedProfilesVersion - 1)}},
		{name: "duplicates", values: []driver.Value{int64(1), int64(1)}},
		{name: "limit", values: limitValues},
		{
			name:        "surplus",
			values:      surplusValues,
			wantError:   "warehouse returned too many obsolete profile table versions",
			wantPartial: true,
		},
		{
			name:      "negative",
			values:    []driver.Value{int64(-1)},
			wantError: "warehouse returned an invalid obsolete profile table version -1",
		},
		{
			name:      "negative after valid",
			values:    []driver.Value{int64(0), int64(-1)},
			wantError: "warehouse returned an invalid obsolete profile table version -1",
		},
		{
			name:   "equal to published version",
			values: []driver.Value{int64(publishedProfilesVersion)},
			wantError: fmt.Sprintf("warehouse returned an invalid obsolete profile table version %d",
				publishedProfilesVersion),
		},
		{name: "missing", values: []driver.Value{nil}, wantAnyError: true},
		{name: "fractional", values: []driver.Value{"1.5"}, wantAnyError: true},
		{
			name:         "scan error after valid",
			values:       []driver.Value{int64(0), "1.5"},
			wantAnyError: true,
		},
		{name: "overflow", values: []driver.Value{"9223372036854775808"}, wantAnyError: true},
	} {

		t.Run(tc.name, func(t *testing.T) {

			rows := make([][]driver.Value, len(tc.values))
			for i, value := range tc.values {
				rows[i] = []driver.Value{value}
			}
			db, _ := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{{
				match: "LIMIT " + strconv.Itoa(maxObsoleteProfilesTableVersions+1),
				cols:  []string{"VERSION"},
				rows:  rows,
			}})
			defer db.Close()
			versions, err := obsoleteProfilesTableVersions(t.Context(), db, publishedProfilesVersion)
			if db.Stats().InUse != 0 {
				t.Fatal("result rows were not closed")
			}
			if err != nil {
				if tc.wantPartial {
					if len(versions) != len(limitValues) {
						t.Fatalf("expected %d cleanup versions, got %d", len(limitValues), len(versions))
					}
					for i, version := range versions {
						if int64(version) != limitValues[i] {
							t.Fatalf("row %d: unexpected version %d", i, version)
						}
					}
				} else if len(versions) != 0 {
					t.Fatalf("returned a partial cleanup list: %v", versions)
				}
				if tc.wantError != "" {
					if err.Error() != tc.wantError {
						t.Fatalf("unexpected validation error: %v", err)
					}
					return
				}
				if tc.wantAnyError {
					return
				}
				t.Fatal(err)
			}
			if tc.wantError != "" || tc.wantAnyError {
				t.Fatalf("expected an error, got versions %v", versions)
			}
			if len(versions) != len(tc.values) {
				t.Fatalf("unexpected version count %d without an error", len(versions))
			}
			for i, version := range versions {
				if int64(version) != tc.values[i] {
					t.Fatalf("row %d: unexpected version %d", i, version)
				}
			}

		})

	}

}

// TestProfileVersionReadValidation verifies that profile version reads reject values outside the supported range.
func TestProfileVersionReadValidation(t *testing.T) {

	for _, tc := range []struct {
		name               string
		value              driver.Value
		want               int
		wantInvalidVersion bool
		wantAnyError       bool
	}{
		{name: "zero", value: "0"},
		{name: "valid", value: "42", want: 42},
		{name: "version limit", value: "2147483647", want: math.MaxInt32},
		{name: "above version limit", value: "2147483648", wantInvalidVersion: true},
		{name: "max int64", value: "9223372036854775807", wantInvalidVersion: true},
		{name: "missing", value: nil, wantAnyError: true},
		{name: "not numeric", value: "invalid", wantAnyError: true},
		{name: "negative", value: "-1", wantInvalidVersion: true},
		{name: "fractional", value: "1.5", wantAnyError: true},
		{name: "above max int64", value: "9223372036854775808", wantAnyError: true},
	} {

		t.Run(tc.name, func(t *testing.T) {
			for _, reader := range []struct {
				name                string
				read                func(*Snowflake, context.Context) (int, error)
				invalidVersionError string
			}{
				{
					name:                "max",
					read:                (*Snowflake).maxProfilesVersion,
					invalidVersionError: "warehouse returned an invalid profile table version",
				},
				{
					name:                "published",
					read:                (*Snowflake).publishedProfilesVersion,
					invalidVersionError: "warehouse returned an invalid published profile table version",
				},
			} {
				t.Run(reader.name, func(t *testing.T) {
					db, _ := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{{
						match: `MAX(`,
						cols:  []string{"VERSION"},
						rows:  [][]driver.Value{{tc.value}},
					}})
					defer db.Close()
					warehouse := &Snowflake{db: db}
					version, err := reader.read(warehouse, t.Context())
					if tc.wantInvalidVersion {
						if err == nil {
							t.Fatalf("expected an invalid version error, got version %d", version)
						}
						if err.Error() != reader.invalidVersionError {
							t.Fatalf("unexpected invalid version error: %v", err)
						}
						return
					}
					if tc.wantAnyError {
						if err == nil {
							t.Fatalf("expected an error, got version %d", version)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if version != tc.want {
						t.Fatalf("expected version %d, got %d", tc.want, version)
					}
				})
			}
		})

	}

}

// TestResolveIdentitiesProfileVersionBounds verifies that ResolveIdentities attempts to create the maximum profile
// version but rejects its successor before DDL.
func TestResolveIdentitiesProfileVersionBounds(t *testing.T) {

	for _, version := range []int{math.MaxInt32 - 1, math.MaxInt32, math.MaxInt32 + 1} {

		t.Run(strconv.Itoa(version), func(t *testing.T) {

			responses := []checkReadOnlyQuery{
				{match: `SELECT "COMPLETED_AT"`, cols: []string{"COMPLETED_AT", "ERROR"}},
				{match: `INSERT INTO "KRENALIS_SYSTEM_OPERATIONS"`},
				{match: `MAX("VERSION")`, cols: []string{"VERSION"}, rows: [][]driver.Value{{int64(version)}}},
			}
			message := "profile table version limit reached"
			if version > math.MaxInt32 {
				message = "warehouse returned an invalid profile table version"
			}
			if version < math.MaxInt32 {
				name := fmt.Sprintf("KRENALIS_PROFILES_%d", version+1)
				responses = append(responses,
					checkReadOnlyQuery{
						match: `MAX("V"."VERSION")`, cols: []string{"VERSION"}, rows: [][]driver.Value{{int64(version)}},
					},
					checkReadOnlyQuery{match: `CREATE TABLE "` + name + `"`, err: errors.New("stop before DDL")},
				)
				message = fmt.Sprintf("cannot create profiles table (with name %s): stop before DDL", quoteIdent(name))
			}
			responses = append(responses,
				checkReadOnlyQuery{match: `UPDATE "KRENALIS_SYSTEM_OPERATIONS"`},
				checkReadOnlyQuery{
					match: `SELECT "COMPLETED_AT"`, cols: []string{"COMPLETED_AT", "ERROR"},
					rows: [][]driver.Value{{time.Now().UTC(), message}},
				},
			)
			db, queries := newProfileVersionTestDB(t, responses)
			defer db.Close()
			err := (&Snowflake{db: db}).ResolveIdentities(t.Context(), "profile-version-boundary", nil, nil, nil)
			if err != nil {
				opError, ok := errors.AsType[*warehouses.OperationError](err)
				if !ok || opError.Error() != message {
					t.Fatalf("unexpected operation error: %v", err)
				}
				if len(*queries) != len(responses) {
					t.Fatalf("expected %d statements, got %d", len(responses), len(*queries))
				}
				statements := strings.Join(*queries, "\n")
				attemptedDDL := strings.Contains(statements, `CREATE TABLE`)
				if attemptedDDL != (version < math.MaxInt32) {
					t.Fatalf("unexpected DDL attempt for current version %d", version)
				}
				if version >= math.MaxInt32 && strings.Contains(statements, `INSERT INTO "KRENALIS_PROFILE_SCHEMA_VERSIONS"`) {
					t.Fatalf("recorded a successor to profile version %d", version)
				}
				return
			}
			t.Fatal("expected the operation to stop at the configured boundary")

		})

	}

}
