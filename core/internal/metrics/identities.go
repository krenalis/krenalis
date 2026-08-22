// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

var (
	// ErrConnectionNotFound indicates that a connection does not exist
	// or does not belong to the specified workspace.
	ErrConnectionNotFound = errors.New("connection not found in workspace")

	// ErrWorkspaceNotFound indicates that a workspace does not exist.
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

// Identities refreshes and stores identity metrics.
type Identities struct {
	metrics *Metrics
	now     func() time.Time
}

// IdentityMetric contains the latest persisted workspace identity state.
type IdentityMetric struct {
	ObservedAt time.Time // observation time in UTC

	Total          int // total identities
	Anonymous      int // anonymous identities
	Recognized     int // recognized identities
	WithoutProfile int // identities without a profile

	// Connections is a non-nil slice containing the live Source connections for
	// the workspace, ordered by connection identifier. Connections without a
	// metric for the latest workspace observation are represented with zero counts.
	Connections []IdentityConnectionMetric
}

// IdentityConnectionMetric contains identity counts for one connection.
type IdentityConnectionMetric struct {
	Connection     string // connection identifier
	Anonymous      int    // anonymous identities
	Recognized     int    // recognized identities
	WithoutProfile int    // identities without a profile
}

// IdentityMetricDay contains a known identity state for one UTC day.
type IdentityMetricDay struct {
	Day        time.Time // UTC day
	Total      int       // total identities
	Anonymous  int       // anonymous identities
	Recognized int       // recognized identities
}

// Latest returns the latest persisted identity state for a workspace.
// If the workspace does not exist, it returns [ErrWorkspaceNotFound].
func (identities *Identities) Latest(ctx context.Context, workspace string) (IdentityMetric, error) {

	latest := IdentityMetric{
		Connections: make([]IdentityConnectionMetric, 0),
	}

	rows, err := identities.metrics.db.Query(ctx, `WITH latest AS (
		SELECT workspace, day, observed_at,
			identities_anonymous, identities_recognized, identities_without_profile
		FROM identity_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1
	)
	SELECT (l.day + l.observed_at) AT TIME ZONE 'UTC',
		l.identities_anonymous, l.identities_recognized, l.identities_without_profile,
		live.id,
		COALESCE(c.identities_anonymous, 0),
		COALESCE(c.identities_recognized, 0),
		COALESCE(c.identities_without_profile, 0)
	FROM latest AS l
	LEFT JOIN connections AS live
		ON live.workspace = l.workspace AND live.role = 'Source'
	LEFT JOIN identity_connection_metrics AS c
		ON c.connection = live.id AND c.day = l.day
	ORDER BY live.id`, workspace)
	if err != nil {
		return IdentityMetric{}, err
	}
	defer rows.Close()

	hasObservation := false

	for rows.Next() {
		var connection *string
		var metric IdentityConnectionMetric
		if err := rows.Scan(&latest.ObservedAt, &latest.Anonymous, &latest.Recognized, &latest.WithoutProfile,
			&connection, &metric.Anonymous, &metric.Recognized, &metric.WithoutProfile); err != nil {
			return IdentityMetric{}, err
		}
		hasObservation = true
		if connection == nil {
			continue
		}
		metric.Connection = *connection
		latest.Connections = append(latest.Connections, metric)
	}
	if err := rows.Err(); err != nil {
		return IdentityMetric{}, err
	}
	if !hasObservation {
		return IdentityMetric{}, ErrWorkspaceNotFound
	}
	latest.ObservedAt = latest.ObservedAt.UTC()
	latest.Total = latest.Anonymous + latest.Recognized

	return latest, nil
}

const (
	// deletedConnectionMetricsPerDateQuery reads the daily metrics attributable
	// to connections that no longer exist.
	deletedConnectionMetricsPerDateQuery = `WITH seed AS (
	SELECT workspace, day, identities_anonymous, identities_recognized
	FROM identity_metrics
	WHERE workspace = $1 AND day < $2
	ORDER BY day DESC
	LIMIT 1
), range_rows AS (
	SELECT workspace, day, identities_anonymous, identities_recognized
	FROM identity_metrics
	WHERE workspace = $1 AND day >= $2 AND day < $3
), parent_rows AS (
	SELECT * FROM seed
	UNION ALL
	SELECT * FROM range_rows
)
SELECT p.day,
	p.identities_anonymous - COALESCE(SUM(c.identities_anonymous), 0),
	p.identities_recognized - COALESCE(SUM(c.identities_recognized), 0),
	TRUE
FROM parent_rows AS p
LEFT JOIN connections AS live ON live.workspace = p.workspace
LEFT JOIN identity_connection_metrics AS c
	ON c.connection = live.id AND c.day = p.day
GROUP BY p.workspace, p.day, p.identities_anonymous, p.identities_recognized
ORDER BY p.day`
	// identityConnectionMetricsPerDateQuery reads the daily identity state for
	// one connection. The final column indicates whether the selected
	// connection belongs to the workspace.
	identityConnectionMetricsPerDateQuery = `WITH selected_connection AS (
	SELECT id
	FROM connections
	WHERE workspace = $1 AND id = $2
), seed AS (
	SELECT day
	FROM identity_metrics
	WHERE workspace = $1 AND day < $3
	ORDER BY day DESC
	LIMIT 1
), range_rows AS (
	SELECT day
	FROM identity_metrics
	WHERE workspace = $1 AND day >= $3 AND day < $4
), parent_rows AS (
	SELECT * FROM seed
	UNION ALL
	SELECT * FROM range_rows
), first_observation AS (
	SELECT MIN(day) AS day
	FROM identity_connection_metrics
	WHERE connection = $2
)
SELECT p.day,
	COALESCE(c.identities_anonymous, 0),
	COALESCE(c.identities_recognized, 0),
	s.id IS NOT NULL
FROM (VALUES (TRUE)) AS request(single)
LEFT JOIN selected_connection AS s ON TRUE
LEFT JOIN first_observation AS f ON s.id IS NOT NULL
LEFT JOIN parent_rows AS p ON f.day <= p.day
LEFT JOIN identity_connection_metrics AS c
	ON c.connection = s.id AND c.day = p.day
ORDER BY p.day`
	// identityMetricsPerDateQuery reads the workspace's daily identity state.
	identityMetricsPerDateQuery = `WITH seed AS (
	SELECT day, identities_anonymous, identities_recognized
	FROM identity_metrics
	WHERE workspace = $1 AND day < $2
	ORDER BY day DESC
	LIMIT 1
), range_rows AS (
	SELECT day, identities_anonymous, identities_recognized
	FROM identity_metrics
	WHERE workspace = $1 AND day >= $2 AND day < $3
)
SELECT day, identities_anonymous, identities_recognized, TRUE
FROM seed
UNION ALL
SELECT day, identities_anonymous, identities_recognized, TRUE
FROM range_rows
ORDER BY day`
)

// MetricsPerDate returns daily identity metrics over the interval [start,end).
// Start and end must be UTC day boundaries, and end must be after start.
//
// The returned metrics are scoped according to connectionSelection:
//   - nil selects the entire workspace;
//   - a connection identifier selects that connection within the workspace;
//   - "deleted" selects the aggregate contribution of deleted connections.
//
// The daily series is carried forward from the latest observation before start,
// when available. Otherwise, it begins with the first observation in the
// requested interval. A valid selection with no observable state returns an
// empty series.
//
// MetricsPerDate returns ErrConnectionNotFound when a selected connection does
// not belong to the specified workspace.
func (identities *Identities) MetricsPerDate(ctx context.Context, workspace string, start, end time.Time, connectionSelection *string) ([]IdentityMetricDay, error) {

	var query string
	var args []any
	if connectionSelection == nil {
		query = identityMetricsPerDateQuery
		args = []any{workspace, start, end}
	} else if *connectionSelection == "deleted" {
		query = deletedConnectionMetricsPerDateQuery
		args = []any{workspace, start, end}
	} else {
		query = identityConnectionMetricsPerDateQuery
		args = []any{workspace, *connectionSelection, start, end}
	}

	rows, err := identities.metrics.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dayCount := int(end.Sub(start) / (24 * time.Hour))
	updates := map[int]identityMetricState{}
	var seed identityMetricState
	hasSeed := false

	for rows.Next() {
		var day *time.Time
		var s identityMetricState
		var found bool
		if err := rows.Scan(&day, &s.anonymous, &s.recognized, &found); err != nil {
			return nil, err
		}
		if !found {
			return nil, ErrConnectionNotFound
		}
		// A NULL day represents a valid selection without an observable state.
		if day == nil {
			continue
		}
		metricDay := day.UTC()
		if metricDay.Before(start) {
			seed = s
			hasSeed = true
			continue
		}
		i := int(metricDay.Sub(start) / (24 * time.Hour))
		if i < 0 || i >= dayCount {
			return nil, fmt.Errorf("metrics: identity metrics contain a day outside the requested interval")
		}
		updates[i] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	days := make([]IdentityMetricDay, 0, dayCount)
	current := seed
	hasCurrent := hasSeed
	for i := range dayCount {
		if s, ok := updates[i]; ok {
			current = s
			hasCurrent = true
		}
		if !hasCurrent {
			continue
		}
		days = append(days, IdentityMetricDay{
			Day:        start.AddDate(0, 0, i),
			Total:      current.anonymous + current.recognized,
			Anonymous:  current.anonymous,
			Recognized: current.recognized,
		})
	}

	return days, nil
}

// Refresh observes and stores identity metrics for a workspace.
//
// If the workspace does not exist, it returns [ErrWorkspaceNotFound].
// If the data warehouse is in maintenance mode, it returns the
// datastore.ErrMaintenanceMode error. If an error occurs with the data
// warehouse, it returns an *datastore.UnavailableError error.
func (identities *Identities) Refresh(ctx context.Context, workspace string) error {

	ws, ok := identities.metrics.state.Workspace(workspace)
	if !ok {
		return ErrWorkspaceNotFound
	}
	var pipelines []string
	for _, connection := range ws.Connections() {
		if connection.Role != state.Source {
			continue
		}
		for _, pipeline := range connection.Pipelines() {
			if pipeline.Target == state.TargetUser {
				pipelines = append(pipelines, pipeline.ID)
			}
		}
	}
	slices.Sort(pipelines)
	store, ok := identities.metrics.datastore.Store(workspace)
	if !ok {
		return ErrWorkspaceNotFound
	}
	observedAt := identities.now().UTC().Truncate(time.Microsecond)
	counts, err := store.CountIdentities(ctx, pipelines)
	if err != nil {
		return err
	}

	return identities.storeSnapshot(ctx, newIdentitySnapshot(workspace, observedAt, counts))
}

// storeSnapshot stores a workspace identity snapshot and metrics for its
// existing connections when it is newer than the persisted snapshot for the
// same workspace and UTC day.
//
// If the workspace does not exist, it returns [ErrWorkspaceNotFound].
func (identities *Identities) storeSnapshot(ctx context.Context, snapshot identitySnapshot) error {

	day := snapshot.observedAt.Truncate(24 * time.Hour)
	err := identities.metrics.db.Transaction(ctx, func(tx *db.Tx) error {

		var accepted bool
		err := tx.QueryRow(ctx, `INSERT INTO identity_metrics AS m
			(workspace, day, observed_at, identities_anonymous, identities_recognized, identities_without_profile)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (workspace, day) DO UPDATE SET
			observed_at = EXCLUDED.observed_at,
			identities_anonymous = EXCLUDED.identities_anonymous,
			identities_recognized = EXCLUDED.identities_recognized,
			identities_without_profile = EXCLUDED.identities_without_profile
			WHERE EXCLUDED.observed_at > m.observed_at
			RETURNING true`, snapshot.workspace, day, snapshot.observedAt.Format("15:04:05.999999"),
			snapshot.anonymous, snapshot.recognized, snapshot.withoutProfile).Scan(&accepted)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "identity_metrics_workspace_fkey" {
				return ErrWorkspaceNotFound
			}
			return err
		}
		if !accepted {
			return nil
		}

		_, err = tx.Exec(ctx, `DELETE FROM identity_connection_metrics AS m
			USING connections AS c
			WHERE m.connection = c.id AND c.workspace = $1 AND m.day = $2`,
			snapshot.workspace, day)
		if err != nil {
			return err
		}

		// Insert metrics for connections that still exist.
		if len(snapshot.connections) > 0 {
			var query strings.Builder
			query.WriteString(`INSERT INTO identity_connection_metrics
				(connection, day, identities_anonymous, identities_recognized, identities_without_profile)
				SELECT o.* FROM (VALUES `)
			args := make([]any, 0, len(snapshot.connections)*5+1)
			for i, c := range snapshot.connections {
				if i > 0 {
					query.WriteByte(',')
				}
				base := i*5 + 1
				_, _ = fmt.Fprintf(&query, "($%d::varchar,$%d::date,$%d::bigint,$%d::bigint,$%d::bigint)",
					base, base+1, base+2, base+3, base+4)
				args = append(args, c.id, day, c.anonymous, c.recognized, c.withoutProfile)
			}
			query.WriteString(`) AS o
				(connection, day, identities_anonymous, identities_recognized, identities_without_profile)
				JOIN connections AS c ON c.id = o.connection AND c.workspace = $`)
			query.WriteString(strconv.Itoa(len(args) + 1))
			query.WriteString("\n\t\t\t\tFOR KEY SHARE OF c")
			args = append(args, snapshot.workspace)
			_, err = tx.Exec(ctx, query.String(), args...)
		}

		return err
	})

	return err
}

// identityConnection contains normalized identity counts for one connection.
type identityConnection struct {
	id             string // connection identifier
	anonymous      int    // anonymous identities
	recognized     int    // recognized identities
	withoutProfile int    // identities without a profile
}

// identityMetricState contains identity counts carried between daily
// observations.
type identityMetricState struct {
	anonymous  int
	recognized int
}

// identitySnapshot contains one full workspace identity observation.
type identitySnapshot struct {
	workspace      string               // workspace identifier
	observedAt     time.Time            // observation time in UTC
	anonymous      int                  // anonymous identities
	recognized     int                  // recognized identities
	withoutProfile int                  // identities without a profile
	connections    []identityConnection // connection metrics
}

// newIdentitySnapshot constructs a full identity snapshot by normalizing
// prevalidated sparse identity counts.
func newIdentitySnapshot(workspace string, observedAt time.Time, counts *warehouses.IdentityCounts) identitySnapshot {

	var connectionIDs []string
	for _, m := range [...]map[string]int{
		counts.Anonymous,
		counts.Recognized,
		counts.WithoutProfile,
	} {
		for connection := range m {
			connectionIDs = append(connectionIDs, connection)
		}
	}
	slices.Sort(connectionIDs)
	connectionIDs = slices.Compact(connectionIDs)

	snapshot := identitySnapshot{
		workspace:   workspace,
		observedAt:  observedAt,
		connections: make([]identityConnection, len(connectionIDs)),
	}
	for i, id := range connectionIDs {
		connection := identityConnection{
			id:             id,
			anonymous:      counts.Anonymous[id],
			recognized:     counts.Recognized[id],
			withoutProfile: counts.WithoutProfile[id],
		}
		snapshot.connections[i] = connection
		snapshot.anonymous += connection.anonymous
		snapshot.recognized += connection.recognized
		snapshot.withoutProfile += connection.withoutProfile
	}

	return snapshot
}
