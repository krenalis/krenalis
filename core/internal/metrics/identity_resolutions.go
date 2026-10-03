// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// IdentityResolutions stores metrics for successful Identity Resolution runs.
type IdentityResolutions struct {
	metrics *Metrics
}

// IdentityResolutionMetric contains metrics for one successful Identity
// Resolution.
type IdentityResolutionMetric struct {
	ObservedAt time.Time

	Identities IdentityResolutionMetricCounts
	Profiles   IdentityResolutionMetricCounts

	Composition IdentityResolutionComposition

	IdentitiesPerProfile float64

	// LinkedIdentitiesRate is the share of identities that belong to a profile
	// containing more than one identity.
	LinkedIdentitiesRate float64
}

// IdentityResolutionMetricCounts contains anonymous, recognized, and total
// counts for one successful Identity Resolution.
type IdentityResolutionMetricCounts struct {
	Total      int
	Anonymous  int
	Recognized int
}

// IdentityResolutionComposition contains the profile-size distribution buckets.
type IdentityResolutionComposition struct {
	One            int
	Two            int
	Three          int
	FourToTen      int
	ElevenToTwenty int
	MoreThanTwenty int
}

// IdentityResolutionMetricDay contains the successful Identity Resolution state
// for one UTC day.
type IdentityResolutionMetricDay struct {
	Day time.Time

	Identities         int
	Profiles           int
	ProfilesAnonymous  int
	ProfilesRecognized int

	IdentitiesPerProfile float64

	// LinkedIdentitiesRate is the share of identities that belong to a profile
	// containing more than one identity.
	LinkedIdentitiesRate float64
}

// identityResolutionHistoryState contains the values needed to construct one
// historical Identity Resolution state.
type identityResolutionHistoryState struct {
	identitiesAnonymous  int
	identitiesRecognized int
	profilesAnonymous    int
	profilesRecognized   int
	profilesWithOne      int
}

// Latest returns the latest successful Identity Resolution metric for a
// workspace. It returns nil when no successful Identity Resolution exists.
//
// The caller must provide a valid workspace identifier.
func (resolutions *IdentityResolutions) Latest(ctx context.Context, workspace string) (*IdentityResolutionMetric, error) {

	var latest IdentityResolutionMetric

	err := resolutions.metrics.db.QueryRow(ctx, `SELECT (day + observed_at) AT TIME ZONE 'UTC',
		profiles_anonymous, profiles_recognized,
		identities_anonymous, identities_recognized, composition_one, composition_two, composition_three,
		composition_four_to_ten, composition_eleven_to_twenty, composition_more_than_twenty
		FROM identity_resolution_metrics WHERE workspace = $1
		ORDER BY day DESC LIMIT 1`, workspace).Scan(
		&latest.ObservedAt, &latest.Profiles.Anonymous, &latest.Profiles.Recognized,
		&latest.Identities.Anonymous, &latest.Identities.Recognized,
		&latest.Composition.One, &latest.Composition.Two, &latest.Composition.Three,
		&latest.Composition.FourToTen, &latest.Composition.ElevenToTwenty, &latest.Composition.MoreThanTwenty)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	latest.ObservedAt = latest.ObservedAt.UTC()
	latest.Identities.Total = latest.Identities.Anonymous + latest.Identities.Recognized
	latest.Profiles.Total = latest.Profiles.Anonymous + latest.Profiles.Recognized
	if latest.Profiles.Total != 0 {
		latest.IdentitiesPerProfile = float64(latest.Identities.Total) / float64(latest.Profiles.Total)
	}
	if latest.Identities.Total != 0 {
		latest.LinkedIdentitiesRate =
			float64(latest.Identities.Total-latest.Composition.One) / float64(latest.Identities.Total)
	}

	return &latest, nil
}

// MetricsPerDate returns daily successful Identity Resolution metrics for a
// workspace over the interval [start,end).
//
// Start and end must be UTC day boundaries, and end must be after start.
//
// The daily series is carried forward from the latest observation before start,
// when available. Otherwise, it begins with the first observation in the
// requested interval.
func (resolutions *IdentityResolutions) MetricsPerDate(ctx context.Context, workspace string, start, end time.Time) ([]IdentityResolutionMetricDay, error) {

	rows, err := resolutions.metrics.db.Query(ctx, `WITH seed AS (
		SELECT day, profiles_anonymous, profiles_recognized,
			identities_anonymous, identities_recognized, composition_one
		FROM identity_resolution_metrics
		WHERE workspace = $1 AND day < $2
		ORDER BY day DESC
		LIMIT 1
	), range_rows AS (
		SELECT day, profiles_anonymous, profiles_recognized,
			identities_anonymous, identities_recognized, composition_one
		FROM identity_resolution_metrics
		WHERE workspace = $1 AND day >= $2 AND day < $3
	)
	SELECT * FROM seed
	UNION ALL
	SELECT * FROM range_rows
	ORDER BY day`, workspace, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dayCount := int(end.Sub(start) / (24 * time.Hour))
	updates := map[int]identityResolutionHistoryState{}
	var seed identityResolutionHistoryState
	hasSeed := false

	for rows.Next() {
		var day time.Time
		var s identityResolutionHistoryState
		if err := rows.Scan(&day, &s.profilesAnonymous, &s.profilesRecognized,
			&s.identitiesAnonymous, &s.identitiesRecognized, &s.profilesWithOne); err != nil {
			return nil, err
		}
		day = day.UTC()
		if day.Before(start) {
			seed = s
			hasSeed = true
			continue
		}
		i := int(day.Sub(start) / (24 * time.Hour))
		if i < 0 || i >= dayCount {
			return nil, fmt.Errorf("metrics: identity resolution metrics contain a day outside the requested interval")
		}
		updates[i] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	days := make([]IdentityResolutionMetricDay, 0, dayCount)
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
		day := IdentityResolutionMetricDay{
			Day:                start.AddDate(0, 0, i),
			Identities:         current.identitiesAnonymous + current.identitiesRecognized,
			Profiles:           current.profilesAnonymous + current.profilesRecognized,
			ProfilesAnonymous:  current.profilesAnonymous,
			ProfilesRecognized: current.profilesRecognized,
		}
		if day.Profiles != 0 {
			day.IdentitiesPerProfile = float64(day.Identities) / float64(day.Profiles)
		}
		if day.Identities != 0 {
			day.LinkedIdentitiesRate = float64(day.Identities-current.profilesWithOne) / float64(day.Identities)
		}
		days = append(days, day)
	}

	return days, nil
}

// RecordResult stores a successful Identity Resolution result for the UTC day
// of its observation timestamp. It replaces a result already stored for that
// day only if the new timestamp is later. The caller provides and controls the
// transaction used for the database writes.
func (resolutions *IdentityResolutions) RecordResult(ctx context.Context, tx *db.Tx, workspace string, observedAt time.Time, counts *warehouses.IdentityResolutionCounts) error {

	observedAt = observedAt.UTC()
	day := observedAt.Truncate(24 * time.Hour)

	_, err := tx.Exec(ctx, `INSERT INTO identity_resolution_metrics AS m
		(workspace, day, observed_at, profiles_anonymous, profiles_recognized,
		identities_anonymous, identities_recognized,
		composition_one, composition_two, composition_three, composition_four_to_ten,
		composition_eleven_to_twenty, composition_more_than_twenty)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	ON CONFLICT (workspace, day) DO UPDATE SET
		observed_at = EXCLUDED.observed_at,
		profiles_anonymous = EXCLUDED.profiles_anonymous,
		profiles_recognized = EXCLUDED.profiles_recognized,
		identities_anonymous = EXCLUDED.identities_anonymous,
		identities_recognized = EXCLUDED.identities_recognized,
		composition_one = EXCLUDED.composition_one,
		composition_two = EXCLUDED.composition_two,
		composition_three = EXCLUDED.composition_three,
		composition_four_to_ten = EXCLUDED.composition_four_to_ten,
		composition_eleven_to_twenty = EXCLUDED.composition_eleven_to_twenty,
		composition_more_than_twenty = EXCLUDED.composition_more_than_twenty
	WHERE EXCLUDED.observed_at > m.observed_at`,
		workspace, day, observedAt.Format("15:04:05.999999"),
		counts.Profiles.Anonymous, counts.Profiles.Recognized,
		counts.Identities.Anonymous, counts.Identities.Recognized,
		counts.Composition.One, counts.Composition.Two, counts.Composition.Three,
		counts.Composition.FourToTen, counts.Composition.ElevenToTwenty,
		counts.Composition.MoreThanTwenty)

	return err
}
