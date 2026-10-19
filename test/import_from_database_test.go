// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func TestImportFromDatabase(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	pgSQL := k.CreateSourcePostgreSQL()

	importUsers := k.CreatePipeline(pgSQL, "User", krenalistester.PipelineToSet{
		Name:    "Import users",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "id", Type: types.String().WithMaxLength(12), Nullable: true},
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		Query:           `SELECT id, 'a@b' as "email", 'ABC123' as "customer_id" FROM members LIMIT ${limit}`,
		UserIDColumn:    "id",
		UpdatedAtColumn: "",
		UpdatedAtFormat: "",
	})

	run := k.StartPipelineRun(importUsers)

	k.WaitForRunsCompletion(run)

	identities, total := k.ConnectionIdentities(pgSQL, 0, 100)

	const expectedCount = 1
	if total != expectedCount {
		t.Fatalf("expected %d identities, got %d", expectedCount, total)
	}

	for _, identity := range identities {
		if identity.Pipeline != importUsers {
			t.Fatalf("expected identity pipeline %s, got %s", importUsers, identity.Pipeline)
		}
	}
}

// TestImportFromDatabasePreservesCursorOnRecordError verifies that an
// incremental import preserves its cursor when the only source result is a
// record-level error whose update time was not read and remains zero.
func TestImportFromDatabasePreservesCursorOnRecordError(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	ctx := t.Context()
	k.ExecQueryTestDatabase(ctx, `
		CREATE TABLE import_cursor_records (
			id TEXT,
			email TEXT,
			updated_at TIMESTAMPTZ NOT NULL
		)`)
	defer k.ExecQueryTestDatabase(ctx, "DROP TABLE import_cursor_records")

	firstUpdatedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	k.ExecQueryTestDatabase(ctx,
		"INSERT INTO import_cursor_records (id, email, updated_at) VALUES ($1, $2, $3)",
		"user-1", "user-1@example.com", firstUpdatedAt)

	pgSQL := k.CreateSourcePostgreSQL()
	pipeline := k.CreatePipeline(pgSQL, "User", krenalistester.PipelineToSet{
		Name:        "Import users incrementally",
		Enabled:     true,
		Incremental: true,
		InSchema: types.Object([]types.Property{
			{Name: "id", Type: types.String(), Nullable: true},
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "updated_at", Type: types.DateTime(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		Query: `
			SELECT id, email, updated_at
			FROM import_cursor_records
			WHERE updated_at >= COALESCE(${updated_at}, '-infinity'::timestamptz)
			ORDER BY updated_at
			LIMIT ${limit}`,
		UserIDColumn:    "id",
		UpdatedAtColumn: "updated_at",
	})

	run := k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletion(run)

	var cursor time.Time
	k.QueryRowTestDatabase(ctx, &cursor, "SELECT cursor FROM pipelines WHERE id = $1", pipeline)
	if !cursor.Equal(firstUpdatedAt) {
		t.Fatalf("expected initial cursor %s, got %s", firstUpdatedAt, cursor)
	}

	secondUpdatedAt := firstUpdatedAt.Add(time.Hour)
	k.ExecQueryTestDatabase(ctx, "TRUNCATE import_cursor_records")
	k.ExecQueryTestDatabase(ctx,
		"INSERT INTO import_cursor_records (id, email, updated_at) VALUES (NULL, $1, $2)",
		"invalid@example.com", secondUpdatedAt)

	run = k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletionAllowFailed(run)

	completedRun := k.PipelineRun(run)
	if failed := completedRun.Failed[core.ReceiveStep]; failed != 1 {
		t.Fatalf("expected one record receive failure, got %d", failed)
	}

	k.QueryRowTestDatabase(ctx, &cursor, "SELECT cursor FROM pipelines WHERE id = $1", pipeline)
	if !cursor.Equal(firstUpdatedAt) {
		t.Fatalf("expected cursor to remain %s after a record error, got %s", firstUpdatedAt, cursor)
	}

}
