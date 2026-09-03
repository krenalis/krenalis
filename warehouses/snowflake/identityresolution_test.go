// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
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

	const opID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d0"
	_, err := db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_SYSTEM_OPERATIONS"
		("ID", "OPERATION_TYPE", "COMPLETED_AT") VALUES (?, ?, ?)`, opID, identityResolution, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	localErr := errors.New("commit response unavailable")
	err = warehouse.finalizeIdentityResolution(t.Context(), db, opID, localErr)
	if err != nil {
		t.Fatal(err)
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

// TestResolveIdentitiesFinalizesCurrentProfilesVersionOnRetry verifies that
// retrying the operation that published the current profiles version finalizes
// the staged view and removes obsolete profile tables, while retrying an older
// completed operation leaves the current view unchanged.
func TestResolveIdentitiesFinalizesCurrentProfilesVersionOnRetry(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	olderProfileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	err := warehouse.Initialize(t.Context(), olderProfileColumns)
	if err != nil {
		t.Fatal(err)
	}

	const olderOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770db"
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_SYSTEM_OPERATIONS"
		("ID", "OPERATION_TYPE", "COMPLETED_AT") VALUES (?, ?, ?)`, olderOpID, identityResolution, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `CREATE TABLE "KRENALIS_PROFILES_1" LIKE "KRENALIS_PROFILES_0"`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_PROFILE_SCHEMA_VERSIONS"
		("VERSION", "OPERATION", "TIMESTAMP") VALUES (?, ?, ?)`, 1, olderOpID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `ALTER TABLE "KRENALIS_PROFILES_1" ADD COLUMN "PHONE" VARCHAR`)
	if err != nil {
		t.Fatal(err)
	}

	currentProfileColumns := []warehouses.Column{
		{Name: "email", Type: types.String(), Nullable: true},
		{Name: "phone", Type: types.String(), Nullable: true},
	}
	const currentOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770dc"
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_SYSTEM_OPERATIONS"
		("ID", "OPERATION_TYPE", "COMPLETED_AT") VALUES (?, ?, ?)`, currentOpID, identityResolution, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `CREATE TABLE "KRENALIS_PROFILES_2" LIKE "KRENALIS_PROFILES_1"`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_PROFILES_2"
		("_KPID", "_IDENTITIES", "_UPDATED_AT", "EMAIL", "PHONE")
		SELECT ?, ARRAY_CONSTRUCT(1), ?, ?, ?`,
		"a44731d8-d89d-44b9-ac87-a8ce1a8770dd", time.Now().UTC(), "person@example.com", "+390000000000")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_PROFILE_SCHEMA_VERSIONS"
		("VERSION", "OPERATION", "TIMESTAMP") VALUES (?, ?, ?)`, 2, currentOpID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), createPendingViewQuery("KRENALIS_PROFILES_1", "KRENALIS_PROFILES_2",
		currentOpID, currentProfileColumns))
	if err != nil {
		t.Fatal(err)
	}

	err = warehouse.ResolveIdentities(t.Context(), currentOpID, nil, currentProfileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"KRENALIS_PROFILES_0", "KRENALIS_PROFILES_1"} {
		var exists bool
		err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = CURRENT_SCHEMA() AND TABLE_NAME = ?`, name).Scan(&exists)
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Errorf("expected obsolete profiles table %q to be removed", name)
		}
	}

	err = warehouse.ResolveIdentities(t.Context(), olderOpID, nil, olderProfileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}
	var phone string
	err = db.QueryRowContext(t.Context(), `SELECT "PHONE" FROM "PROFILES"`).Scan(&phone)
	if err != nil {
		t.Fatal(err)
	}
	if phone != "+390000000000" {
		t.Fatalf("older operation changed the published profiles view, got phone %q", phone)
	}

	// A staged view still depends on the operation row to select the new table.
	// After finalization, removing that row must not affect the published view.
	_, err = db.ExecContext(t.Context(), `DELETE FROM "KRENALIS_SYSTEM_OPERATIONS" WHERE "ID" = ?`, currentOpID)
	if err != nil {
		t.Fatal(err)
	}
	var profileCount int
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM "PROFILES"`).Scan(&profileCount)
	if err != nil {
		t.Fatal(err)
	}
	if profileCount != 1 {
		t.Fatalf("expected the finalized view to expose one profile, got %d", profileCount)
	}

}

// TestAlterProfileSchemaUsesPublishedProfilesAfterIdentityResolutionFailure
// verifies that after a failed Identity Resolution, altering the profile schema
// keeps the last published profiles visible and updates the published profiles
// table.
func TestAlterProfileSchemaUsesPublishedProfilesAfterIdentityResolutionFailure(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	identifiers := profileColumns
	err := warehouse.Initialize(t.Context(), profileColumns)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.ExecContext(t.Context(), `INSERT INTO "KRENALIS_PROFILES_0"
		("_KPID", "_IDENTITIES", "_UPDATED_AT", "EMAIL") SELECT ?, ARRAY_CONSTRUCT(1), ?, ?`,
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
	err = db.QueryRowContext(t.Context(), `SELECT "EMAIL" FROM "PROFILES" WHERE "_KPID" = ?`,
		"a44731d8-d89d-44b9-ac87-a8ce1a8770d7").Scan(&email)
	if err != nil {
		t.Fatal(err)
	}
	if email != "person@example.com" {
		t.Errorf("expected the published profile to remain visible after failed Identity Resolution, got email %q", email)
	}

	var failedProfilesTableExists bool
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_PROFILES_1'`).Scan(&failedProfilesTableExists)
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
	if !strings.Contains(previewSQL, `ALTER TABLE "KRENALIS_PROFILES_0"`) {
		t.Errorf("expected preview to alter the published profiles table, got:\n%s", previewSQL)
	}
	if strings.Contains(previewSQL, `ALTER TABLE "KRENALIS_PROFILES_1"`) {
		t.Errorf("expected preview not to alter the failed profiles table, got:\n%s", previewSQL)
	}

	const alterOpID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d9"
	err = warehouse.AlterProfileSchema(t.Context(), alterOpID, newProfileColumns, alterOperations)
	if err != nil {
		t.Fatal(err)
	}

	err = db.QueryRowContext(t.Context(), `SELECT "EMAIL" FROM "PROFILES" WHERE "_KPID" = ?`,
		"a44731d8-d89d-44b9-ac87-a8ce1a8770d7").Scan(&email)
	if err != nil {
		t.Fatal(err)
	}
	if email != "person@example.com" {
		t.Errorf("expected the published profile to remain visible, got email %q", email)
	}

	var columnExists bool
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_PROFILES_0'
			AND COLUMN_NAME = 'PHONE'`).Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if !columnExists {
		t.Error("expected the published profiles table to contain the new column")
	}
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_PROFILES_1'`).Scan(&failedProfilesTableExists)
	if err != nil {
		t.Fatal(err)
	}
	if !failedProfilesTableExists {
		t.Fatal("expected the failed profiles table to remain after schema alteration")
	}
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_PROFILES_1'
			AND COLUMN_NAME = 'PHONE'`).Scan(&columnExists)
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

	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) > 0
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = CURRENT_SCHEMA()
			AND TABLE_NAME = 'KRENALIS_PROFILES_2'
			AND COLUMN_NAME = 'PHONE'`).Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if !columnExists {
		t.Error("expected the next profiles table to copy the published schema")
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

	var profilesBefore int
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM "PROFILES"`).Scan(&profilesBefore)
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
	err = warehouse.ResolveIdentities(t.Context(), replacementOpID, profileColumns, badProfileColumns, nil)
	isOperationError := false
	if err != nil {
		_, isOperationError = errors.AsType[*warehouses.OperationError](err)
	}
	if !isOperationError {
		t.Fatalf("expected replacement operation to return *warehouses.OperationError, got %v", err)
	}
	var profilesAfter int
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM "PROFILES"`).Scan(&profilesAfter)
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
