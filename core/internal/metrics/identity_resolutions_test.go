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

// TestIdentityResolutionMetricsPerDateAndLatest verifies the latest snapshot,
// derived metrics, daily history with carry-forward, date boundaries, zero
// counts, and behavior when no resolution results exist.
func TestIdentityResolutionMetricsPerDateAndLatest(t *testing.T) {

	database := newIdentityMetricsTestDatabase(t)
	resolutions := IdentityResolutions{metrics: &Metrics{db: database}}
	ctx := t.Context()
	workspace := "resoiution11"
	zeroWorkspace := "workspace111"
	emptyWorkspace := "workspace222"
	singletonWorkspace := "workspace333"
	start := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)
	// August 2 starts the history; August 4 replaces it with a valid zero state.
	// August 10 supplies the latest result outside the requested history range.
	_, err := database.Exec(ctx, `INSERT INTO identity_resolution_metrics
		(workspace, day, observed_at,
		profiles_anonymous, profiles_recognized, identities_anonymous, identities_recognized,
		composition_one, composition_two, composition_three,
		composition_four_to_ten, composition_eleven_to_twenty,
		composition_more_than_twenty)
		VALUES
		($1, '2026-08-02', '10:05:00',
		 2, 3, 9, 4, 2, 2, 0, 1, 0, 0),
		($1, '2026-08-04', '10:05:00',
		 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		($1, '2026-08-10', '09:15:00',
		 9, 12, 100, 150, 1, 2, 3, 4, 5, 6),
		($2, '2026-08-10', '10:15:00',
		 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		($3, '2026-08-02', '10:05:00',
		 2, 3, 2, 3, 5, 0, 0, 0, 0, 0)`, workspace, zeroWorkspace, singletonWorkspace)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	latest, err := resolutions.Latest(ctx, workspace)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
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
	assertIdentityResolutionFloat(t, "latest identities per profile", latest.IdentitiesPerProfile, 250.0/21.0)
	assertIdentityResolutionFloat(t, "latest linked identities rate", latest.LinkedIdentitiesRate, 249.0/250.0)

	days, err := resolutions.MetricsPerDate(ctx, workspace, start, end)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertIdentityResolutionMetricDays(t, days, start.AddDate(0, 0, 1), []identityResolutionDayExpectation{
		{identities: 13, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
			identitiesPerProfile: 2.6, linkedIdentitiesRate: 11.0 / 13.0},
		{identities: 13, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
			identitiesPerProfile: 2.6, linkedIdentitiesRate: 11.0 / 13.0},
		{},
		{},
	})
	seedStart := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC)
	seeded, err := resolutions.MetricsPerDate(ctx, workspace, seedStart, seedStart.AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertIdentityResolutionMetricDays(t, seeded, seedStart, []identityResolutionDayExpectation{
		{identities: 13, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
			identitiesPerProfile: 2.6, linkedIdentitiesRate: 11.0 / 13.0},
		{},
	})

	for _, test := range []struct {
		name       string
		start, end time.Time
		want       []identityResolutionDayExpectation
	}{
		{
			name:  "latest prior observation without updates",
			start: time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
			want:  []identityResolutionDayExpectation{{}, {}},
		},
		{
			name:  "observation at start replaces prior state",
			start: time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC),
			want:  []identityResolutionDayExpectation{{}, {}},
		},
		{
			name:  "observation at end is excluded",
			start: time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC),
			want: []identityResolutionDayExpectation{
				{identities: 13, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
					identitiesPerProfile: 2.6, linkedIdentitiesRate: 11.0 / 13.0},
				{identities: 13, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
					identitiesPerProfile: 2.6, linkedIdentitiesRate: 11.0 / 13.0},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			days, err := resolutions.MetricsPerDate(t.Context(), workspace, test.start, test.end)
			if err != nil {
				t.Fatalf("expected daily metrics, got %v", err)
			}
			assertIdentityResolutionMetricDays(t, days, test.start, test.want)
		})
	}

	t.Run("single-identity profiles", func(t *testing.T) {

		latest, err := resolutions.Latest(t.Context(), singletonWorkspace)
		if err != nil {
			t.Fatalf("expected latest metric for single-identity profiles, got %v", err)
		}
		if latest == nil {
			t.Fatal("expected latest metric for single-identity profiles, got nil")
		}
		if latest.Identities.Total != 5 || latest.Profiles.Total != 5 || latest.Composition.One != 5 {
			t.Fatalf("expected 5 identities in 5 single-identity profiles, got %#v", latest)
		}
		assertIdentityResolutionFloat(t, "latest identities per profile", latest.IdentitiesPerProfile, 1)
		assertIdentityResolutionFloat(t, "latest linked identities rate", latest.LinkedIdentitiesRate, 0)

		start := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
		days, err := resolutions.MetricsPerDate(t.Context(), singletonWorkspace, start, start.AddDate(0, 0, 2))
		if err != nil {
			t.Fatalf("expected daily metrics for single-identity profiles, got %v", err)
		}
		assertIdentityResolutionMetricDays(t, days, start, []identityResolutionDayExpectation{
			{identities: 5, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
				identitiesPerProfile: 1, linkedIdentitiesRate: 0},
			{identities: 5, profiles: 5, profilesAnonymous: 2, profilesRecognized: 3,
				identitiesPerProfile: 1, linkedIdentitiesRate: 0},
		})

	})

	zeroLatest, err := resolutions.Latest(ctx, zeroWorkspace)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if zeroLatest == nil {
		t.Fatal("expected a zero Identity Resolution metric, got nil")
	}
	if zeroLatest.Identities != (IdentityResolutionMetricCounts{}) ||
		zeroLatest.Profiles != (IdentityResolutionMetricCounts{}) ||
		zeroLatest.Composition != (IdentityResolutionComposition{}) {
		t.Fatalf("expected zero Identity Resolution counts, got %#v", zeroLatest)
	}
	assertIdentityResolutionFloat(t, "zero latest identities per profile", zeroLatest.IdentitiesPerProfile, 0)
	assertIdentityResolutionFloat(t, "zero latest linked identities rate", zeroLatest.LinkedIdentitiesRate, 0)

	empty, err := resolutions.MetricsPerDate(ctx, emptyWorkspace, start, end)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no Identity Resolution days, got %#v", empty)
	}
	emptyLatest, err := resolutions.Latest(ctx, emptyWorkspace)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if emptyLatest != nil {
		t.Fatalf("expected no latest Identity Resolution metric, got %#v", emptyLatest)
	}

}

// TestIdentityResolutionMetricsRecord verifies persisted timestamps and counts,
// same-day updates, and that observations with equal or older timestamps
// do not overwrite existing results.
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
	newerObservedAt := observedAt.Add(time.Microsecond)
	recordIdentityResolutionResult(t, database, &recorder, "workspace111", newerObservedAt, newerCounts)
	row = readIdentityResolutionMetric(t, database, "workspace111", day)
	if !row.observedAt.Equal(newerObservedAt) {
		t.Fatalf("expected updated observation timestamp %s, got %s", newerObservedAt, row.observedAt)
	}
	if row.profiles != newerCounts.Profiles || row.identities != newerCounts.Identities ||
		row.composition != newerCounts.Composition {
		t.Fatalf("expected updated counts %#v, got %#v", newerCounts, row)
	}

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

// identityResolutionDayExpectation holds the expected counts and ratios
// for one day of Identity Resolution history.
type identityResolutionDayExpectation struct {
	identities           int
	profiles             int
	profilesAnonymous    int
	profilesRecognized   int
	identitiesPerProfile float64
	linkedIdentitiesRate float64
}

// persistedIdentityResolutionMetric holds an Identity Resolution snapshot
// read from the database for test assertions.
type persistedIdentityResolutionMetric struct {
	day         time.Time
	observedAt  time.Time
	profiles    warehouses.Counts
	identities  warehouses.Counts
	composition warehouses.IdentityResolutionComposition
}

// assertIdentityResolutionFloat checks a floating-point metric against its
// expected value.
func assertIdentityResolutionFloat(t *testing.T, metric string, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > 1e-12 {
		t.Errorf("expected %s %v, got %v", metric, want, got)
	}
}

// assertIdentityResolutionMetricDays verifies consecutive daily metrics and
// their derived values.
func assertIdentityResolutionMetricDays(t *testing.T, got []IdentityResolutionMetricDay, start time.Time, want []identityResolutionDayExpectation) {

	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d Identity Resolution days, got %d", len(want), len(got))
	}

	for index := range want {
		expectedDay := start.AddDate(0, 0, index)
		if got[index].Day != expectedDay {
			t.Errorf("expected day %s, got %s", expectedDay, got[index].Day)
		}
		day := expectedDay.Format(time.DateOnly)
		if got[index].Identities != want[index].identities {
			t.Errorf("expected identities on %s to be %d, got %d", day, want[index].identities, got[index].Identities)
		}
		if got[index].Profiles != want[index].profiles {
			t.Errorf("expected profiles on %s to be %d, got %d", day, want[index].profiles, got[index].Profiles)
		}
		if got[index].ProfilesAnonymous != want[index].profilesAnonymous {
			t.Errorf("expected anonymous profiles on %s to be %d, got %d",
				day, want[index].profilesAnonymous, got[index].ProfilesAnonymous)
		}
		if got[index].ProfilesRecognized != want[index].profilesRecognized {
			t.Errorf("expected recognized profiles on %s to be %d, got %d",
				day, want[index].profilesRecognized, got[index].ProfilesRecognized)
		}
		assertIdentityResolutionFloat(t, "identities per profile on "+day,
			got[index].IdentitiesPerProfile, want[index].identitiesPerProfile)
		assertIdentityResolutionFloat(t, "linked identities rate on "+day,
			got[index].LinkedIdentitiesRate, want[index].linkedIdentitiesRate)
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
		t.Fatalf("expected no error, got %v", err)
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
		t.Fatalf("expected no error, got %v", err)
	}
}
