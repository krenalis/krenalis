// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"

	"github.com/krenalis/analytics-go"
)

type usageMetricsResponse struct {
	Start   string              `json:"start"`
	End     string              `json:"end"`
	Metrics []usageMetricSeries `json:"metrics"`
}

type usageMetricSeries struct {
	Workspace string           `json:"workspace"`
	Days      []usageMetricDay `json:"days"`
}

type usageMetricDay struct {
	Day         string `json:"day"`
	Profiles    int    `json:"profiles"`
	ProfilesAvg int    `json:"profilesAvg"`
	Events      int    `json:"events"`
}

// TestUsageMetricsHTTPContract verifies usage metric collection, authorization,
// aggregation, API representation, and workspace deletion behavior.
func TestUsageMetricsHTTPContract(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	now := time.Now().UTC()
	if nextDay := now.Truncate(24 * time.Hour).Add(24 * time.Hour); nextDay.Sub(now) < 5*time.Minute {
		t.Skip("test started too close to the next UTC day")
	}
	start := now.AddDate(0, 0, -1).Format(time.DateOnly)
	end := now.AddDate(0, 0, 1).Format(time.DateOnly)
	path := fmt.Sprintf("/v1/metrics/usage/dates/%s/%s", start, end)
	var storedRows int
	k.QueryRowTestDatabase(t.Context(), &storedRows, "SELECT COUNT(*) FROM usage_metrics")
	if storedRows != 0 {
		t.Fatalf("expected organization and workspace creation not to create usage metrics, got %d rows", storedRows)
	}
	k.CreateOrganization("Usage metrics organization", true, krenalistester.DefaultOrganizationLimits)
	k.QueryRowTestDatabase(t.Context(), &storedRows, "SELECT COUNT(*) FROM usage_metrics")
	if storedRows != 0 {
		t.Fatalf("expected organization creation not to create usage metrics, got %d rows", storedRows)
	}
	var organization string
	k.QueryRowTestDatabase(t.Context(), &organization,
		"SELECT organization FROM workspaces WHERE id = $1", k.WorkspaceID())

	var initial usageMetricsResponse
	k.Call("GET", path, nil, nil, &initial)
	assertUsageMetricsResponse(t, initial, k.WorkspaceID(), 2)
	if initial.Metrics[0].Days[0].Profiles != 0 || initial.Metrics[0].Days[0].ProfilesAvg != 0 {
		t.Fatalf("expected pre-tracking day to contain zero profiles, got %#v", initial.Metrics[0].Days[0])
	}

	connection := k.CreateJavaScriptSource("Usage metrics source", nil)
	keys := k.EventWriteKeys(connection)
	if len(keys) != 1 {
		t.Fatalf("expected one event write key, got %d", len(keys))
	}
	event := analytics.Track{
		MessageId:   "usage-metrics-duplicate",
		AnonymousId: "usage-metrics-anonymous-id",
		Event:       "Usage Metrics Test",
	}
	k.SendEvent(keys[0], event)
	k.SendEvent(keys[0], event)

	deadline := time.Now().Add(5 * time.Second)
	var workspaceMetrics usageMetricsResponse
	for {
		k.Call("GET", path, nil, nil, &workspaceMetrics)
		if usageDay(t, workspaceMetrics, now.Format(time.DateOnly)).Events == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected one accepted non-duplicate event, got %#v", workspaceMetrics)
		}
		time.Sleep(50 * time.Millisecond)
	}
	k.CreatePipeline(connection, "Event", krenalistester.PipelineToSet{
		Name:    "Usage metrics event pipeline",
		Enabled: true,
	})
	event.MessageId = "usage-metrics-with-pipeline"
	k.SendEvent(keys[0], event)
	deadline = time.Now().Add(5 * time.Second)
	for {
		k.Call("GET", path, nil, nil, &workspaceMetrics)
		if usageDay(t, workspaceMetrics, now.Format(time.DateOnly)).Events == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected two accepted events independent of pipeline presence, got %#v", workspaceMetrics)
		}
		time.Sleep(50 * time.Millisecond)
	}

	createPipelineMetricsFixture(t, k)
	k.RunIdentityResolutionAndWait()
	_, _, total := k.Profiles(nil, "", false, 0, 100)
	var resolutionObservedAt time.Time
	var resolutionProfiles int
	k.QueryRowTestDatabase(t.Context(), &resolutionObservedAt, `
		SELECT (day + observed_at) AT TIME ZONE 'UTC'
		FROM identity_resolution_metrics
		WHERE workspace = $1`, k.WorkspaceID())
	k.QueryRowTestDatabase(t.Context(), &resolutionProfiles, `
		SELECT profiles_anonymous + profiles_recognized
		FROM identity_resolution_metrics
		WHERE workspace = $1`, k.WorkspaceID())
	var lifecycleCompletedAt time.Time
	k.QueryRowTestDatabase(t.Context(), &lifecycleCompletedAt, `
		SELECT ir_end_time AT TIME ZONE 'UTC'
		FROM workspaces
		WHERE organization = $1 AND id = $2`, organization, k.WorkspaceID())
	if !resolutionObservedAt.Equal(lifecycleCompletedAt) {
		t.Fatalf("expected Resolution observed_at %s, got %s", lifecycleCompletedAt, resolutionObservedAt)
	}
	if resolutionProfiles != total {
		t.Fatalf("expected Resolution profile count %d, got %d", total, resolutionProfiles)
	}
	k.Call("GET", path, nil, nil, &workspaceMetrics)
	today := usageDay(t, workspaceMetrics, now.Format(time.DateOnly))
	if today.Profiles != total {
		t.Fatalf("expected profile count %d, got %d", total, today.Profiles)
	}
	if today.ProfilesAvg < 0 || today.ProfilesAvg > today.Profiles {
		t.Fatalf("expected current daily average between zero and %d, got %d", today.Profiles, today.ProfilesAvg)
	}

	var organizationMetrics usageMetricsResponse
	k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &organizationMetrics)
	assertUsageMetricsResponse(t, organizationMetrics, "", 2)
	var rawOrganizationMetrics struct {
		Metrics []map[string]json.RawMessage `json:"metrics"`
	}
	k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &rawOrganizationMetrics)
	if _, found := rawOrganizationMetrics.Metrics[0]["workspace"]; found {
		t.Fatal("expected the organization metric series to omit workspace")
	}
	var rawDays []map[string]json.RawMessage
	if err := json.Unmarshal(rawOrganizationMetrics.Metrics[0]["days"], &rawDays); err != nil {
		t.Fatal(err)
	}
	for _, day := range rawDays {
		if _, found := day["observedAt"]; found {
			t.Fatal("expected usage metric days not to expose profile observation time")
		}
		if _, found := day["profileSeconds"]; found {
			t.Fatal("expected usage metric days not to expose profile seconds")
		}
	}
	var rawResponse map[string]json.RawMessage
	k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &rawResponse)
	if _, found := rawResponse["timezone"]; found {
		t.Fatal("expected usage metrics not to expose a timezone field")
	}
	if _, found := rawResponse["asOf"]; found {
		t.Fatal("expected usage metrics not to expose an asOf field")
	}
	organizationToday := usageDay(t, organizationMetrics, now.Format(time.DateOnly))
	if organizationToday.Profiles != today.Profiles ||
		organizationToday.ProfilesAvg != today.ProfilesAvg ||
		organizationToday.Events != today.Events {
		t.Fatalf("expected organization metrics to match its only workspace, got organization=%#v workspace=%#v", organizationToday, today)
	}
	if today.Profiles <= 0 {
		t.Fatalf("expected positive profile count before workspace deletion, got %d", today.Profiles)
	}

	var selected usageMetricsResponse
	k.Call("GET", path+"?workspaces="+k.WorkspaceID(), http.Header{"Krenalis-Workspace": nil}, nil, &selected)
	assertUsageMetricsResponse(t, selected, k.WorkspaceID(), 2)

	token := k.CreateWorkspaceRestrictedAPIKey("usage-metrics-workspace")
	headers := http.Header{
		"Authorization":      []string{"Bearer " + token},
		"Krenalis-Workspace": nil,
	}
	var empty usageMetricsResponse
	k.Call("GET", path+"?workspaces=7QaT3mN7KxP5", headers, nil, &empty)
	if empty.Metrics == nil || len(empty.Metrics) != 0 {
		t.Fatalf("expected empty metric series for a disjoint workspace selection, got %#v", empty.Metrics)
	}
	var restricted usageMetricsResponse
	k.Call("GET", path, headers, nil, &restricted)
	assertUsageMetricsResponse(t, restricted, k.WorkspaceID(), 2)
	var compatible usageMetricsResponse
	k.Call("GET", path+"?workspaces="+k.WorkspaceID(), headers, nil, &compatible)
	assertUsageMetricsResponse(t, compatible, k.WorkspaceID(), 2)

	var profileSecondsBeforeDelete int
	k.QueryRowTestDatabase(t.Context(), &profileSecondsBeforeDelete, `
		SELECT profile_seconds
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, k.WorkspaceID())
	var profileObservedAt time.Time
	k.QueryRowTestDatabase(t.Context(), &profileObservedAt, `
		SELECT (day + observed_at) AT TIME ZONE 'UTC'
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, k.WorkspaceID())
	profileObservationUnix := int(profileObservedAt.Unix())
	for time.Now().UTC().Unix() <= int64(profileObservationUnix) {
		time.Sleep(10 * time.Millisecond)
	}

	deletedWorkspace := k.WorkspaceID()
	deleteStartedAt := time.Now().UTC()
	k.Call("DELETE", "/v1/workspaces/current", nil, nil, nil)
	deleteFinishedAt := time.Now().UTC()

	var retainedRows int
	k.QueryRowTestDatabase(t.Context(), &retainedRows, `
		SELECT COUNT(*)
		FROM usage_metrics
		WHERE workspace = $1`, deletedWorkspace)
	if retainedRows == 0 {
		t.Fatal("expected usage metrics for deleted workspace to be retained")
	}

	var deletedProfiles int
	k.QueryRowTestDatabase(t.Context(), &deletedProfiles, `
		SELECT profiles
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, deletedWorkspace)
	if deletedProfiles != 0 {
		t.Fatalf("expected deleted workspace profile count zero, got %d", deletedProfiles)
	}

	var deletedProfileSeconds int
	k.QueryRowTestDatabase(t.Context(), &deletedProfileSeconds, `
		SELECT profile_seconds
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, deletedWorkspace)
	if deletedProfileSeconds < 0 {
		t.Fatalf("expected non-negative profile seconds, got %d", deletedProfileSeconds)
	}
	var deletedDayUnix int
	k.QueryRowTestDatabase(t.Context(), &deletedDayUnix, `
		SELECT EXTRACT(EPOCH FROM (day::timestamp AT TIME ZONE 'UTC'))::bigint
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, deletedWorkspace)
	baseProfileSeconds := 0
	profileStateStartedAt := deletedDayUnix
	if profileObservationUnix >= deletedDayUnix {
		baseProfileSeconds = profileSecondsBeforeDelete
		profileStateStartedAt = profileObservationUnix
	}
	minimumProfileSeconds := baseProfileSeconds + today.Profiles*max(0, int(deleteStartedAt.Unix())-profileStateStartedAt)
	maximumProfileSeconds := baseProfileSeconds + today.Profiles*max(0, int(deleteFinishedAt.Unix())-profileStateStartedAt)
	if deletedProfileSeconds < minimumProfileSeconds || deletedProfileSeconds > maximumProfileSeconds {
		t.Fatalf("expected deleted workspace profile seconds in [%d,%d], got %d",
			minimumProfileSeconds, maximumProfileSeconds, deletedProfileSeconds)
	}

	var hasObservedAt bool
	k.QueryRowTestDatabase(t.Context(), &hasObservedAt, `
		SELECT observed_at IS NOT NULL
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, deletedWorkspace)
	if !hasObservedAt {
		t.Fatal("expected deleted workspace observation time")
	}

	var deletedEvents int
	k.QueryRowTestDatabase(t.Context(), &deletedEvents, `
		SELECT events
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, deletedWorkspace)
	if deletedEvents != today.Events {
		t.Fatalf("expected deleted workspace events %d, got %d", today.Events, deletedEvents)
	}

	var afterDelete usageMetricsResponse
	k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &afterDelete)
	afterDeleteToday := usageDay(t, afterDelete, now.Format(time.DateOnly))
	if afterDeleteToday.Profiles != 0 {
		t.Fatalf("expected organization profile count zero after workspace deletion, got %d", afterDeleteToday.Profiles)
	}
	if afterDeleteToday.ProfilesAvg < 0 || afterDeleteToday.ProfilesAvg > today.Profiles {
		t.Fatalf("expected organization average between zero and %d after deletion, got %d",
			today.Profiles, afterDeleteToday.ProfilesAvg)
	}
	if afterDeleteToday.Events != organizationToday.Events {
		t.Fatalf("expected workspace deletion to preserve organization event history %d, got %d",
			organizationToday.Events, afterDeleteToday.Events)
	}
}

// TestUsageMetricsRetriesEventPersistence verifies that an event count is
// persisted once after PostgreSQL rejects the first write.
func TestUsageMetricsRetriesEventPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	connection := k.CreateJavaScriptSource("Usage metrics retry source", nil)
	keys := k.EventWriteKeys(connection)
	if len(keys) != 1 {
		t.Fatalf("expected one event write key, got %d", len(keys))
	}

	// Fail exactly the first metrics write. Sequence values are not rolled back,
	// so the retry sees the next value and succeeds.
	k.ExecQueryTestDatabase(t.Context(), `
		CREATE SEQUENCE usage_metrics_test_write_attempts;
		CREATE FUNCTION fail_first_usage_metrics_test_write()
		RETURNS trigger
		LANGUAGE plpgsql
		AS $$
		BEGIN
			IF nextval('usage_metrics_test_write_attempts') = 1 THEN
				RAISE EXCEPTION 'simulated temporary usage metrics write failure';
			END IF;
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER fail_first_usage_metrics_test_write
		BEFORE INSERT OR UPDATE ON usage_metrics
		FOR EACH ROW
		EXECUTE FUNCTION fail_first_usage_metrics_test_write()`)
	defer func() {
		k.ExecQueryTestDatabase(t.Context(), `
			DROP TRIGGER fail_first_usage_metrics_test_write ON usage_metrics;
			DROP FUNCTION fail_first_usage_metrics_test_write();
			DROP SEQUENCE usage_metrics_test_write_attempts`)
	}()
	k.SendEvent(keys[0], analytics.Track{
		MessageId:   "usage-metrics-retry",
		AnonymousId: "usage-metrics-retry-anonymous-id",
		Event:       "Usage Metrics Retry Test",
	})

	deadline := time.Now().Add(15 * time.Second)
	for {
		var events int
		k.QueryRowTestDatabase(t.Context(), &events,
			"SELECT COALESCE(SUM(events), 0) FROM usage_metrics WHERE workspace = $1",
			k.WorkspaceID())
		if events == 1 {
			break
		}
		if events > 1 {
			t.Fatalf("expected one persisted event, got %d", events)
		}
		if time.Now().After(deadline) {
			t.Fatal("event count was not persisted after database writes recovered")
		}
		time.Sleep(50 * time.Millisecond)
	}

	var attempts int
	k.QueryRowTestDatabase(t.Context(), &attempts,
		"SELECT last_value FROM usage_metrics_test_write_attempts")
	if attempts < 2 {
		t.Fatalf("expected at least two persistence attempts, got %d", attempts)
	}
}

// TestUsageMetricsWorkspaceDeletionWithoutProfileObservation verifies that
// deleting a workspace with no profile observation records a final zero state.
func TestUsageMetricsWorkspaceDeletionWithoutProfileObservation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	workspace := k.WorkspaceID()
	k.Call("DELETE", "/v1/workspaces/current", nil, nil, nil)

	var profiles int
	k.QueryRowTestDatabase(t.Context(), &profiles, `
		SELECT profiles
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, workspace)
	if profiles != 0 {
		t.Fatalf("expected deleted workspace profile count zero, got %d", profiles)
	}

	var profileSeconds int
	k.QueryRowTestDatabase(t.Context(), &profileSeconds, `
		SELECT profile_seconds
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, workspace)
	if profileSeconds != 0 {
		t.Fatalf("expected deleted workspace profile seconds zero, got %d", profileSeconds)
	}

	var hasObservedAt bool
	k.QueryRowTestDatabase(t.Context(), &hasObservedAt, `
		SELECT observed_at IS NOT NULL
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, workspace)
	if !hasObservedAt {
		t.Fatal("expected deleted workspace observation time")
	}

	var events int
	k.QueryRowTestDatabase(t.Context(), &events, `
		SELECT events
		FROM usage_metrics
		WHERE workspace = $1
		ORDER BY day DESC
		LIMIT 1`, workspace)
	if events != 0 {
		t.Fatalf("expected deleted workspace events zero, got %d", events)
	}
}

// TestUsageMetricsCarriesPreviousProfileState verifies that profile state is
// carried across UTC days and profile seconds accumulate until workspace deletion.
func TestUsageMetricsCarriesPreviousProfileState(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	now := time.Now().UTC()
	workspace := k.WorkspaceID()
	k.ExecQueryTestDatabase(t.Context(), `
		INSERT INTO usage_metrics
			(organization, workspace, day, profiles, profile_seconds, observed_at)
		SELECT organization, id, $2, 42, 0, '12:00:00'
		FROM workspaces
		WHERE id = $1`, workspace, now.AddDate(0, 0, -1).Format(time.DateOnly))

	start := now.Format(time.DateOnly)
	end := now.AddDate(0, 0, 1).Format(time.DateOnly)
	path := fmt.Sprintf("/v1/metrics/usage/dates/%s/%s", start, end)

	var workspaceMetrics usageMetricsResponse
	k.Call("GET", path, nil, nil, &workspaceMetrics)
	workspaceDay := usageDay(t, workspaceMetrics, start)
	if workspaceDay.Profiles != 42 || workspaceDay.ProfilesAvg != 42 {
		t.Fatalf("expected previous workspace profile state to be carried forward, got %#v", workspaceDay)
	}

	var organizationMetrics usageMetricsResponse
	k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &organizationMetrics)
	organizationDay := usageDay(t, organizationMetrics, start)
	if organizationDay.Profiles != 42 || organizationDay.ProfilesAvg != 42 {
		t.Fatalf("expected previous organization profile state to be carried forward, got %#v", organizationDay)
	}

	deleteStartedAt := time.Now().UTC()
	k.Call("DELETE", "/v1/workspaces/current", nil, nil, nil)
	deleteFinishedAt := time.Now().UTC()
	if deleteStartedAt.Format(time.DateOnly) != deleteFinishedAt.Format(time.DateOnly) {
		t.Skip("workspace deletion crossed a UTC day boundary")
	}
	var terminalProfiles int
	k.QueryRowTestDatabase(t.Context(), &terminalProfiles, `
		SELECT profiles
		FROM usage_metrics
		WHERE workspace = $1 AND day = $2`, workspace, deleteStartedAt.Format(time.DateOnly))
	if terminalProfiles != 0 {
		t.Fatalf("expected terminal profile count zero, got %d", terminalProfiles)
	}
	var profileSeconds int
	k.QueryRowTestDatabase(t.Context(), &profileSeconds, `
		SELECT profile_seconds
		FROM usage_metrics
		WHERE workspace = $1 AND day = $2`, workspace, deleteStartedAt.Format(time.DateOnly))
	dayUnix := deleteStartedAt.Truncate(24 * time.Hour).Unix()
	minimumProfileSeconds := 42 * int(deleteStartedAt.Unix()-dayUnix)
	maximumProfileSeconds := 42 * int(deleteFinishedAt.Unix()-dayUnix)
	if profileSeconds < minimumProfileSeconds || profileSeconds > maximumProfileSeconds {
		t.Fatalf("expected carried profile seconds in [%d,%d], got %d",
			minimumProfileSeconds, maximumProfileSeconds, profileSeconds)
	}
}

// TestUsageMetricsAggregatesBeforeRounding verifies that organization usage is
// combined before calculating the rounded profile average.
func TestUsageMetricsAggregatesBeforeRounding(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	var organization string
	k.QueryRowTestDatabase(t.Context(), &organization,
		"SELECT organization FROM workspaces WHERE id = $1", k.WorkspaceID())
	now := time.Now().UTC()
	day := now.AddDate(0, 0, -1).Format(time.DateOnly)
	// Each workspace averages 0.4 profiles and would round to zero on its own.
	// Their combined average is 0.8 and must therefore round to one.
	for i, workspace := range []string{"7QaT3mN7KxP5", "8QaT3mN7KxP5"} {
		k.ExecQueryTestDatabase(t.Context(), `
			INSERT INTO usage_metrics
				(organization, workspace, day, profiles, profile_seconds, observed_at, events)
			VALUES ($1, $2, $3, 0, $4, '12:00:00', $5)`,
			organization, workspace, day, 34_560, 3+i)
	}

	path := fmt.Sprintf("/v1/metrics/usage/dates/%s/%s", day, now.Format(time.DateOnly))
	var response usageMetricsResponse
	k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &response)
	metricDay := usageDay(t, response, day)
	if metricDay.Profiles != 0 {
		t.Fatalf("expected final organization profile count zero, got %d", metricDay.Profiles)
	}
	if metricDay.ProfilesAvg != 1 {
		t.Fatalf("expected organization average 1 after aggregating before rounding, got %d", metricDay.ProfilesAvg)
	}
	if metricDay.Events != 7 {
		t.Fatalf("expected 7 organization events, got %d", metricDay.Events)
	}
}

// assertUsageMetricsResponse verifies that a usage metrics response contains
// one series for the expected workspace and number of days.
func assertUsageMetricsResponse(t *testing.T, response usageMetricsResponse, workspace string, days int) {
	t.Helper()
	if len(response.Metrics) != 1 {
		t.Fatalf("expected one usage metric series, got %d", len(response.Metrics))
	}
	if response.Metrics[0].Workspace != workspace {
		t.Fatalf("expected workspace %q, got %q", workspace, response.Metrics[0].Workspace)
	}
	if len(response.Metrics[0].Days) != days {
		t.Fatalf("expected %d daily buckets, got %d", days, len(response.Metrics[0].Days))
	}
}

// usageDay returns the metrics for the given day or fails the test if no
// matching day exists.
func usageDay(t *testing.T, response usageMetricsResponse, day string) usageMetricDay {
	t.Helper()
	for _, metricDay := range response.Metrics[0].Days {
		if metricDay.Day == day {
			return metricDay
		}
	}
	t.Fatalf("expected usage metrics for day %s, got %#v", day, response)
	return usageMetricDay{}
}
