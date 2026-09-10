// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
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

	const opID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d0"
	_, err := pool.Exec(t.Context(), `INSERT INTO "krenalis_system_operations"
		("id", "operation_type", "completed_at") VALUES ($1, $2, $3)`, opID, identityResolution, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	localErr := errors.New("commit response unavailable")
	err = warehouse.finalizeIdentityResolution(t.Context(), pool, opID, localErr)
	if err != nil {
		t.Fatal(err)
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

// TestAlterProfileSchemaUsesPublishedProfilesAfterIdentityResolutionFailure
// verifies that after a failed Identity Resolution, altering the profile schema
// keeps the last published profiles visible and updates the published profiles
// table.
func TestAlterProfileSchemaUsesPublishedProfilesAfterIdentityResolutionFailure(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	identifiers := profileColumns
	err := warehouse.Initialize(t.Context(), profileColumns)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_profiles_0"
		("_kpid", "_identities", "_updated_at", "email") VALUES ($1, ARRAY[1], $2, $3)`,
		"a44731d8-d89d-44b9-ac87-a8ce1a8770d7", time.Now().UTC(), "person@example.com")
	if err != nil {
		t.Fatal(err)
	}

	badProfileColumns := append([]warehouses.Column{}, profileColumns...)
	badProfileColumns = append(badProfileColumns, warehouses.Column{
		Name:     "column_not_in_profiles_table",
		Type:     types.String(),
		Nullable: true,
	})
	const failedOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d8"
	err = warehouse.ResolveIdentities(t.Context(), failedOpID, identifiers, badProfileColumns, nil)
	if err != nil {
		_, ok := errors.AsType[*warehouses.OperationError](err)
		if !ok {
			t.Fatalf("expected failed operation to return *warehouses.OperationError, got %T: %v", err, err)
		}
	}
	if err == nil {
		t.Fatal("expected Identity Resolution to fail")
	}

	var email string
	err = pool.QueryRow(t.Context(), `SELECT "email" FROM "profiles" WHERE "_kpid" = $1`,
		"a44731d8-d89d-44b9-ac87-a8ce1a8770d7").Scan(&email)
	if err != nil {
		t.Fatal(err)
	}
	if email != "person@example.com" {
		t.Errorf("expected the published profile to remain visible after failed Identity Resolution, got email %q", email)
	}

	var failedProfilesTableExists bool
	err = pool.QueryRow(t.Context(), `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_1'
	)`).Scan(&failedProfilesTableExists)
	if err != nil {
		t.Fatal(err)
	}
	if !failedProfilesTableExists {
		t.Fatal("expected the failed profiles table to exist")
	}

	newProfileColumns := append([]warehouses.Column{}, profileColumns...)
	newProfileColumns = append(newProfileColumns, warehouses.Column{Name: "phone", Type: types.String(), Nullable: true})
	alterOperations := []warehouses.AlterOperation{{
		Operation: warehouses.OperationAddColumn,
		Column:    "phone",
		Type:      types.String(),
	}}
	preview, err := warehouse.PreviewAlterProfileSchema(t.Context(), newProfileColumns, alterOperations)
	if err != nil {
		t.Fatal(err)
	}
	previewSQL := strings.Join(preview, "\n")
	if !strings.Contains(previewSQL, `ALTER TABLE "krenalis_profiles_0"`) {
		t.Errorf("expected preview to alter the published profiles table, got:\n%s", previewSQL)
	}
	if strings.Contains(previewSQL, `ALTER TABLE "krenalis_profiles_1"`) {
		t.Errorf("expected preview not to alter the failed profiles table, got:\n%s", previewSQL)
	}

	const alterOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d9"
	err = warehouse.AlterProfileSchema(t.Context(), alterOpID, newProfileColumns, alterOperations)
	if err != nil {
		t.Fatal(err)
	}

	err = pool.QueryRow(t.Context(), `SELECT "email" FROM "profiles" WHERE "_kpid" = $1`,
		"a44731d8-d89d-44b9-ac87-a8ce1a8770d7").Scan(&email)
	if err != nil {
		t.Fatal(err)
	}
	if email != "person@example.com" {
		t.Errorf("expected the published profile to remain visible, got email %q", email)
	}

	var columnExists bool
	err = pool.QueryRow(t.Context(), `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_0'
			AND column_name = 'phone'
	)`).Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if !columnExists {
		t.Error("expected the published profiles table to contain the new column")
	}
	err = pool.QueryRow(t.Context(), `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_1'
	)`).Scan(&failedProfilesTableExists)
	if err != nil {
		t.Fatal(err)
	}
	if !failedProfilesTableExists {
		t.Fatal("expected the failed profiles table to remain after schema alteration")
	}
	err = pool.QueryRow(t.Context(), `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_1'
			AND column_name = 'phone'
	)`).Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if columnExists {
		t.Error("expected the failed profiles table not to contain the new column")
	}

	const successfulOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770da"
	err = warehouse.ResolveIdentities(t.Context(), successfulOpID, identifiers, newProfileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = pool.QueryRow(t.Context(), `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'krenalis_profiles_2'
			AND column_name = 'phone'
	)`).Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if !columnExists {
		t.Error("expected the next profiles table to copy the published schema")
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
	err := warehouse.ResolveIdentities(t.Context(), failedOpID, profileColumns, badProfileColumns, nil)
	isOperationError := false
	if err != nil {
		_, isOperationError = errors.AsType[*warehouses.OperationError](err)
	}
	if !isOperationError {
		t.Fatalf("expected failed operation to return *warehouses.OperationError, got %v", err)
	}

	const successfulOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d2"
	err = warehouse.ResolveIdentities(t.Context(), successfulOpID, profileColumns, profileColumns, nil)
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
	err = warehouse.ResolveIdentities(t.Context(), conflictingOpID, profileColumns, profileColumns, nil)
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
