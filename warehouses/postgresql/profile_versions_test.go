// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jackc/pgx/v5"
)

// TestProfileVersionReadValidation verifies that profile version reads reject values outside the supported range.
func TestProfileVersionReadValidation(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	for _, statement := range []string{
		createSystemOperationsTable,
		createProfileSchemaVersionsTable,
		`ALTER TABLE "krenalis_profile_schema_versions" ALTER COLUMN "version" TYPE numeric`,
	} {
		_, err := pool.Exec(t.Context(), statement)
		if err != nil {
			t.Fatal(err)
		}
	}
	const opID = "acdbbd41-015f-4943-9afc-99e79f8be132"
	_, err := pool.Exec(t.Context(), `INSERT INTO "krenalis_system_operations"
		("id", "operation_type", "completed_at") VALUES ($1, $2, $3)`, opID, identityResolution, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name               string
		value              string
		want               int
		wantInvalidVersion bool
		wantScanError      bool
	}{
		{name: "empty"},
		{name: "zero", value: "0"},
		{name: "valid", value: "42", want: 42},
		{name: "version limit", value: "2147483647", want: math.MaxInt32},
		{name: "above version limit", value: "2147483648", wantInvalidVersion: true},
		{name: "max int64", value: "9223372036854775807", wantInvalidVersion: true},
		{name: "negative", value: "-1", wantInvalidVersion: true},
		{name: "fractional", value: "1.5", wantScanError: true},
		{name: "above max int64", value: "9223372036854775808", wantScanError: true},
	} {

		t.Run(tc.name, func(t *testing.T) {

			_, err := pool.Exec(t.Context(), `TRUNCATE "krenalis_profile_schema_versions"`)
			if err != nil {
				t.Fatal(err)
			}
			if tc.value != "" {
				_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_profile_schema_versions"
					("version", "operation", "timestamp") VALUES ($1::text::numeric, $2, $3)`,
					tc.value, opID, time.Now().UTC())
				if err != nil {
					t.Fatal(err)
				}
			}
			readers := []struct {
				name                string
				read                func(context.Context) (int, error)
				invalidVersionError string
			}{
				{
					name:                "max",
					read:                warehouse.maxProfilesVersion,
					invalidVersionError: "warehouse returned an invalid profile table version",
				},
				{
					name:                "published",
					read:                warehouse.publishedProfilesVersion,
					invalidVersionError: "warehouse returned an invalid published profile table version",
				},
			}
			for _, reader := range readers {
				t.Run(reader.name, func(t *testing.T) {
					version, err := reader.read(t.Context())
					if tc.wantInvalidVersion {
						if err == nil {
							t.Fatalf("expected an invalid version error, got version %d", version)
						}
						if err.Error() != reader.invalidVersionError {
							t.Fatalf("unexpected invalid version error: %v", err)
						}
						return
					}
					if tc.wantScanError {
						if err == nil {
							t.Fatalf("expected a scan error, got version %d", version)
						}
						_, ok := errors.AsType[pgx.ScanArgError](err)
						if !ok {
							t.Fatalf("expected pgx.ScanArgError, got %T: %v", err, err)
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

// TestResolveIdentitiesProfileVersionBounds verifies that ResolveIdentities accepts the maximum profile version and
// rejects the next one.
func TestResolveIdentitiesProfileVersionBounds(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	columns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	err := warehouse.Initialize(t.Context(), columns)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(t.Context(), `ALTER TABLE "krenalis_profiles_0" RENAME TO "krenalis_profiles_2147483646"`)
	if err != nil {
		t.Fatal(err)
	}
	const previousOpID = "c22212f3-95d9-4117-8358-5465f75ead52"
	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_system_operations"
		("id", "operation_type", "completed_at") VALUES ($1, $2, $3)`, previousOpID, identityResolution, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_profile_schema_versions"
		("version", "operation", "timestamp") VALUES ($1, $2, $3)`, math.MaxInt32-1, previousOpID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	const lastOpID = "996d4292-9967-414d-b5c3-511c6ee71ca1"
	err = warehouse.ResolveIdentities(t.Context(), lastOpID, columns, columns, nil)
	if err != nil {
		t.Fatal(err)
	}
	version, err := warehouse.publishedProfilesVersion(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if version != math.MaxInt32 {
		t.Fatalf("expected the last storable version, got %d", version)
	}

	t.Run("exhausted", func(t *testing.T) {
		const overflowOpID = "279ec998-f0c9-463e-8a2b-e8c3f9c874e8"
		err := warehouse.ResolveIdentities(t.Context(), overflowOpID, columns, columns, nil)
		if err != nil {
			opError, ok := errors.AsType[*warehouses.OperationError](err)
			if !ok || opError.Error() != "profile table version limit reached" {
				t.Fatalf("unexpected operation error: %v", err)
			}
			return
		}
		t.Fatal("expected the version limit to prevent another identity resolution")
	})
	for _, reader := range []struct {
		name string
		read func(context.Context) (int, error)
	}{
		{name: "max", read: warehouse.maxProfilesVersion},
		{name: "published", read: warehouse.publishedProfilesVersion},
	} {
		t.Run(reader.name+" version unchanged", func(t *testing.T) {
			version, err := reader.read(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if version != math.MaxInt32 {
				t.Fatalf("expected version to remain %d, got %d", math.MaxInt32, version)
			}
		})
	}
	err = warehouse.ResolveIdentities(t.Context(), lastOpID, columns, columns, nil)
	if err != nil {
		t.Fatalf("retrying the last successful operation must still succeed: %v", err)
	}
	var profileCount int
	err = pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM "profiles"`).Scan(&profileCount)
	if err != nil {
		t.Fatalf("published profiles are no longer readable: %v", err)
	}
	if profileCount != 0 {
		t.Fatalf("expected no published profiles, got %d", profileCount)
	}

}
