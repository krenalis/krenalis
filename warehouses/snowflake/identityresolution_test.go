// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Test_createPendingViewQuery verifies that the pending view selects the table
// corresponding to the persisted operation outcome.
func Test_createPendingViewQuery(t *testing.T) {
	query := createPendingViewQuery("KRENALIS_PROFILES_1", "KRENALIS_PROFILES_2", "operation'id", []warehouses.Column{{
		Name: "email",
		Type: types.String(),
	}})
	wants := []string{
		`SELECT` + "\n\t" + `"_KPID",` + "\n\t" + `"_UPDATED_AT",` + "\n\t" + `"EMAIL"`,
		`SELECT * FROM "KRENALIS_PROFILES_2" WHERE EXISTS (`,
		`WHERE "ID" = 'operation\'id'`,
		`AND "COMPLETED_AT" IS NOT NULL`,
		`AND "RESULT" IS NOT NULL`,
		`AND "ERROR" = ''`,
		`SELECT * FROM "KRENALIS_PROFILES_1" WHERE NOT EXISTS (`,
	}
	for _, want := range wants {
		if !strings.Contains(query, want) {
			t.Errorf("expected pending view query to contain %q, got:\n%s", want, query)
		}
	}
}

// Test_finalizeIdentityResolutionPreservesPersistedSuccess verifies that a
// persisted successful operation takes precedence over a conflicting local
// error.
func Test_finalizeIdentityResolutionPreservesPersistedSuccess(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	if _, err := db.ExecContext(t.Context(), createOperationsTable); err != nil {
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
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_SYSTEM_OPERATIONS"
		("ID", "OPERATION_TYPE", "COMPLETED_AT", "RESULT")
		VALUES (?, ?, ?, PARSE_JSON(?))`, opID, identityResolution, time.Now().UTC(), string(persistedResult))
	if err != nil {
		t.Fatal(err)
	}

	localErr := errors.New("commit response unavailable")
	counts, err := warehouse.finalizeIdentityResolution(t.Context(), db, opID, nil, localErr)
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
	if err := db.QueryRowContext(t.Context(),
		`SELECT "ERROR" FROM "KRENALIS_SYSTEM_OPERATIONS" WHERE "ID" = ?`, opID).Scan(&operationError); err != nil {
		t.Fatal(err)
	}
	if operationError != "" {
		t.Fatalf("expected successful operation to remain successful, got error %q", operationError)
	}

}

// TestResolveIdentitiesPreservesProfilesViewAfterPendingFailure verifies that
// replacing a failed pending operation does not leave the profiles view
// referencing a removed table when the replacement also fails.
func TestResolveIdentitiesPreservesProfilesViewAfterPendingFailure(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	if err := warehouse.Initialize(t.Context(), profileColumns); err != nil {
		t.Fatal(err)
	}

	const failedOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d5"
	_, err := db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_SYSTEM_OPERATIONS"
		("ID", "OPERATION_TYPE", "COMPLETED_AT", "ERROR") VALUES (?, ?, ?, ?)`,
		failedOpID, identityResolution, time.Now().UTC(), "identity resolution failed")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `CREATE TABLE "KRENALIS_PROFILES_1" LIKE "KRENALIS_PROFILES_0"`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_PROFILE_SCHEMA_VERSIONS" ("VERSION", "OPERATION", "TIMESTAMP")`+
		` VALUES (?, ?, ?)`, 1, failedOpID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(),
		createPendingViewQuery("KRENALIS_PROFILES_0", "KRENALIS_PROFILES_1", failedOpID, profileColumns))
	if err != nil {
		t.Fatal(err)
	}

	profilesBefore, err := warehouse.Count(t.Context(), "profiles")
	if err != nil {
		t.Fatal(err)
	}
	badProfileColumns := append([]warehouses.Column{}, profileColumns...)
	badProfileColumns = append(badProfileColumns, warehouses.Column{
		Name:     "column_not_in_profiles_table",
		Type:     types.String(),
		Nullable: true,
	})
	const replacementOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d6"
	_, err = warehouse.ResolveIdentities(t.Context(), replacementOpID, profileColumns, badProfileColumns, nil)
	isOperationError := false
	if err != nil {
		_, isOperationError = errors.AsType[*warehouses.OperationError](err)
	}
	if !isOperationError {
		t.Fatalf("expected replacement operation to return *warehouses.OperationError, got %v", err)
	}
	profilesAfter, err := warehouse.Count(t.Context(), "profiles")
	if err != nil {
		t.Fatal(err)
	}
	if profilesAfter != profilesBefore {
		t.Fatalf("failed replacement changed visible profiles from %d to %d", profilesBefore, profilesAfter)
	}

}

// TestResolveIdentitiesRemovesObsoleteProfileTables verifies that a successful
// operation after a failure removes obsolete profile tables and retains only
// the published table.
func TestResolveIdentitiesRemovesObsoleteProfileTables(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
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
		{name: "KRENALIS_PROFILES_0", shouldExist: false},
		{name: "KRENALIS_PROFILES_1", shouldExist: false},
		{name: "KRENALIS_PROFILES_2", shouldExist: true},
	}
	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = CURRENT_SCHEMA() AND TABLE_NAME = ?`, table.name).Scan(&exists)
		if err != nil {
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

	warehouse, db := newTestSnowflakeWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	if err := warehouse.Initialize(t.Context(), profileColumns); err != nil {
		t.Fatal(err)
	}

	const runningOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d3"
	_, err := db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_SYSTEM_OPERATIONS" ("ID", "OPERATION_TYPE") VALUES (?, ?)`,
		runningOpID, identityResolution)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `CREATE TABLE "KRENALIS_PROFILES_1" LIKE "KRENALIS_PROFILES_0"`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_PROFILE_SCHEMA_VERSIONS" ("VERSION", "OPERATION", "TIMESTAMP")`+
		` VALUES (?, ?, ?)`, 1, runningOpID, time.Now().UTC())
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
		{name: "KRENALIS_PROFILES_1", shouldExist: true},
		{name: "KRENALIS_PROFILES_2", shouldExist: false},
	}
	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = CURRENT_SCHEMA() AND TABLE_NAME = ?`, table.name).Scan(&exists)
		if err != nil {
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

	warehouse, db := newTestSnowflakeWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	if err := warehouse.Initialize(t.Context(), profileColumns); err != nil {
		t.Fatal(err)
	}

	_, err := db.ExecContext(t.Context(), `ALTER TABLE "KRENALIS_SYSTEM_OPERATIONS" DROP COLUMN "RESULT"`)
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"_ANONYMOUS_COUNT", "_RECOGNIZED_COUNT"} {
		if _, err := db.ExecContext(
			t.Context(), `ALTER TABLE "KRENALIS_PROFILES_0" DROP COLUMN `+quoteIdent(column)); err != nil {
			t.Fatal(err)
		}
	}

	const legacyResolutionID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d7"
	_, err = db.ExecContext(t.Context(),
		`INSERT INTO "KRENALIS_SYSTEM_OPERATIONS" ("ID", "OPERATION_TYPE", "COMPLETED_AT")`+
			` VALUES (?, ?, ?)`, legacyResolutionID, identityResolution, time.Now().UTC())
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
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_SYSTEM_OPERATIONS'
			AND COLUMN_NAME = 'RESULT'`).Scan(&resultColumnExists)
	if err != nil {
		t.Fatal(err)
	}
	if !resultColumnExists {
		t.Fatal("expected AlterProfileSchema to add the result column")
	}
	var alterResultIsNull bool
	err = db.QueryRowContext(t.Context(),
		`SELECT "RESULT" IS NULL FROM "KRENALIS_SYSTEM_OPERATIONS" WHERE "ID" = ?`, alterSchemaID).
		Scan(&alterResultIsNull)
	if err != nil {
		t.Fatal(err)
	}
	if !alterResultIsNull {
		t.Fatal("expected AlterProfileSchema not to persist a result")
	}
	var countColumns int
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_PROFILES_0'
			AND COLUMN_NAME IN ('_ANONYMOUS_COUNT', '_RECOGNIZED_COUNT')`).Scan(&countColumns)
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

	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_PROFILES_1'
			AND COLUMN_NAME IN ('_ANONYMOUS_COUNT', '_RECOGNIZED_COUNT')`).Scan(&countColumns)
	if err != nil {
		t.Fatal(err)
	}
	if countColumns != 2 {
		t.Fatalf("expected two count columns in the new profiles table, got %d", countColumns)
	}

	var resultPersisted bool
	err = db.QueryRowContext(t.Context(),
		`SELECT "RESULT" IS NOT NULL FROM "KRENALIS_SYSTEM_OPERATIONS" WHERE "ID" = ?`,
		resolutionID).Scan(&resultPersisted)
	if err != nil {
		t.Fatal(err)
	}
	if !resultPersisted {
		t.Fatal("expected Identity Resolution to persist its result")
	}

}
