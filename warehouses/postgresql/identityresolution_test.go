// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Test_finalizeIdentityResolutionPreservesPersistedSuccess verifies that a
// persisted successful operation takes precedence over a conflicting local
// error.
func Test_finalizeIdentityResolutionPreservesPersistedSuccess(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	if _, err := pool.Exec(t.Context(), createSystemOperationsTable); err != nil {
		t.Fatal(err)
	}

	want := warehouses.IdentityResolutionCounts{
		Profiles:    warehouses.Counts{Anonymous: 1},
		Identities:  warehouses.Counts{Anonymous: 1},
		Composition: warehouses.IdentityResolutionComposition{One: 1},
	}
	persistedResult, err := json.Marshal(struct {
		Counts warehouses.IdentityResolutionCounts `json:"counts"`
	}{Counts: want})
	if err != nil {
		t.Fatal(err)
	}

	const opID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d0"
	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_system_operations"
		("id", "operation_type", "completed_at", "result")
		VALUES ($1, $2, $3, $4::jsonb)`, opID, identityResolution, time.Now().UTC(), []byte(persistedResult))
	if err != nil {
		t.Fatal(err)
	}

	localErr := errors.New("commit response unavailable")
	counts, err := warehouse.finalizeIdentityResolution(
		t.Context(), pool, opID, nil, localErr)
	if err != nil {
		t.Fatal(err)
	}
	if counts == nil {
		t.Fatal("expected persisted counts, got nil")
	}
	if *counts != want {
		t.Fatalf("expected persisted counts %v, got %v", want, *counts)
	}

	var operationError string
	if err := pool.QueryRow(t.Context(), `SELECT "error" FROM "krenalis_system_operations" WHERE "id" = $1`, opID).
		Scan(&operationError); err != nil {
		t.Fatal(err)
	}
	if operationError != "" {
		t.Fatalf("expected successful operation to remain successful, got error %q", operationError)
	}

}

// TestResolveIdentitiesAddsMissingIdentityCountColumns verifies that a new
// profiles table includes identity count columns when copied from a previous
// schema.
func TestResolveIdentitiesAddsMissingIdentityCountColumns(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	if err := warehouse.Initialize(t.Context(), profileColumns); err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"_anonymous_count", "_recognized_count"} {
		_, err := pool.Exec(t.Context(), `ALTER TABLE "krenalis_profiles_0" DROP COLUMN `+quoteIdent(column))
		if err != nil {
			t.Fatal(err)
		}
	}

	const opID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d7"
	_, err := warehouse.ResolveIdentities(t.Context(), opID, profileColumns, profileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}

	var countColumns int
	err = pool.QueryRow(t.Context(), `SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_1'
			AND column_name IN ('_anonymous_count', '_recognized_count')`).Scan(&countColumns)
	if err != nil {
		t.Fatal(err)
	}
	if countColumns != 2 {
		t.Fatalf("expected two identity count columns in the new profiles table, got %d", countColumns)
	}

}

// TestResolveIdentitiesRemovesObsoleteProfileTables verifies that a successful
// operation after a failure removes obsolete profile tables and retains only
// the published table.
func TestResolveIdentitiesRemovesObsoleteProfileTables(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	if err := warehouse.Initialize(t.Context(), profileColumns); err != nil {
		t.Fatal(err)
	}

	badProfileColumns := append([]warehouses.Column{}, profileColumns...)
	badProfileColumns = append(badProfileColumns, warehouses.Column{
		Name:     "column_not_in_profiles_table",
		Type:     types.String(),
		Nullable: true,
	})
	const failedOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d1"
	_, err := warehouse.ResolveIdentities(t.Context(), failedOpID, profileColumns, badProfileColumns, nil)
	isOperationError := false
	if err != nil {
		_, isOperationError = errors.AsType[*warehouses.OperationError](err)
	}
	if !isOperationError {
		t.Fatalf("expected failed operation to return *warehouses.OperationError, got %v", err)
	}

	const successfulOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d2"
	_, err = warehouse.ResolveIdentities(t.Context(), successfulOpID, profileColumns, profileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}

	tables := []struct {
		name        string
		shouldExist bool
	}{
		{name: "krenalis_profiles_0", shouldExist: false},
		{name: "krenalis_profiles_1", shouldExist: false},
		{name: "krenalis_profiles_2", shouldExist: true},
	}
	for _, table := range tables {
		var exists bool
		if err := pool.QueryRow(t.Context(), `SELECT to_regclass($1) IS NOT NULL`, table.name).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != table.shouldExist {
			t.Fatalf("expected table %q existence to be %t, got %t", table.name, table.shouldExist, exists)
		}
	}

}

// TestResolveIdentitiesRetainsProfileTableFromRunningOperation verifies that
// a second operation fails without removing the unpublished profiles table of
// an operation still in progress.
func TestResolveIdentitiesRetainsProfileTableFromRunningOperation(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	if err := warehouse.Initialize(t.Context(), profileColumns); err != nil {
		t.Fatal(err)
	}

	const runningOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d3"
	_, err := pool.Exec(t.Context(), `INSERT INTO "krenalis_system_operations" ("id", "operation_type") VALUES ($1, $2)`,
		runningOpID, identityResolution)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `CREATE TABLE "krenalis_profiles_1" (LIKE "krenalis_profiles_0")`); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_profile_schema_versions" ("version", "operation", "timestamp")`+
		` VALUES ($1, $2, $3)`, 1, runningOpID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	const conflictingOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d4"
	_, err = warehouse.ResolveIdentities(t.Context(), conflictingOpID, profileColumns, profileColumns, nil)
	isOperationError := false
	if err != nil {
		_, isOperationError = errors.AsType[*warehouses.OperationError](err)
	}
	if !isOperationError {
		t.Fatalf("expected conflicting operation to return *warehouses.OperationError, got %v", err)
	}

	tables := []struct {
		name        string
		shouldExist bool
	}{
		{name: "krenalis_profiles_1", shouldExist: true},
		{name: "krenalis_profiles_2", shouldExist: false},
	}
	for _, table := range tables {
		var exists bool
		if err := pool.QueryRow(t.Context(), `SELECT to_regclass($1) IS NOT NULL`, table.name).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != table.shouldExist {
			t.Fatalf("expected table %q existence to be %t, got %t", table.name, table.shouldExist, exists)
		}
	}

}

// TestResolveIdentitiesUpgradesPreviousWarehouseSchema verifies that operations
// add missing result storage and Identity Resolution adds count columns only to
// the new profiles table.
func TestResolveIdentitiesUpgradesPreviousWarehouseSchema(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	if err := warehouse.Initialize(t.Context(), profileColumns); err != nil {
		t.Fatal(err)
	}

	_, err := pool.Exec(t.Context(), `ALTER TABLE "krenalis_system_operations" DROP COLUMN "result"`)
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"_anonymous_count", "_recognized_count"} {
		if _, err := pool.Exec(t.Context(), `ALTER TABLE "krenalis_profiles_0" DROP COLUMN `+quoteIdent(column)); err != nil {
			t.Fatal(err)
		}
	}

	const legacyResolutionID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d7"
	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_system_operations" ("id", "operation_type", "completed_at")`+
		` VALUES ($1, $2, $3)`, legacyResolutionID, identityResolution, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	alteredProfileColumns := append([]warehouses.Column{}, profileColumns...)
	alteredProfileColumns = append(alteredProfileColumns,
		warehouses.Column{Name: "name", Type: types.String(), Nullable: true})
	const alterSchemaID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d8"
	err = warehouse.AlterProfileSchema(t.Context(), alterSchemaID, alteredProfileColumns, []warehouses.AlterOperation{{
		Operation: warehouses.OperationAddColumn,
		Column:    "name",
		Type:      types.String(),
	}})
	if err != nil {
		t.Fatal(err)
	}

	var resultColumnExists bool
	err = pool.QueryRow(t.Context(), `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_system_operations'
			AND column_name = 'result'
	)`).Scan(&resultColumnExists)
	if err != nil {
		t.Fatal(err)
	}
	if !resultColumnExists {
		t.Fatal("expected AlterProfileSchema to add the result column")
	}
	var alterResultIsNull bool
	err = pool.QueryRow(t.Context(), `SELECT "result" IS NULL FROM "krenalis_system_operations" WHERE "id" = $1`,
		alterSchemaID).Scan(&alterResultIsNull)
	if err != nil {
		t.Fatal(err)
	}
	if !alterResultIsNull {
		t.Fatal("expected AlterProfileSchema not to persist a result")
	}
	var countColumns int
	err = pool.QueryRow(t.Context(), `SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_0'
			AND column_name IN ('_anonymous_count', '_recognized_count')`).Scan(&countColumns)
	if err != nil {
		t.Fatal(err)
	}
	if countColumns != 0 {
		t.Fatalf("expected no count columns in the legacy profiles table, got %d", countColumns)
	}

	counts, err := warehouse.ResolveIdentities(
		t.Context(), legacyResolutionID, profileColumns, alteredProfileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counts == nil {
		t.Fatal("expected zero legacy counts, got nil")
	}
	if *counts != (warehouses.IdentityResolutionCounts{}) {
		t.Fatalf("expected zero legacy counts, got %v", *counts)
	}

	const resolutionID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d9"
	counts, err = warehouse.ResolveIdentities(
		t.Context(), resolutionID, profileColumns, alteredProfileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counts == nil {
		t.Fatal("expected zero counts, got nil")
	}
	if *counts != (warehouses.IdentityResolutionCounts{}) {
		t.Fatalf("expected zero counts, got %v", *counts)
	}

	err = pool.QueryRow(t.Context(), `SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_1'
			AND column_name IN ('_anonymous_count', '_recognized_count')`).Scan(&countColumns)
	if err != nil {
		t.Fatal(err)
	}
	if countColumns != 2 {
		t.Fatalf("expected two count columns in the new profiles table, got %d", countColumns)
	}

	var resultPersisted bool
	err = pool.QueryRow(t.Context(), `SELECT "result" IS NOT NULL FROM "krenalis_system_operations" WHERE "id" = $1`,
		resolutionID).Scan(&resultPersisted)
	if err != nil {
		t.Fatal(err)
	}
	if !resultPersisted {
		t.Fatal("expected Identity Resolution to persist its result")
	}

}
