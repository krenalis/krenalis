// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestRequiredConsentPurposeLocking verifies the compatibility and scope of the
// workspace and pipeline locks acquired by required consent purpose checks,
// and the errors returned for missing resources.
func TestRequiredConsentPurposeLocking(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}
	container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
		postgres.WithDatabase("krenalis"), postgres.WithUsername("krenalis"), postgres.WithPassword("krenalis"),
		postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("expected PostgreSQL container, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container termination, got %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected container host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected container port, got %v", err)
	}
	database, err := db.Open(&db.Options{
		Host: host, Port: int(port.Num()), Username: "krenalis", Password: "krenalis", Database: "krenalis",
	})
	if err != nil {
		t.Fatalf("expected database pool, got %v", err)
	}
	t.Cleanup(database.Close)
	_, err = database.Exec(t.Context(), `
		CREATE TABLE workspaces (id text PRIMARY KEY, name text);
		CREATE TABLE pipelines (id text PRIMARY KEY, required_consents_purposes varchar[]);
		CREATE TABLE consent_purposes (
			id text PRIMARY KEY, workspace text, event_purpose_codes varchar[] NOT NULL, profile_property text NOT NULL
		);
		INSERT INTO workspaces VALUES ('workspace', 'Workspace'), ('other-workspace', 'Other Workspace');
		INSERT INTO pipelines VALUES ('pipeline', '{purpose}'), ('other', '{}');
		INSERT INTO consent_purposes VALUES ('purpose', 'workspace', '{marketing}', 'marketing');`)
	if err != nil {
		t.Fatalf("expected consent fixtures, got %v", err)
	}

	tests := []struct {
		name         string
		pipeline     string
		purposes     []string
		query        string
		wantConflict bool
	}{
		{"concurrent creation", "", []string{"purpose"}, "SELECT FROM workspaces WHERE id = 'workspace' FOR KEY SHARE NOWAIT", false},
		{"purpose change during creation", "", []string{"purpose"}, "SELECT FROM workspaces WHERE id = 'workspace' FOR UPDATE NOWAIT", true},
		{"purpose change during update", "pipeline", []string{"purpose"}, "SELECT FROM workspaces WHERE id = 'workspace' FOR UPDATE NOWAIT", true},
		{
			"same pipeline update", "pipeline", []string{"purpose"},
			"SELECT FROM pipelines WHERE id = 'pipeline' FOR NO KEY UPDATE NOWAIT", true,
		},
		{"same pipeline check serialization", "pipeline", []string{"purpose"}, "SELECT FROM pipelines WHERE id = 'pipeline' FOR SHARE NOWAIT", true},
		{
			"different pipeline update", "pipeline", []string{"purpose"},
			"SELECT FROM pipelines WHERE id = 'other' FOR UPDATE NOWAIT", false,
		},
		{"different workspace update", "", []string{"purpose"}, "SELECT FROM workspaces WHERE id = 'other-workspace' FOR UPDATE NOWAIT", false},
		{"different workspace update during pipeline update", "pipeline", []string{"purpose"}, "SELECT FROM workspaces WHERE id = 'other-workspace' FOR UPDATE NOWAIT", false},
		{"workspace metadata update", "", []string{"purpose"}, "SELECT FROM workspaces WHERE id = 'workspace' FOR NO KEY UPDATE NOWAIT", false},
		{"creation without purposes", "", nil, "SELECT FROM workspaces WHERE id = 'workspace' FOR UPDATE NOWAIT", false},
		{"update without purposes", "pipeline", nil, "SELECT FROM workspaces WHERE id = 'workspace' FOR UPDATE NOWAIT", false},
		{"same pipeline update without purposes", "pipeline", nil, "SELECT FROM pipelines WHERE id = 'pipeline' FOR UPDATE NOWAIT", false},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			first, err := database.Begin(t.Context())
			if err != nil {
				t.Fatalf("expected first transaction, got %v", err)
			}
			defer first.Rollback(t.Context())
			err = checkRequiredConsentPurposesTx(t.Context(), first,
				"workspace", test.pipeline, state.TargetEvent, test.purposes)
			if err != nil {
				t.Fatalf("expected valid required purposes, got %v", err)
			}

			second, err := database.Begin(t.Context())
			if err != nil {
				t.Fatalf("expected second transaction, got %v", err)
			}
			defer second.Rollback(t.Context())
			exists, err := second.QueryExists(t.Context(), test.query)
			if err != nil {
				pgErr, ok := errors.AsType[*pgconn.PgError](err)
				if !test.wantConflict || !ok || pgErr.Code != "55P03" {
					t.Fatalf("expected lock conflict %t, got %v", test.wantConflict, err)
				}
				return
			}
			if test.wantConflict {
				t.Fatal("expected lock conflict, got successful lock acquisition")
			}
			if !exists {
				t.Fatal("expected an existing row to lock, got no rows")
			}

		})

	}

	for _, pipeline := range []struct {
		name string
		id   string
	}{
		{"creation", ""},
		{"update", "pipeline"},
	} {

		t.Run("missing workspace during "+pipeline.name, func(t *testing.T) {

			tx, err := database.Begin(t.Context())
			if err != nil {
				t.Fatalf("expected transaction, got %v", err)
			}
			defer tx.Rollback(t.Context())

			err = checkRequiredConsentPurposesTx(t.Context(), tx,
				"missing", pipeline.id, state.TargetEvent, []string{"first", "second"})
			if err != nil {
				unprocessable, ok := errors.AsType[*errors.UnprocessableError](err)
				if !ok || unprocessable.Code != ConsentPurposeNotExist {
					t.Fatalf("expected ConsentPurposeNotExist, got %v", err)
				}
				if err.Error() != "consent purpose first does not exist" {
					t.Fatalf("expected error for first purpose, got %v", err)
				}
				return
			}

			t.Fatal("expected ConsentPurposeNotExist, got nil")

		})

	}

	t.Run("missing pipeline during update", func(t *testing.T) {

		tx, err := database.Begin(t.Context())
		if err != nil {
			t.Fatalf("expected transaction, got %v", err)
		}
		defer tx.Rollback(t.Context())

		err = checkRequiredConsentPurposesTx(t.Context(), tx,
			"workspace", "missing-pipeline", state.TargetEvent, []string{"purpose"})
		if err != nil {
			if _, ok := errors.AsType[*errors.NotFoundError](err); !ok {
				t.Fatalf("expected NotFoundError, got %v", err)
			}
			if err.Error() != "pipeline missing-pipeline does not exist" {
				t.Fatalf("expected error for missing pipeline, got %v", err)
			}
			return
		}

		t.Fatal("expected NotFoundError, got nil")

	})

}
