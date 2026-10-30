// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
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

// Test_finalizeIdentityResolutionMarksCanceledOperationAsFailed verifies that
// an operation whose context has been canceled, as happens on shutdown, is
// still marked as failed.
func Test_finalizeIdentityResolutionMarksCanceledOperationAsFailed(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	_, err := pool.Exec(t.Context(), createSystemOperationsTable)
	if err != nil {
		t.Fatal(err)
	}

	const opID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d1"
	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_system_operations"
		("id", "operation_type") VALUES ($1, $2)`, opID, identityResolution)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = warehouse.finalizeIdentityResolution(ctx, pool, opID, ctx.Err())
	if err != nil {
		_, ok := errors.AsType[*warehouses.OperationError](err)
		if !ok {
			t.Fatalf("expected *warehouses.OperationError, got %T: %v", err, err)
		}
	}
	if err == nil {
		t.Fatal("expected *warehouses.OperationError, got nil")
	}
	var completedAt *time.Time
	var operationError string
	err = pool.QueryRow(t.Context(),
		`SELECT "completed_at", "error" FROM "krenalis_system_operations" WHERE "id" = $1`, opID).
		Scan(&completedAt, &operationError)
	if err != nil {
		t.Fatal(err)
	}
	if completedAt == nil {
		t.Fatal("expected operation to be completed, got uncompleted operation")
	}
	if operationError == "" {
		t.Fatal("expected operation to be failed, got successful operation")
	}

}

// TestResolveIdentitiesAddsMissingIdentityCountColumns verifies that a new
// profiles table includes identity count columns when copied from a previous
// schema.
func TestResolveIdentitiesAddsMissingIdentityCountColumns(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	err := warehouse.Initialize(t.Context(), profileColumns)
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"_anonymous_count", "_recognized_count"} {
		_, err = pool.Exec(t.Context(), `ALTER TABLE "krenalis_profiles_0" DROP COLUMN `+quoteIdent(column))
		if err != nil {
			t.Fatal(err)
		}
	}

	const opID = "a44731d8-d89d-44b9-ac87-a8ce1a8770d7"
	err = warehouse.ResolveIdentities(t.Context(), opID, profileColumns, profileColumns, nil)
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

// TestResolveIdentitiesAssociatesEventsWithProfiles verifies that an event
// with a user ID is associated with the recognized identity with that ID, and
// that an event without one is associated through its anonymous ID with a
// recognized identity holding it or, failing that, with the anonymous identity
// with that ID. Only identities of the event's connection are considered.
func TestResolveIdentitiesAssociatesEventsWithProfiles(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	profileColumns := []warehouses.Column{{Name: "email", Type: types.String(), Nullable: true}}
	err := warehouse.Initialize(t.Context(), profileColumns)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_identities"
		("_pipeline", "_is_anonymous", "_identity_id", "_connection", "_anonymous_ids", "_updated_at") VALUES
		('10', false, 'user-1', '1', ARRAY['anonymous-1'], $1),
		('10', true, 'anonymous-2', '1', NULL, $1),
		('10', true, 'anonymous-3', '1', NULL, $1),
		('10', false, 'user-3', '1', ARRAY['anonymous-3'], $1),
		('20', false, 'user-4', '2', ARRAY['anonymous-2'], $1)`, now)
	if err != nil {
		t.Fatal(err)
	}

	// identity is the identity ID of the identity whose profile the event is
	// expected to be associated with, or empty if it is expected to remain
	// unassociated.
	events := []struct {
		messageID   string
		connection  string
		userID      any
		anonymousID string
		identity    string
	}{
		{messageID: "1", connection: "1", userID: "user-1", anonymousID: "anonymous-9", identity: "user-1"},
		{messageID: "2", connection: "1", userID: nil, anonymousID: "anonymous-1", identity: "user-1"},
		{messageID: "3", connection: "1", userID: nil, anonymousID: "anonymous-2", identity: "anonymous-2"},
		{messageID: "4", connection: "1", userID: "user-9", anonymousID: "anonymous-1", identity: ""},
		{messageID: "5", connection: "1", userID: nil, anonymousID: "user-1", identity: ""},
		{messageID: "6", connection: "1", userID: "anonymous-2", anonymousID: "anonymous-9", identity: ""},
		{messageID: "7", connection: "2", userID: "user-1", anonymousID: "anonymous-9", identity: ""},
		{messageID: "8", connection: "2", userID: nil, anonymousID: "anonymous-1", identity: ""},
		{messageID: "9", connection: "2", userID: nil, anonymousID: "anonymous-2", identity: "user-4"},
		{messageID: "10", connection: "1", userID: nil, anonymousID: "anonymous-3", identity: "user-3"},
	}
	for _, event := range events {
		_, err = pool.Exec(t.Context(), `INSERT INTO "krenalis_events"
			("connection_id", "anonymous_id", "message_id",
			"received_at", "sent_at", "timestamp", "traits", "type", "user_id")
			VALUES ($1, $2, $3, $4, $4, $4, '{}', 'track', $5)`,
			event.connection, event.anonymousID, event.messageID, now, event.userID)
		if err != nil {
			t.Fatal(err)
		}
	}

	const opID = "a44731d8-d89d-44b9-ac87-a8ce1a8770db"
	err = warehouse.ResolveIdentities(t.Context(), opID, profileColumns, profileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, event := range events {
		var identity string
		err = pool.QueryRow(t.Context(), `SELECT coalesce("i"."_identity_id", '')
			FROM "krenalis_events" "e"
			LEFT JOIN "krenalis_identities" "i" ON "i"."_kpid" = "e"."kpid"
			WHERE "e"."message_id" = $1`, event.messageID).Scan(&identity)
		if err != nil {
			t.Fatal(err)
		}
		if identity != event.identity {
			t.Errorf("event %s: expected association with identity %q, got %q",
				event.messageID, event.identity, identity)
		}
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
		("_kpid", "_identities", "_anonymous_count", "_recognized_count", "_updated_at", "email")
		VALUES ($1, ARRAY[1], 0, 0, $2, $3)`, "a44731d8-d89d-44b9-ac87-a8ce1a8770d7", time.Now().UTC(), "person@example.com")
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
