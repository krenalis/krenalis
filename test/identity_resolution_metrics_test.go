// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestIdentityResolutionHTTPContract verifies metric retrieval and run history.
// It also checks that a failed resolution preserves recorded metrics without
// recording new usage observations.
func TestIdentityResolutionHTTPContract(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	t.Run("metrics", func(t *testing.T) {

		const latestPath = "/v1/metrics/identity-resolution/latest"
		const datesPath = "/v1/metrics/identity-resolution/dates/2020-01-01/2020-01-04"
		var initial json.RawMessage
		k.Call("GET", latestPath, nil, nil, &initial)
		if string(initial) != "null" {
			t.Fatalf("expected null before a successful resolution, got %s", initial)
		}
		var days []core.IdentityResolutionMetricDay
		k.Call("GET", datesPath, nil, nil, &days)
		if days == nil || len(days) != 0 {
			t.Fatalf("expected an empty daily metrics array, got %#v", days)
		}

		k.ExecQueryTestDatabase(t.Context(), `INSERT INTO identity_resolution_metrics
		(workspace, day, observed_at, profiles_anonymous, profiles_recognized,
		identities_anonymous, identities_recognized, composition_one, composition_two,
		composition_three, composition_four_to_ten, composition_eleven_to_twenty, composition_more_than_twenty)
		VALUES ($1, '2020-01-02', '10:00:00', 1, 1, 2, 1, 1, 1, 0, 0, 0, 0)`, k.WorkspaceID())
		want := core.IdentityResolutionMetric{
			ObservedAt:           time.Date(2020, time.January, 2, 10, 0, 0, 0, time.UTC),
			Identities:           core.IdentityResolutionMetricCounts{Total: 3, Anonymous: 2, Recognized: 1},
			Profiles:             core.IdentityResolutionMetricCounts{Total: 2, Anonymous: 1, Recognized: 1},
			Composition:          core.IdentityResolutionComposition{One: 1, Two: 1},
			IdentitiesPerProfile: 1.5,
			LinkedIdentitiesRate: 2.0 / 3.0,
		}
		var latest core.IdentityResolutionMetric
		k.Call("GET", latestPath, nil, nil, &latest)
		if latest != want {
			t.Fatalf("expected latest resolution metric %#v, got %#v", want, latest)
		}
		// Carry the January 2 observation forward to January 3; omit unknown January 1.
		k.Call("GET", datesPath, nil, nil, &days)
		if len(days) != 2 {
			t.Fatalf("expected two known days, got %#v", days)
		}
		for index, day := range days {
			wantDay := core.IdentityResolutionMetricDay{
				Day:                  want.ObservedAt.AddDate(0, 0, index).Format(time.DateOnly),
				Identities:           3,
				Profiles:             2,
				ProfilesAnonymous:    1,
				ProfilesRecognized:   1,
				IdentitiesPerProfile: 1.5,
				LinkedIdentitiesRate: 2.0 / 3.0,
			}
			if day != wantDay {
				t.Fatalf("expected daily resolution metric %#v, got %#v", wantDay, day)
			}
		}

		for _, interval := range []string{"invalid/2020-01-04", "2020-01-04/2020-01-01"} {
			t.Run(interval, func(t *testing.T) {
				err := k.TryCall("GET", "/v1/metrics/identity-resolution/dates/"+interval, nil, nil, nil)
				if err != nil {
					expectAPIError(t, err, http.StatusNotFound, "NotFound")
					return
				}
				t.Fatal("expected an invalid interval error, got nil")
			})
		}

		// Fail an actual warehouse operation, then observe its committed lifecycle.
		var settings krenalistester.DBSettings
		err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), &settings)
		if err != nil {
			t.Fatalf("expected warehouse settings, got %v", err)
		}
		pool, err := krenalistester.ConnectionPool(t.Context(), &settings)
		if err != nil {
			t.Fatalf("expected a warehouse connection, got %v", err)
		}
		defer pool.Close()
		_, err = pool.Exec(t.Context(), `DROP TABLE "krenalis_identities"`)
		if err != nil {
			t.Fatalf("expected identities table deletion, got %v", err)
		}
		// An empty history ensures the failed run belongs to this request.
		if runs := k.IdentityResolutionRuns(0, 1, ""); len(runs) != 0 {
			t.Fatalf("expected no previous resolution runs, got %#v", runs)
		}
		var usageRowsBefore int
		k.QueryRowTestDatabase(t.Context(), &usageRowsBefore, "SELECT COUNT(*) FROM usage_metrics WHERE workspace = $1",
			k.WorkspaceID())
		k.Call("POST", "/v1/identity-resolution/start", nil, nil, nil)
		deadline := time.Now().Add(30 * time.Second)
		var failed []krenalistester.IdentityResolutionRun
		for {
			failed = k.IdentityResolutionRuns(0, 1, "failed")
			if len(failed) == 1 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("expected a finalized failed run, got %#v", failed)
			}
			time.Sleep(20 * time.Millisecond)
		}
		run := failed[0]
		if run.Status != "failed" || run.EndTime == nil || run.Error == nil || run.StartTime.IsZero() {
			t.Fatalf("expected a failed run with lifecycle timestamps and an error, got %#v", run)
		}
		if run.EndTime.Before(run.StartTime) {
			t.Fatalf("expected end time at or after start time, got %#v", run)
		}
		if !strings.Contains(*run.Error, "krenalis_identities") {
			t.Fatalf("expected a missing identities table error, got %q", *run.Error)
		}
		var pending bool
		k.QueryRowTestDatabase(t.Context(), &pending, "SELECT ir_id IS NOT NULL FROM workspaces WHERE id = $1",
			k.WorkspaceID())
		if pending {
			t.Fatal("expected the workspace lifecycle to be closed, got a pending resolution")
		}
		latest = core.IdentityResolutionMetric{}
		k.Call("GET", latestPath, nil, nil, &latest)
		if latest != want {
			t.Fatalf("expected failed resolution to preserve metric %#v, got %#v", want, latest)
		}
		var usageRows int
		k.QueryRowTestDatabase(t.Context(), &usageRows, "SELECT COUNT(*) FROM usage_metrics WHERE workspace = $1",
			k.WorkspaceID())
		if usageRows != usageRowsBefore {
			t.Fatalf("expected %d usage rows after a failed resolution, got %d", usageRowsBefore, usageRows)
		}

	})

	t.Run("history", func(t *testing.T) {

		k.ExecQueryTestDatabase(t.Context(), "DELETE FROM identity_resolution_runs WHERE workspace = $1", k.WorkspaceID())
		if runs := k.IdentityResolutionRuns(0, 100, ""); runs == nil || len(runs) != 0 {
			t.Fatalf("expected an empty runs array, got %#v", runs)
		}

		const (
			successfulID = "11111111-1111-1111-1111-111111111111"
			failedID     = "22222222-2222-2222-2222-222222222222"
			runningID    = "33333333-3333-3333-3333-333333333333"
		)
		statuses := map[string]string{
			successfulID: "successful",
			failedID:     "failed",
			runningID:    "running",
		}
		// Equal start times exercise the descending ID tie-breaker.
		k.ExecQueryTestDatabase(t.Context(), `INSERT INTO identity_resolution_runs
		(id, workspace, start_time, end_time, error) VALUES
		($1, $4, '2020-01-01', '2020-01-01 00:01:00', NULL),
		($2, $4, '2020-01-02', '2020-01-02 00:01:00', 'warehouse failure'),
		($3, $4, '2020-01-02', NULL, NULL)`, successfulID, failedID, runningID, k.WorkspaceID())
		for _, test := range []struct {
			name         string
			first, limit int
			status       string
			ids          []string
		}{
			{name: "all", limit: 100, ids: []string{runningID, failedID, successfulID}},
			{name: "running", limit: 100, status: "running", ids: []string{runningID}},
			{name: "failed", limit: 100, status: "failed", ids: []string{failedID}},
			{name: "successful", limit: 100, status: "successful", ids: []string{successfulID}},
			{name: "offset and limit", first: 1, limit: 1, ids: []string{failedID}},
			{name: "past the last run", first: 3, limit: 1},
		} {
			t.Run(test.name, func(t *testing.T) {
				runs := k.IdentityResolutionRuns(test.first, test.limit, test.status)
				if runs == nil || len(runs) != len(test.ids) {
					t.Fatalf("expected %d runs in an array, got %#v", len(test.ids), runs)
				}
				for index, run := range runs {
					status := statuses[test.ids[index]]
					if run.ID != test.ids[index] || run.Status != status {
						t.Fatalf("expected run %s with status %q, got %#v", test.ids[index], status, run)
					}
				}
			})
		}

		for _, query := range []string{
			"first=invalid", "first=-1", "limit=0", "limit=1001", "status=invalid", "first=0&first=1",
		} {
			t.Run(query, func(t *testing.T) {
				err := k.TryCall("GET", "/v1/identity-resolution/runs?"+query, nil, nil, nil)
				if err != nil {
					expectAPIError(t, err, http.StatusBadRequest, "BadRequest")
					return
				}
				t.Fatal("expected an invalid query error, got nil")
			})
		}

	})

}
