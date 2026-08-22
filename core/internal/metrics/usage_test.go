// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
)

// TestBuildUsageSeriesAtStartOfCurrentDayUsesCurrentProfiles verifies that the
// average uses the current profile count when less than one second has elapsed.
func TestBuildUsageSeriesAtStartOfCurrentDayUsesCurrentProfiles(t *testing.T) {
	start := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	stored := &usageStoredWorkspace{profiles: 42, days: map[int]usageStoredDay{}}
	series, err := buildUsageSeries("workspace", start, start.Add(500*time.Millisecond), 1, stored)
	if err != nil {
		t.Fatal(err)
	}
	if series.Days[0].Profiles != 42 || series.Days[0].ProfilesAvg != 42 {
		t.Fatalf("expected current profile count as the average, got %#v", series.Days[0])
	}
}

// TestBuildUsageSeriesCalculatesCompletedAndCurrentDayAverages verifies that
// profile averages are time-weighted for completed and current days.
func TestBuildUsageSeriesCalculatesCompletedAndCurrentDayAverages(t *testing.T) {
	start := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	calculatedAt := start.Add(36 * time.Hour)
	firstObservedAt := start.Add(10 * time.Hour)
	secondObservedAt := start.Add(30 * time.Hour)
	stored := &usageStoredWorkspace{
		profiles: 80,
		days: map[int]usageStoredDay{
			0: {
				profiles:       100,
				profileSeconds: 80 * 10 * 60 * 60,
				observedAt:     &firstObservedAt,
				events:         12,
			},
			1: {
				profiles:       140,
				profileSeconds: 100 * 6 * 60 * 60,
				observedAt:     &secondObservedAt,
				events:         7,
			},
		},
	}

	series, err := buildUsageSeries("workspace", start, calculatedAt, 2, stored)
	if err != nil {
		t.Fatal(err)
	}
	if series.Days[0].Profiles != 100 || series.Days[0].Events != 12 {
		t.Fatalf("unexpected completed day: %#v", series.Days[0])
	}
	if series.Days[0].ProfilesAvg != 92 {
		t.Fatalf("expected rounded average 92, got %d", series.Days[0].ProfilesAvg)
	}
	if series.Days[1].Profiles != 140 || series.Days[1].Events != 7 {
		t.Fatalf("unexpected current day: %#v", series.Days[1])
	}
	if series.Days[1].ProfilesAvg != 120 {
		t.Fatalf("expected average 120, got %d", series.Days[1].ProfilesAvg)
	}
}

// TestBuildUsageSeriesIgnoresProfileDefaultsWithoutObservation verifies that a
// day with events but no profile observation preserves the previous profile
// count.
func TestBuildUsageSeriesIgnoresProfileDefaultsWithoutObservation(t *testing.T) {
	start := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	stored := &usageStoredWorkspace{
		profiles: 42,
		days: map[int]usageStoredDay{
			0: {events: 7},
		},
	}
	series, err := buildUsageSeries("workspace", start, start.Add(24*time.Hour), 1, stored)
	if err != nil {
		t.Fatal(err)
	}
	day := series.Days[0]
	if day.Profiles != 42 || day.ProfilesAvg != 42 || day.Events != 7 {
		t.Fatalf("expected event-only row to preserve previous profile count, got %#v", day)
	}
}

// TestBuildUsageSeriesPreservesLargeRepresentableAverage verifies that an
// intermediate profile-second value exceeding the maximum int value still
// produces an exact result.
func TestBuildUsageSeriesPreservesLargeRepresentableAverage(t *testing.T) {
	start := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	stored := &usageStoredWorkspace{profiles: math.MaxInt, days: map[int]usageStoredDay{}}
	series, err := buildUsageSeries("workspace", start, start.Add(24*time.Hour), 1, stored)
	if err != nil {
		t.Fatal(err)
	}
	day := series.Days[0]
	if day.Profiles != math.MaxInt || day.ProfilesAvg != math.MaxInt {
		t.Fatalf("expected maximum profile count and average, got %#v", day)
	}
}

// TestIngestedEventsAggregatesByOrganizationWorkspaceAndUTCDay verifies that
// event counts are grouped by organization, workspace, and UTC day.
func TestIngestedEventsAggregatesByOrganizationWorkspaceAndUTCDay(t *testing.T) {
	u := &Usage{events: map[eventCountKey]int{}}
	receivedAt := time.Date(2026, time.August, 2, 0, 30, 0, 0, time.FixedZone("test", 2*60*60))
	u.IngestedEvents("organization", "workspace", receivedAt, 2)
	u.IngestedEvents("organization", "workspace", receivedAt.Add(2*time.Hour), 3)
	u.IngestedEvents("organization", "other-workspace", receivedAt, 4)
	u.IngestedEvents("other-organization", "workspace", receivedAt, 5)
	u.IngestedEvents("organization", "workspace", receivedAt, 0)
	u.IngestedEvents("organization", "workspace", receivedAt, -1)

	firstDay := receivedAt.UTC().Truncate(24 * time.Hour)
	secondDay := receivedAt.Add(2 * time.Hour).UTC().Truncate(24 * time.Hour)
	if firstDay == secondDay {
		t.Fatal("test timestamps must fall on different UTC days")
	}
	if got := u.events[eventCountKey{"organization", "workspace", firstDay}]; got != 2 {
		t.Fatalf("expected first UTC day count 2, got %d", got)
	}
	if got := u.events[eventCountKey{"organization", "workspace", secondDay}]; got != 3 {
		t.Fatalf("expected second UTC day count 3, got %d", got)
	}
	if got := u.events[eventCountKey{"organization", "other-workspace", firstDay}]; got != 4 {
		t.Fatalf("expected other workspace count 4, got %d", got)
	}
	if got := u.events[eventCountKey{"other-organization", "workspace", firstDay}]; got != 5 {
		t.Fatalf("expected other organization count 5, got %d", got)
	}
}

// TestUsageAverageRejectsUnrepresentableResult verifies that an average outside
// the int range returns ErrMetricResultTooLarge.
func TestUsageAverageRejectsUnrepresentableResult(t *testing.T) {
	var profileSeconds big.Int
	profileSeconds.SetInt64(int64(math.MaxInt))
	profileSeconds.Add(&profileSeconds, big.NewInt(1))
	_, err := usageAverage(&profileSeconds, 1, 0)
	if err != nil {
		if !errors.Is(err, ErrMetricResultTooLarge) {
			t.Fatalf("expected ErrMetricResultTooLarge, got %v", err)
		}
		return
	}
	t.Fatal("expected ErrMetricResultTooLarge, got nil")
}

// TestUsageAverageRoundsToNearestInteger verifies that the time-weighted
// profile average is rounded to the nearest integer.
func TestUsageAverageRoundsToNearestInteger(t *testing.T) {
	for _, test := range []struct {
		profileSeconds int
		seconds        int
		expected       int
	}{
		{profileSeconds: 1, seconds: 3, expected: 0},
		{profileSeconds: 2, seconds: 3, expected: 1},
		{profileSeconds: 1, seconds: 2, expected: 1},
		{profileSeconds: 4, seconds: 3, expected: 1},
		{profileSeconds: 5, seconds: 3, expected: 2},
	} {
		var profileSeconds big.Int
		profileSeconds.SetInt64(int64(test.profileSeconds))
		got, err := usageAverage(&profileSeconds, test.seconds, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.expected {
			t.Fatalf("usageAverage(%d, %d) = %d; expected %d",
				test.profileSeconds, test.seconds, got, test.expected)
		}
	}
}

// TestUsageMetricsPerDateRejectsUnrepresentableOrganizationTotals verifies that
// ErrMetricResultTooLarge is returned for organization profile and event totals
// outside the int range.
func TestUsageMetricsPerDateRejectsUnrepresentableOrganizationTotals(t *testing.T) {

	database, organization := newAggregateMetricsTestDatabase(t)
	usage := Usage{metrics: &Metrics{db: database}}
	start := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	observedAt := "00:00:00"
	tests := []struct {
		name       string
		day        time.Time
		profiles   int
		observedAt *string
		events     int
	}{
		{name: "profiles", day: start.AddDate(0, 0, -1), profiles: math.MaxInt, observedAt: &observedAt},
		{name: "events", day: start, events: math.MaxInt},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if _, err := database.Exec(t.Context(), "DELETE FROM usage_metrics"); err != nil {
				t.Fatal(err)
			}
			for _, workspace := range []string{"overflow1111", "overflow2222"} {
				_, err := database.Exec(t.Context(), `INSERT INTO usage_metrics
					(organization, workspace, day, profiles, observed_at, events)
					VALUES ($1, $2, $3, $4, $5, $6)`, organization, workspace,
					test.day, test.profiles, test.observedAt, test.events)
				if err != nil {
					t.Fatal(err)
				}
			}
			_, err := usage.MetricsPerDate(t.Context(), organization, start,
				start.AddDate(0, 0, 1), nil)
			if err != nil {
				if !errors.Is(err, ErrMetricResultTooLarge) {
					t.Fatalf("expected ErrMetricResultTooLarge, got %v", err)
				}
				return
			}
			t.Fatal("expected ErrMetricResultTooLarge, got nil")
		})
	}

}
