// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/warehouses"
)

// identityResolutionDayExpectation contains the expected fields for a
// historical Identity Resolution day.
type identityResolutionDayExpectation struct {
	identities           int
	profiles             int
	profilesAnonymous    int
	profilesRecognized   int
	identitiesPerProfile float64
	linkedIdentitiesRate float64
}

// persistedIdentityResolutionMetric contains the persisted fields used to
// verify an Identity Resolution result.
type persistedIdentityResolutionMetric struct {
	day         time.Time
	observedAt  time.Time
	profiles    warehouses.Counts
	identities  warehouses.Counts
	composition warehouses.IdentityResolutionComposition
}

// assertIdentityResolutionFloat verifies a floating-point metric.
func assertIdentityResolutionFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > 1e-12 {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// assertIdentityResolutionMetricDays verifies dense Identity Resolution history
// and its derived values.
func assertIdentityResolutionMetricDays(t *testing.T, got []IdentityResolutionMetricDay, start time.Time, want []identityResolutionDayExpectation) {

	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d Identity Resolution days, got %d", len(want), len(got))
	}
	for index := range want {
		expectedDay := start.AddDate(0, 0, index)
		if got[index].Day != expectedDay {
			t.Fatalf("expected day %s, got %s", expectedDay, got[index].Day)
		}
		if got[index].Identities != want[index].identities ||
			got[index].Profiles != want[index].profiles ||
			got[index].ProfilesAnonymous != want[index].profilesAnonymous ||
			got[index].ProfilesRecognized != want[index].profilesRecognized {
			t.Fatalf("expected counts %#v, got %#v", want[index], got[index])
		}
		assertIdentityResolutionFloat(t, got[index].IdentitiesPerProfile, want[index].identitiesPerProfile)
		assertIdentityResolutionFloat(t, got[index].LinkedIdentitiesRate, want[index].linkedIdentitiesRate)
	}

}

// readIdentityResolutionMetric reads a persisted Identity Resolution snapshot
// for a workspace and UTC day.
func readIdentityResolutionMetric(t *testing.T, database *db.DB, workspace string, day time.Time) persistedIdentityResolutionMetric {
	t.Helper()
	var metric persistedIdentityResolutionMetric
	err := database.QueryRow(t.Context(), `SELECT day,
		(day + observed_at) AT TIME ZONE 'UTC',
		profiles_anonymous, profiles_recognized, identities_anonymous, identities_recognized,
		composition_one, composition_two, composition_three, composition_four_to_ten,
		composition_eleven_to_twenty, composition_more_than_twenty
		FROM identity_resolution_metrics
		WHERE workspace = $1 AND day = $2`, workspace, day).Scan(
		&metric.day, &metric.observedAt,
		&metric.profiles.Anonymous, &metric.profiles.Recognized,
		&metric.identities.Anonymous, &metric.identities.Recognized,
		&metric.composition.One, &metric.composition.Two, &metric.composition.Three,
		&metric.composition.FourToTen, &metric.composition.ElevenToTwenty,
		&metric.composition.MoreThanTwenty)
	if err != nil {
		t.Fatal(err)
	}
	return metric
}

// recordIdentityResolutionResult records one Identity Resolution result in a
// database transaction.
func recordIdentityResolutionResult(t *testing.T, database *db.DB, recorder *IdentityResolutions, workspace string, observedAt time.Time, counts *warehouses.IdentityResolutionCounts) {

	t.Helper()
	err := database.Transaction(t.Context(), func(tx *db.Tx) error {
		return recorder.RecordResult(t.Context(), tx, workspace, observedAt, counts)
	})
	if err != nil {
		t.Fatal(err)
	}

}

// TestIdentityResolutionMetricsPerDateAndLatest verifies latest mapping,
// derived values, dense carry-forward history, leading-gap omission, zero
// semantics, and behavior when no successful Identity Resolution exists.
func TestIdentityResolutionMetricsPerDateAndLatest(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	resolutions := IdentityResolutions{metrics: &Metrics{db: database}}
	ctx := t.Context()
	workspace := "resoiution11"
	zeroWorkspace := "workspace111"
	emptyWorkspace := "workspace222"
	start := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)
	_, err := database.Exec(ctx, `INSERT INTO identity_resolution_metrics
		(workspace, day, observed_at,
		profiles_anonymous, profiles_recognized, identities_anonymous, identities_recognized,
		composition_one, composition_two, composition_three,
		composition_four_to_ten, composition_eleven_to_twenty,
		composition_more_than_twenty)
		VALUES
		($1, '2026-08-02', '10:05:00',
		 2, 3, 6, 4, 2, 2, 0, 1, 0, 0),
		($1, '2026-08-04', '10:05:00',
		 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		($1, '2026-08-10', '09:15:00',
		 9, 12, 100, 150, 1, 2, 3, 4, 5, 6),
		($2, '2026-08-10', '10:15:00',
		 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)`, workspace, zeroWorkspace)
	if err != nil {
		t.Fatal(err)
	}

	latest, err := resolutions.Latest(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil {
		t.Fatal("expected a latest Identity Resolution metric, got nil")
	}
	if latest.ObservedAt != time.Date(2026, time.August, 10, 9, 15, 0, 0, time.UTC) {
		t.Fatalf("expected observation timestamp 09:15 UTC, got %s", latest.ObservedAt)
	}
	if latest.Identities != (IdentityResolutionMetricCounts{Total: 250, Anonymous: 100, Recognized: 150}) {
		t.Fatalf("expected identity counts 250/100/150, got %#v", latest.Identities)
	}
	if latest.Profiles != (IdentityResolutionMetricCounts{Total: 21, Anonymous: 9, Recognized: 12}) {
		t.Fatalf("expected profile counts 21/9/12, got %#v", latest.Profiles)
	}
	if latest.Composition != (IdentityResolutionComposition{
		One: 1, Two: 2, Three: 3, FourToTen: 4, ElevenToTwenty: 5, MoreThanTwenty: 6,
	}) {
		t.Fatalf("expected composition 1/2/3/4/5/6, got %#v", latest.Composition)
	}
	assertIdentityResolutionFloat(t, latest.IdentitiesPerProfile, 250.0/21.0)
	assertIdentityResolutionFloat(t, latest.LinkedIdentitiesRate, 249.0/250.0)

	days, err := resolutions.MetricsPerDate(ctx, workspace, start, end)
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityResolutionMetricDays(t, days, start.AddDate(0, 0, 1), []identityResolutionDayExpectation{
		{identities: 10, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
			identitiesPerProfile: 2.0, linkedIdentitiesRate: 0.8},
		{identities: 10, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
			identitiesPerProfile: 2.0, linkedIdentitiesRate: 0.8},
		{},
		{},
	})
	seedStart := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC)
	seeded, err := resolutions.MetricsPerDate(ctx, workspace, seedStart, seedStart.AddDate(0, 0, 2))
	if err != nil {
		t.Fatal(err)
	}
	assertIdentityResolutionMetricDays(t, seeded, seedStart, []identityResolutionDayExpectation{
		{identities: 10, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
			identitiesPerProfile: 2.0, linkedIdentitiesRate: 0.8},
		{},
	})

	zeroLatest, err := resolutions.Latest(ctx, zeroWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if zeroLatest == nil {
		t.Fatal("expected a zero Identity Resolution metric, got nil")
	}
	if zeroLatest.Identities != (IdentityResolutionMetricCounts{}) ||
		zeroLatest.Profiles != (IdentityResolutionMetricCounts{}) ||
		zeroLatest.Composition != (IdentityResolutionComposition{}) {
		t.Fatalf("expected zero Identity Resolution counts, got %#v", zeroLatest)
	}
	assertIdentityResolutionFloat(t, zeroLatest.IdentitiesPerProfile, 0)
	assertIdentityResolutionFloat(t, zeroLatest.LinkedIdentitiesRate, 0)

	empty, err := resolutions.MetricsPerDate(ctx, emptyWorkspace, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no Identity Resolution days, got %#v", empty)
	}
	emptyLatest, err := resolutions.Latest(ctx, emptyWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if emptyLatest != nil {
		t.Fatalf("expected no latest Identity Resolution metric, got %#v", emptyLatest)
	}

}

// TestIdentityResolutionMetricsRecord verifies exact recording, equal-timestamp
// preservation, same-day replacement, and out-of-order protection.
func TestIdentityResolutionMetricsRecord(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	recorder := IdentityResolutions{}
	observedAt := time.Date(2026, time.August, 10, 0, 10, 0, 654321000, time.UTC)
	counts := &warehouses.IdentityResolutionCounts{
		Profiles:   warehouses.Counts{Anonymous: 9, Recognized: 12},
		Identities: warehouses.Counts{Anonymous: 100, Recognized: 150},
		Composition: warehouses.IdentityResolutionComposition{
			One: 1, Two: 2, Three: 3, FourToTen: 4, ElevenToTwenty: 5, MoreThanTwenty: 6,
		},
	}
	day := observedAt.Truncate(24 * time.Hour)
	recordIdentityResolutionResult(t, database, &recorder, "workspace111", observedAt, counts)
	equalTimestampCounts := *counts
	equalTimestampCounts.Identities.Anonymous++
	recordIdentityResolutionResult(t, database, &recorder, "workspace111", observedAt, &equalTimestampCounts)

	row := readIdentityResolutionMetric(t, database, "workspace111", day)
	if !row.observedAt.Equal(observedAt) {
		t.Fatalf("expected observed_at %s, got %s", observedAt, row.observedAt)
	}
	if !row.day.Equal(day) {
		t.Fatalf("expected day %s, got %s", day, row.day)
	}
	if row.profiles != counts.Profiles || row.identities != counts.Identities ||
		row.composition != counts.Composition {
		t.Fatalf("expected counts %#v, got %#v", counts, row)
	}

	newerCounts := &warehouses.IdentityResolutionCounts{
		Profiles:   warehouses.Counts{Anonymous: 10, Recognized: 17},
		Identities: warehouses.Counts{Anonymous: 120, Recognized: 180},
		Composition: warehouses.IdentityResolutionComposition{
			One: 2, Two: 3, Three: 4, FourToTen: 5, ElevenToTwenty: 6, MoreThanTwenty: 7,
		},
	}
	newerObservedAt := observedAt.Add(time.Hour)
	recordIdentityResolutionResult(t, database, &recorder, "workspace111", newerObservedAt, newerCounts)
	recordIdentityResolutionResult(t, database, &recorder, "workspace111", observedAt, counts)
	row = readIdentityResolutionMetric(t, database, "workspace111", day)
	if !row.observedAt.Equal(newerObservedAt) {
		t.Fatalf("expected newer observation timestamp %s, got %s", newerObservedAt, row.observedAt)
	}
	if row.profiles != newerCounts.Profiles || row.identities != newerCounts.Identities ||
		row.composition != newerCounts.Composition {
		t.Fatalf("expected complete newer counts %#v, got %#v", newerCounts, row)
	}

}
