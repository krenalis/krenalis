// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"context"
	"encoding/base64"
	"slices"
	"sync"
	"testing"
	"time"

	_ "github.com/krenalis/krenalis/connectors/dummy"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/warehouses"
	_ "github.com/krenalis/krenalis/warehouses/postgresql"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// identityDatastoreFake records Store and CountIdentities calls and returns
// configured results for refresh tests.
type identityDatastoreFake struct {
	counts      *warehouses.IdentityCounts
	err         error
	calls       int
	storeCalls  int
	workspace   string
	pipelines   []string
	beforeCount func()
	available   bool
}

// CountIdentities records the requested pipelines and returns the configured
// result.
func (f *identityDatastoreFake) CountIdentities(_ context.Context, pipelines []string) (*warehouses.IdentityCounts, error) {

	f.calls++
	f.pipelines = slices.Clone(pipelines)
	if f.beforeCount != nil {
		f.beforeCount()
	}

	return f.counts, f.err
}

// Store records the requested workspace and returns the fake identity counter
// with its configured availability.
func (f *identityDatastoreFake) Store(workspace string) (identityCounter, bool) {
	f.storeCalls++
	f.workspace = workspace
	return f, f.available
}

// identityMetricDayExpectation contains the expected values for one historical
// identity metric day.
type identityMetricDayExpectation struct {
	total      int
	anonymous  int
	recognized int
}

// assertIdentityMetricDays verifies a dense known identity history.
func assertIdentityMetricDays(t *testing.T, got []IdentityMetricDay, start time.Time, want []identityMetricDayExpectation) {

	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d identity days, got %d", len(want), len(got))
	}
	for index := range want {
		expectedDay := start.AddDate(0, 0, index)
		if got[index].Day != expectedDay {
			t.Fatalf("expected day %s, got %s", expectedDay, got[index].Day)
		}
		if got[index].Total != want[index].total || got[index].Anonymous != want[index].anonymous ||
			got[index].Recognized != want[index].recognized {
			t.Fatalf("expected counts %d/%d/%d, got %d/%d/%d",
				want[index].total, want[index].anonymous, want[index].recognized,
				got[index].Total, got[index].Anonymous, got[index].Recognized)
		}
	}

}

// assertIdentitySnapshot verifies the persisted workspace and connection rows
// for a full identity snapshot.
func assertIdentitySnapshot(t *testing.T, database *db.DB, expected identitySnapshot) {

	t.Helper()
	day := expected.observedAt.Truncate(24 * time.Hour)
	var observedAt time.Time
	var anonymous, recognized, withoutProfile int
	err := database.QueryRow(t.Context(), `SELECT (day + observed_at) AT TIME ZONE 'UTC',
		identities_anonymous, identities_recognized, identities_without_profile
		FROM identity_metrics
		WHERE workspace = $1 AND day = $2`, expected.workspace, day).Scan(
		&observedAt, &anonymous, &recognized, &withoutProfile)
	if err != nil {
		t.Fatal(err)
	}
	if !observedAt.Equal(expected.observedAt) {
		t.Fatalf("expected observed_at %s, got %s", expected.observedAt, observedAt)
	}
	if anonymous != expected.anonymous || recognized != expected.recognized || withoutProfile != expected.withoutProfile {
		t.Fatalf("expected parent counts %d/%d/%d, got %d/%d/%d",
			expected.anonymous, expected.recognized, expected.withoutProfile, anonymous, recognized, withoutProfile)
	}

	rows, err := database.Query(t.Context(), `SELECT m.connection,
		m.identities_anonymous, m.identities_recognized, m.identities_without_profile
		FROM identity_connection_metrics AS m
		JOIN connections AS c ON c.id = m.connection
		WHERE c.workspace = $1 AND m.day = $2
		ORDER BY m.connection`, expected.workspace, day)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []identityConnection
	for rows.Next() {
		var connection identityConnection
		if err := rows.Scan(&connection.id,
			&connection.anonymous, &connection.recognized, &connection.withoutProfile); err != nil {
			t.Fatal(err)
		}
		got = append(got, connection)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(expected.connections) {
		t.Fatalf("expected %d connection rows, got %d: %#v", len(expected.connections), len(got), got)
	}
	for index := range got {
		if got[index] != expected.connections[index] {
			t.Fatalf("expected connection row %#v, got %#v", expected.connections[index], got[index])
		}
	}

}

// assertIntPointer verifies both pointer presence and integer value.
func assertIntPointer(t *testing.T, got, want *int) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("expected nil, got %d", *got)
		}
		return
	}
	if got == nil {
		t.Fatalf("expected %d, got nil", *want)
	}
	if *got != *want {
		t.Fatalf("expected %d, got %d", *want, *got)
	}
}

// awaitDatabaseLock waits until a query containing queryFragment is blocked on
// a PostgreSQL lock matching the specified wait event.
func awaitDatabaseLock(t *testing.T, database *db.DB, waitEvent, queryFragment string) {

	t.Helper()
	ctx := t.Context()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		waiting, err := database.QueryExists(ctx, `SELECT FROM pg_stat_activity
			WHERE datname = current_database()
				AND pid != pg_backend_pid()
				AND wait_event = $1
				AND query LIKE '%' || $2 || '%'`, waitEvent, queryFragment)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-timer.C:
			t.Fatalf("query containing %q did not wait for a %s lock", queryFragment, waitEvent)
		case <-ticker.C:
		}
	}

}

// blockAdvisoryLock holds a PostgreSQL advisory lock until the returned
// function is called.
func blockAdvisoryLock(t *testing.T, database *db.DB, key int64) func() {

	t.Helper()
	ctx := t.Context()
	connection, err := database.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		connection.Close()
		t.Fatal(err)
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			if _, err := connection.Exec(cleanupCtx, "SELECT pg_advisory_unlock($1)", key); err != nil {
				t.Error(err)
			}
			if err := connection.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	t.Cleanup(release)

	return release
}

// identityMetricsTestOrganization returns an organization created by the
// canonical database initialization.
func identityMetricsTestOrganization(t *testing.T, database *db.DB) string {
	t.Helper()
	var organization string
	if err := database.QueryRow(t.Context(), "SELECT id FROM organizations ORDER BY id LIMIT 1").Scan(&organization); err != nil {
		t.Fatal(err)
	}
	return organization
}

// newIdentityMetricsTestDatabase starts PostgreSQL and initializes the
// canonical schema for metrics persistence tests.
func newIdentityMetricsTestDatabase(t *testing.T) *db.DB {
	t.Helper()
	ctx := t.Context()
	container, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase("krenalis"),
		postgres.WithUsername("krenalis"),
		postgres.WithPassword("krenalis"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Error(err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(&db.Options{
		Host: host, Port: int(port.Num()), Username: "krenalis", Password: "krenalis", Database: "krenalis",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	key, err := kms.New(ctx, "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if err := initdb.InitIfEmpty(ctx, database, key, false); err != nil {
		t.Fatal(err)
	}
	organization := identityMetricsTestOrganization(t, database)
	for _, workspace := range []string{
		"workspace111", "workspace222", "workspace333", "workspace444",
		"identity1111", "emptyconn111", "consistent11", "consistent22", "resoiution11",
	} {
		_, err := database.Exec(ctx, `INSERT INTO workspaces
			(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
			kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
			VALUES ($1, $2, 'metrics test', 'metrics test', 'Normal', $3, $3, $3)`,
			workspace, organization, []byte{1})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, connection := range []struct {
		id        string
		workspace string
	}{
		{id: "111111111112", workspace: "workspace111"},
		{id: "111111111113", workspace: "workspace111"},
		{id: "111111111114", workspace: "workspace111"},
		{id: "333333333334", workspace: "workspace333"},
		{id: "444444444445", workspace: "workspace444"},
		{id: "444444444446", workspace: "workspace444"},
		{id: "444444444447", workspace: "workspace444"},
		{id: "511111111111", workspace: "identity1111"},
		{id: "522222222222", workspace: "identity1111"},
		{id: "611111111111", workspace: "consistent11"},
		{id: "622222222222", workspace: "consistent11"},
	} {
		_, err := database.Exec(ctx, `INSERT INTO connections
			(id, workspace, role, kms_encrypted_settings_key)
			VALUES ($1, $2, 'Source', $3)`, connection.id, connection.workspace, []byte{1})
		if err != nil {
			t.Fatal(err)
		}
	}
	return database
}

// TestDeletedConnectionMetricsRead verifies the synthetic deleted-connections
// scope, cascading of metric rows when a connection is deleted, snapshot-local
// residuals, carry-forward, and real zero values.
func TestDeletedConnectionMetricsRead(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	organization := identityMetricsTestOrganization(t, database)
	identities := Identities{metrics: &Metrics{db: database}}
	ctx := t.Context()
	const workspace = "D31111111111"
	const activeConnection = "D32222222222"
	const deletedConnection = "D33333333333"
	_, err := database.Exec(ctx, `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ($1, $2, 'metrics test', 'metrics test', 'Normal', $3, $3, $3)`,
		workspace, organization, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO connections (id, workspace, role, kms_encrypted_settings_key)
		VALUES
		($1, $3, 'Source', $4),
		($2, $3, 'Source', $4)`, activeConnection, deletedConnection, workspace, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_metrics
		(workspace, day, observed_at,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES
		($1, '2026-08-01', '09:00:00', 50, 100, 15),
		($1, '2026-08-03', '10:00:00', 70, 130, 21),
		($1, '2026-08-05', '11:00:00', 60, 120, 18)`, workspace)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_connection_metrics
		(connection, day,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES
		($1, '2026-08-01', 40, 80, 10),
		($2, '2026-08-01', 10, 20, 5),
		($1, '2026-08-03', 55, 100, 15),
		($2, '2026-08-03', 15, 30, 6),
		($1, '2026-08-05', 60, 120, 18),
		('333333333334', '2026-08-01', 1000, 1000, 1000)`,
		activeConnection, deletedConnection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, "DELETE FROM connections WHERE id = $1", deletedConnection); err != nil {
		t.Fatal(err)
	}
	hasDeletedMetrics, err := database.QueryExists(ctx,
		"SELECT FROM identity_connection_metrics WHERE connection = $1", deletedConnection)
	if err != nil {
		t.Fatal(err)
	}
	if hasDeletedMetrics {
		t.Fatal("expected deleted connection metrics to cascade")
	}
	if _, err := identities.MetricsPerDate(ctx, workspace,
		time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		new(deletedConnection)); !errors.Is(err, ErrConnectionNotFound) {
		t.Fatalf("expected an individually deleted connection to be unavailable, got %v", err)
	}

	start := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC)
	days, err := identities.MetricsPerDate(ctx, workspace, start, end, new("deleted"))
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityMetricDays(t, days, start, []identityMetricDayExpectation{
		{total: 30, anonymous: 10, recognized: 20},
		{total: 30, anonymous: 10, recognized: 20},
		{total: 45, anonymous: 15, recognized: 30},
		{total: 45, anonymous: 15, recognized: 30},
		{},
		{},
	})

}

// TestIdentitiesRefresh verifies pipeline selection, timestamped observations,
// zero persistence, behavior when the workspace disappears, and no persistence
// after a datastore error.
func TestIdentitiesRefresh(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	_, err := database.Exec(t.Context(), `UPDATE workspaces SET warehouse_name = 'PostgreSQL';
		UPDATE connections SET connector = 'dummy';
		UPDATE connections SET role = 'Destination' WHERE id = '111111111114';
		INSERT INTO pipelines
			(id, connection, target, event_type, transformation_language,
			matching_in, matching_out, update_on_duplicates, table_key)
		VALUES
			('722222222222', '111111111113', 'User', '', 'JavaScript', '', '', false, ''),
			('733333333333', '111111111112', 'Group', '', 'JavaScript', '', '', false, ''),
			('744444444444', '111111111114', 'User', '', 'JavaScript', '', '', false, ''),
			('711111111111', '111111111112', 'User', '', 'JavaScript', '', '', false, ''),
			('755555555555', '333333333334', 'User', '', 'JavaScript', '', '', false, '')`)
	if err != nil {
		t.Fatal(err)
	}
	key, err := kms.New(t.Context(), "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	applicationState, err := state.New(t.Context(), database, key, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	defer applicationState.Close(t.Context())

	observedAt := time.Date(2026, time.August, 9, 10, 0, 0, 123456789, time.UTC)
	fake := &identityDatastoreFake{
		available: true,
		counts: &warehouses.IdentityCounts{
			Anonymous:      map[string]int{"111111111112": 2},
			Recognized:     map[string]int{},
			WithoutProfile: map[string]int{"111111111112": 1},
		},
	}
	nowCalls := 0
	m := &Metrics{db: database, state: applicationState, datastore: fake}
	identities := Identities{
		metrics: m,
		now: func() time.Time {
			nowCalls++
			return observedAt
		},
	}
	fake.beforeCount = func() {
		if nowCalls != fake.calls {
			t.Errorf("expected one observed_at acquisition before CountIdentities call %d, got %d total acquisitions",
				fake.calls, nowCalls)
		}
	}
	pipelines := []string{"711111111111", "722222222222"}
	if err := identities.Refresh(t.Context(), "workspace111"); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected CountIdentities to be called once, got %d calls", fake.calls)
	}
	if nowCalls != 1 {
		t.Fatalf("expected observed_at to be acquired once, got %d acquisitions", nowCalls)
	}
	if fake.workspace != "workspace111" || !slices.Equal(fake.pipelines, pipelines) {
		t.Fatalf("expected workspace %q and pipelines %v, got workspace %q and pipelines %v",
			"workspace111", pipelines, fake.workspace, fake.pipelines)
	}
	expected := newIdentitySnapshot("workspace111", observedAt.Truncate(time.Microsecond), fake.counts)
	assertIdentitySnapshot(t, database, expected)

	fake.counts = &warehouses.IdentityCounts{
		Anonymous: map[string]int{}, Recognized: map[string]int{}, WithoutProfile: map[string]int{},
	}
	if err := identities.Refresh(t.Context(), "workspace222"); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 2 || nowCalls != 2 {
		t.Fatalf("expected two timestamped CountIdentities calls, got %d calls and %d timestamp acquisitions",
			fake.calls, nowCalls)
	}
	if len(fake.pipelines) != 0 {
		t.Fatalf("expected no selected pipelines, got %v", fake.pipelines)
	}
	zero := newIdentitySnapshot("workspace222", observedAt.Truncate(time.Microsecond), fake.counts)
	assertIdentitySnapshot(t, database, zero)

	fake.err = errors.New("warehouse unavailable")
	err = identities.Refresh(t.Context(), "workspace333")
	if err != nil {
		if !errors.Is(err, fake.err) {
			t.Fatalf("expected warehouse error %v, got %v", fake.err, err)
		}
	}
	if err == nil {
		t.Fatal("expected a warehouse error, got nil")
	}
	if fake.calls != 3 || nowCalls != 3 {
		t.Fatalf("expected three timestamped CountIdentities calls, got %d calls and %d timestamp acquisitions",
			fake.calls, nowCalls)
	}
	hasMetric, err := database.QueryExists(t.Context(),
		"SELECT FROM identity_metrics WHERE workspace = 'workspace333'")
	if err != nil {
		t.Fatal(err)
	}
	if hasMetric {
		t.Fatal("expected no metric after a warehouse error")
	}

	storeCalls, calls, acquisitions := fake.storeCalls, fake.calls, nowCalls
	err = identities.Refresh(t.Context(), "missing11111")
	if err != nil {
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected ErrWorkspaceNotFound for absent state, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected ErrWorkspaceNotFound for absent state, got nil")
	}
	if fake.storeCalls != storeCalls || fake.calls != calls || nowCalls != acquisitions {
		t.Fatal("expected absent state to return before accessing the datastore or acquiring a timestamp")
	}

	fake.available = false
	err = identities.Refresh(t.Context(), "workspace444")
	if err != nil {
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected ErrWorkspaceNotFound for absent datastore, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected ErrWorkspaceNotFound for absent datastore, got nil")
	}
	if fake.storeCalls != storeCalls+1 || fake.calls != calls || nowCalls != acquisitions {
		t.Fatal("expected absent datastore to return before counting identities or acquiring a timestamp")
	}

}

// TestIdentityConnectionMetricsPerDate verifies dense connection carry-forward
// history, zero boundaries established by complete workspace snapshots,
// never-seen connections, and unknown connection handling.
func TestIdentityConnectionMetricsPerDate(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	organization := identityMetricsTestOrganization(t, database)
	identities := Identities{metrics: &Metrics{db: database}}
	ctx := t.Context()
	const workspace = "6PaS2mM6JwN4"
	_, err := database.Exec(ctx, `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ($1, $2, 'metrics test', 'metrics test', 'Normal', $3, $3, $3)`,
		workspace, organization, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	const historicalConnection = "7QaT3mN7KxP5"
	const disappearingConnection = "8RbU4nP8LyQ6"
	const neverObservedConnection = "9ScV5oQ9MzR7"
	const unknownConnection = "ATdW6pRAm1S8"
	_, err = database.Exec(ctx, `INSERT INTO connections
		(id, workspace, role, kms_encrypted_settings_key)
		VALUES
		($1, $4, 'Source', $5),
		($2, $4, 'Source', $5),
		($3, $4, 'Source', $5)`, historicalConnection, disappearingConnection,
		neverObservedConnection, workspace, []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC)
	_, err = database.Exec(ctx, `INSERT INTO identity_metrics
		(workspace, day, observed_at,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES
		($1, '2026-08-01', '09:00:00', 100, 200, 0),
		($1, '2026-08-03', '10:00:00', 110, 210, 0),
		($1, '2026-08-05', '11:00:00', 120, 220, 0)`, workspace)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_connection_metrics
		(connection, day,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES
		($1, '2026-08-01', 10, 20, 1),
		($1, '2026-08-05', 15, 35, 2),
		($2, '2026-08-01', 7, 13, 1),
		('333333333334', '2026-08-01', 1000, 1000, 1000)`,
		historicalConnection, disappearingConnection)
	if err != nil {
		t.Fatal(err)
	}
	selection := historicalConnection
	days, err := identities.MetricsPerDate(ctx, workspace, start, end, &selection)
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityMetricDays(t, days, start, []identityMetricDayExpectation{
		{total: 30, anonymous: 10, recognized: 20},
		{total: 30, anonymous: 10, recognized: 20},
		{},
		{},
		{total: 50, anonymous: 15, recognized: 35},
		{total: 50, anonymous: 15, recognized: 35},
	})

	seededStart := start.AddDate(0, 0, 1)
	seeded, err := identities.MetricsPerDate(ctx, workspace, seededStart,
		seededStart.AddDate(0, 0, 3), &selection)
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityMetricDays(t, seeded, seededStart, []identityMetricDayExpectation{
		{total: 30, anonymous: 10, recognized: 20},
		{},
		{},
	})

	selection = disappearingConnection
	disappeared, err := identities.MetricsPerDate(ctx, workspace, start, end, &selection)
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityMetricDays(t, disappeared, start, []identityMetricDayExpectation{
		{total: 20, anonymous: 7, recognized: 13},
		{total: 20, anonymous: 7, recognized: 13},
		{},
		{},
		{},
		{},
	})

	selection = neverObservedConnection
	neverObserved, err := identities.MetricsPerDate(ctx, workspace, start, end, &selection)
	if err != nil {
		t.Fatal(err)
	}
	if neverObserved == nil {
		t.Fatal("expected an empty identity metric slice")
	}
	assertIdentityMetricDays(t, neverObserved, start, nil)
	selection = historicalConnection
	beforeStart := start.AddDate(0, 0, -1)
	before, err := identities.MetricsPerDate(ctx, workspace, beforeStart, start, &selection)
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityMetricDays(t, before, beforeStart, nil)

	selection = unknownConnection
	if _, err := identities.MetricsPerDate(ctx, workspace, start, end,
		&selection); !errors.Is(err, ErrConnectionNotFound) {
		t.Fatalf("expected ErrConnectionNotFound, got %v", err)
	}
	selection = "333333333334"
	if _, err := identities.MetricsPerDate(ctx, workspace, start, end,
		&selection); !errors.Is(err, ErrConnectionNotFound) {
		t.Fatalf("expected another workspace's connection to be unavailable, got %v", err)
	}

}

// TestIdentityMetricsPerDateAndLatest verifies latest mapping, connection
// loading, dense carry-forward history, zero semantics, and missing-workspace
// handling.
func TestIdentityMetricsPerDateAndLatest(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	identities := Identities{metrics: &Metrics{db: database}}
	ctx := t.Context()
	workspace := "identity1111"
	start := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)
	latestDay := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	_, err := database.Exec(ctx, `INSERT INTO identity_metrics
		(workspace, day, observed_at,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES
		($1, '2026-08-02', '12:00:00', 40, 60, 25),
		($1, '2026-08-04', '18:00:00', 0, 0, 0),
		($1, $2, '09:30:00', 120, 80, 50)`, workspace, latestDay)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_connection_metrics
		(connection, day,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES
		('522222222222', $1, 20, 30, 10),
		('511111111111', $1, 100, 50, 40),
		('333333333334', $1, 1000, 1000, 1000)`, latestDay)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO connections
		(id, workspace, role, kms_encrypted_settings_key)
		VALUES ('533333333333', $1, 'Source', $2)`, workspace, []byte{1})
	if err != nil {
		t.Fatal(err)
	}

	latest, err := identities.Latest(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Total != 200 || latest.Anonymous != 120 ||
		latest.Recognized != 80 || latest.WithoutProfile != 50 {
		t.Fatalf("expected latest counts 200/120/80/50, got %d/%d/%d/%d",
			latest.Total, latest.Anonymous, latest.Recognized, latest.WithoutProfile)
	}
	if latest.ObservedAt != time.Date(2026, time.August, 10, 9, 30, 0, 0, time.UTC) {
		t.Fatalf("expected latest observed_at %s, got %s",
			time.Date(2026, time.August, 10, 9, 30, 0, 0, time.UTC), latest.ObservedAt)
	}
	wantConnections := []IdentityConnectionMetric{
		{Connection: "511111111111", Anonymous: 100, Recognized: 50, WithoutProfile: 40},
		{Connection: "522222222222", Anonymous: 20, Recognized: 30, WithoutProfile: 10},
		{Connection: "533333333333"},
	}
	if len(latest.Connections) != len(wantConnections) {
		t.Fatalf("expected %d connections, got %d: %#v",
			len(wantConnections), len(latest.Connections), latest.Connections)
	}
	for index := range wantConnections {
		if latest.Connections[index] != wantConnections[index] {
			t.Fatalf("expected connection %#v, got %#v",
				wantConnections[index], latest.Connections[index])
		}
	}

	days, err := identities.MetricsPerDate(ctx, workspace, start, end, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityMetricDays(t, days, start.AddDate(0, 0, 1), []identityMetricDayExpectation{
		{total: 100, anonymous: 40, recognized: 60},
		{total: 100, anonymous: 40, recognized: 60},
		{},
		{},
	})
	seedStart := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC)
	seeded, err := identities.MetricsPerDate(ctx, workspace, seedStart,
		seedStart.AddDate(0, 0, 2), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityMetricDays(t, seeded, seedStart, []identityMetricDayExpectation{
		{total: 100, anonymous: 40, recognized: 60},
		{},
	})

	emptyWorkspace := "emptyconn111"
	_, err = database.Exec(ctx, `INSERT INTO identity_metrics
		(workspace, day, observed_at,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES ($1, $2, '10:00:00', 0, 0, 0)`, emptyWorkspace, latestDay)
	if err != nil {
		t.Fatal(err)
	}
	emptyLatest, err := identities.Latest(ctx, emptyWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if emptyLatest.Connections == nil || len(emptyLatest.Connections) != 0 {
		t.Fatalf("expected a non-nil empty connection list, got %#v", emptyLatest.Connections)
	}

	if _, err := database.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", emptyWorkspace); err != nil {
		t.Fatal(err)
	}
	_, err = identities.Latest(ctx, emptyWorkspace)
	if err != nil {
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected ErrWorkspaceNotFound after deleting the workspace, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected ErrWorkspaceNotFound after deleting the workspace, got nil")
	}

	if _, err := database.Exec(ctx, "DROP TABLE identity_connection_metrics"); err != nil {
		t.Fatal(err)
	}
	if _, err := identities.MetricsPerDate(ctx, workspace, start, end, nil); err != nil {
		t.Fatalf("expected days not to query the dropped connection table, got %v", err)
	}

}

// TestLatestIdentityMetricUsesOneStatementSnapshot verifies that a newer
// identity snapshot committed while Latest is running cannot make its workspace
// and connection values observe different states.
func TestLatestIdentityMetricUsesOneStatementSnapshot(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	identities := Identities{metrics: &Metrics{db: database}}
	ctx := t.Context()
	const workspace = "consistent11"
	observedAt := time.Date(2026, time.August, 9, 10, 0, 0, 0, time.UTC)
	oldSnapshot := newIdentitySnapshot(workspace, observedAt, &warehouses.IdentityCounts{
		Anonymous: map[string]int{"611111111111": 1},
	})
	newSnapshot := newIdentitySnapshot(workspace, observedAt.Add(time.Hour), &warehouses.IdentityCounts{
		Recognized: map[string]int{"622222222222": 2},
	})
	newSnapshotDay := newSnapshot.observedAt.Truncate(24 * time.Hour)
	if err := identities.storeSnapshot(ctx, oldSnapshot); err != nil {
		t.Fatal(err)
	}

	// Block the workspace read after its statement snapshot has been established,
	// so the concurrent commit occurs before the connection read starts.
	const advisoryLock int64 = 8_132_041
	if _, err := database.Exec(ctx,
		"ALTER TABLE identity_metrics RENAME TO identity_metrics_data"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `CREATE FUNCTION wait_for_identity_metrics_test(bigint)
		RETURNS bigint
		LANGUAGE SQL VOLATILE
		AS 'SELECT $1 FROM pg_advisory_xact_lock_shared(8132041)'`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `CREATE VIEW identity_metrics AS
		SELECT workspace, day, observed_at,
			CASE WHEN workspace = 'consistent11'
				THEN wait_for_identity_metrics_test(identities_anonymous)
				ELSE identities_anonymous
			END AS identities_anonymous,
			identities_recognized, identities_without_profile
		FROM identity_metrics_data`); err != nil {
		t.Fatal(err)
	}
	release := blockAdvisoryLock(t, database, advisoryLock)

	reads := make(chan struct {
		metric IdentityMetric
		err    error
	}, 1)
	go func() {
		result, err := identities.Latest(ctx, workspace)
		reads <- struct {
			metric IdentityMetric
			err    error
		}{metric: result, err: err}
	}()
	awaitDatabaseLock(t, database, "advisory", "identity_metrics")

	err := database.Transaction(ctx, func(tx *db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE identity_metrics_data
			SET observed_at = $3, identities_anonymous = $4,
				identities_recognized = $5, identities_without_profile = $6
			WHERE workspace = $1 AND day = $2`,
			newSnapshot.workspace, newSnapshotDay, newSnapshot.observedAt.Format("15:04:05.999999"),
			newSnapshot.anonymous, newSnapshot.recognized, newSnapshot.withoutProfile)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM identity_connection_metrics
			WHERE connection = $1 AND day = $2`,
			oldSnapshot.connections[0].id, newSnapshotDay); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO identity_connection_metrics
			(connection, day,
			identities_anonymous, identities_recognized, identities_without_profile)
			VALUES ($1, $2, $3, $4, $5)`,
			newSnapshot.connections[0].id, newSnapshotDay,
			newSnapshot.connections[0].anonymous, newSnapshot.connections[0].recognized,
			newSnapshot.connections[0].withoutProfile)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	release()

	var read struct {
		metric IdentityMetric
		err    error
	}
	select {
	case read = <-reads:
	case <-time.After(5 * time.Second):
		t.Fatal("Latest did not return after releasing the workspace read")
	}
	if read.err != nil {
		t.Fatal(read.err)
	}
	older := read.metric
	if older.Total != 1 || len(older.Connections) != 2 ||
		older.Connections[0] != (IdentityConnectionMetric{Connection: "611111111111", Anonymous: 1}) ||
		older.Connections[1] != (IdentityConnectionMetric{Connection: "622222222222"}) {
		t.Fatalf("expected the complete older snapshot, got %#v", older)
	}

	newer, err := identities.Latest(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if newer.Total != 2 || len(newer.Connections) != 2 ||
		newer.Connections[0] != (IdentityConnectionMetric{Connection: "611111111111"}) ||
		newer.Connections[1] != (IdentityConnectionMetric{Connection: "622222222222", Recognized: 2}) {
		t.Fatalf("expected the subsequently visible newer snapshot, got %#v", newer)
	}

}

// TestIdentityMetricsStoreSnapshot verifies sparse normalization, monotonic
// replacement of workspace and connection metrics, omission of metrics for
// unavailable connections, including connections from other workspaces and
// connections deleted concurrently, zero observations, and persistence on
// separate days.
func TestIdentityMetricsStoreSnapshot(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	identities := Identities{metrics: &Metrics{db: database}}
	observedAt := time.Date(2026, time.August, 9, 10, 0, 0, 123456000, time.UTC)
	counts := &warehouses.IdentityCounts{
		Anonymous:      map[string]int{"111111111112": 2, "111111111113": 1},
		Recognized:     map[string]int{"111111111112": 3},
		WithoutProfile: map[string]int{"111111111113": 1},
	}
	snapshot := newIdentitySnapshot("workspace111", observedAt, counts)
	if snapshot.anonymous != 3 || snapshot.recognized != 3 || snapshot.withoutProfile != 1 {
		t.Fatalf("expected workspace counts 3/3/1, got %d/%d/%d",
			snapshot.anonymous, snapshot.recognized, snapshot.withoutProfile)
	}
	if len(snapshot.connections) != 2 || snapshot.connections[0] != (identityConnection{
		id: "111111111112", anonymous: 2, recognized: 3,
	}) || snapshot.connections[1] != (identityConnection{
		id: "111111111113", anonymous: 1, withoutProfile: 1,
	}) {
		t.Fatalf("expected normalized sparse connection counts, got %#v", snapshot.connections)
	}
	if err := identities.storeSnapshot(t.Context(), snapshot); err != nil {
		t.Fatal(err)
	}
	assertIdentitySnapshot(t, database, snapshot)

	newerCounts := &warehouses.IdentityCounts{
		Anonymous:      map[string]int{"111111111114": 4},
		Recognized:     map[string]int{"111111111114": 2},
		WithoutProfile: map[string]int{"111111111114": 3},
	}
	newer := newIdentitySnapshot("workspace111", observedAt.Add(time.Hour), newerCounts)
	if err := identities.storeSnapshot(t.Context(), newer); err != nil {
		t.Fatal(err)
	}
	if err := identities.storeSnapshot(t.Context(), snapshot); err != nil {
		t.Fatal(err)
	}
	assertIdentitySnapshot(t, database, newer)

	unavailable := newer
	unavailable.observedAt = newer.observedAt.Add(time.Hour)
	unavailable.anonymous = 9
	unavailable.connections = []identityConnection{{id: "555555555555", anonymous: 9}}
	if err := identities.storeSnapshot(t.Context(), unavailable); err != nil {
		t.Fatal(err)
	}
	unavailable.connections = nil
	assertIdentitySnapshot(t, database, unavailable)

	otherWorkspace := unavailable
	otherWorkspace.observedAt = unavailable.observedAt.Add(time.Hour)
	otherWorkspace.anonymous = 8
	otherWorkspace.connections = []identityConnection{{id: "333333333334", anonymous: 8}}
	if err := identities.storeSnapshot(t.Context(), otherWorkspace); err != nil {
		t.Fatal(err)
	}
	otherWorkspace.connections = nil
	assertIdentitySnapshot(t, database, otherWorkspace)
	var stored bool
	if err := database.QueryRow(t.Context(), `SELECT EXISTS (
		SELECT FROM identity_connection_metrics WHERE connection = $1 AND day = $2
	)`, "333333333334", otherWorkspace.observedAt.Truncate(24*time.Hour)).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored {
		t.Fatal("expected a connection from another workspace to be omitted")
	}

	nextDay := newIdentitySnapshot("workspace111", observedAt.Add(24*time.Hour), newerCounts)
	if err := identities.storeSnapshot(t.Context(), nextDay); err != nil {
		t.Fatal(err)
	}
	assertIdentitySnapshot(t, database, nextDay)

	var days int
	if err := database.QueryRow(t.Context(), `SELECT COUNT(*) FROM identity_metrics
		WHERE workspace = 'workspace111'`).Scan(&days); err != nil {
		t.Fatal(err)
	}
	if days != 2 {
		t.Fatalf("expected 2 daily identity snapshots, got %d", days)
	}

	zero := newIdentitySnapshot("workspace222", observedAt, &warehouses.IdentityCounts{
		Anonymous: map[string]int{}, Recognized: map[string]int{}, WithoutProfile: map[string]int{},
	})
	if err := identities.storeSnapshot(t.Context(), zero); err != nil {
		t.Fatal(err)
	}
	assertIdentitySnapshot(t, database, zero)

	explicitZero := newIdentitySnapshot("workspace333", observedAt, &warehouses.IdentityCounts{
		Anonymous: map[string]int{"333333333334": 0}, Recognized: map[string]int{}, WithoutProfile: map[string]int{},
	})
	if len(explicitZero.connections) != 1 || explicitZero.connections[0] != (identityConnection{id: "333333333334"}) {
		t.Fatalf("expected one explicit zero connection, got %#v", explicitZero.connections)
	}
	if err := identities.storeSnapshot(t.Context(), explicitZero); err != nil {
		t.Fatal(err)
	}
	assertIdentitySnapshot(t, database, explicitZero)

	deleting := newIdentitySnapshot("workspace333", observedAt.Add(24*time.Hour), &warehouses.IdentityCounts{
		Anonymous: map[string]int{"333333333334": 5}, Recognized: map[string]int{}, WithoutProfile: map[string]int{},
	})
	deleteReady := make(chan error, 1)
	deleteErrors := make(chan error, 1)
	releaseDelete := make(chan struct{})
	var releaseDeleteOnce sync.Once
	release := func() {
		releaseDeleteOnce.Do(func() {
			close(releaseDelete)
		})
	}
	t.Cleanup(release)
	go func() {

		deleteErrors <- database.Transaction(t.Context(), func(tx *db.Tx) error {

			_, err := tx.Exec(t.Context(), "DELETE FROM connections WHERE id = '333333333334'")
			deleteReady <- err
			if err != nil {
				return err
			}
			<-releaseDelete

			return nil
		})

	}()
	if err := <-deleteReady; err != nil {
		t.Fatal(err)
	}
	storeErrors := make(chan error, 1)
	go func() {
		storeErrors <- identities.storeSnapshot(t.Context(), deleting)
	}()
	awaitDatabaseLock(t, database, "transactionid", "INSERT INTO identity_connection_metrics")
	release()
	if err := <-deleteErrors; err != nil {
		t.Fatal(err)
	}
	if err := <-storeErrors; err != nil {
		t.Fatal(err)
	}
	deleting.connections = nil
	assertIdentitySnapshot(t, database, deleting)

	concurrentOlderCounts := &warehouses.IdentityCounts{
		Anonymous:      map[string]int{"444444444445": 2, "444444444446": 1},
		Recognized:     map[string]int{"444444444445": 3},
		WithoutProfile: map[string]int{"444444444446": 1},
	}
	concurrentNewerCounts := &warehouses.IdentityCounts{
		Anonymous:      map[string]int{"444444444447": 4},
		Recognized:     map[string]int{"444444444447": 2},
		WithoutProfile: map[string]int{"444444444447": 3},
	}
	concurrentOlder := newIdentitySnapshot("workspace444", observedAt, concurrentOlderCounts)
	concurrentNewer := newIdentitySnapshot("workspace444", observedAt.Add(time.Hour), concurrentNewerCounts)
	start := make(chan struct{})
	errors := make(chan error, 2)
	var wait sync.WaitGroup
	for _, concurrent := range []identitySnapshot{concurrentOlder, concurrentNewer} {
		wait.Go(func() {
			<-start
			errors <- identities.storeSnapshot(t.Context(), concurrent)
		})
	}
	close(start)
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertIdentitySnapshot(t, database, concurrentNewer)

}
