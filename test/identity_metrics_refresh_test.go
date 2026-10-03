// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestIdentityMetricsRefreshAfterImport verifies automatic and explicit refresh
// and reads the resulting current and daily metrics through the API.
func TestIdentityMetricsRefreshAfterImport(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()
	connection := k.CreateDummy("Identity metrics source", krenalistester.Source)
	pipeline := k.CreatePipeline(connection, "User", krenalistester.PipelineToSet{
		Name:    "Identity metrics pipeline",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{Mapping: map[string]string{"email": "email"}},
	})
	var initial core.IdentityMetric
	k.Call("GET", "/v1/metrics/identities/latest", nil, nil, &initial)
	if initial.Total != 0 {
		t.Fatalf("expected zero bootstrap identities, got %d", initial.Total)
	}
	k.WaitForRunsCompletion(k.StartPipelineRun(pipeline))

	// Run completion precedes the automatic metrics refresh.
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var latest core.IdentityMetric
	for {
		k.Call("GET", "/v1/metrics/identities/latest", nil, nil, &latest)
		if latest.Total == 10 {
			break
		}
		select {
		case <-t.Context().Done():
			t.Fatalf("expected automatic refresh, got %v", t.Context().Err())
		case <-timer.C:
			t.Fatalf("expected 10 identities after import, got %d", latest.Total)
		case <-ticker.C:
		}
	}
	if len(latest.Connections) != 1 || latest.Connections[0].Connection != connection ||
		latest.Connections[0].Anonymous+latest.Connections[0].Recognized != 10 {
		t.Fatalf("expected one connection with 10 identities, got %#v", latest.Connections)
	}

	k.Call("POST", "/v1/metrics/identities/refresh", nil, nil, nil)
	k.Call("GET", "/v1/metrics/identities/latest", nil, nil, &latest)
	if latest.Total != 10 {
		t.Fatalf("expected 10 identities after explicit refresh, got %d", latest.Total)
	}
	start := latest.ObservedAt.UTC().Truncate(24 * time.Hour)
	path := "/v1/metrics/identities/dates/" + start.Format(time.DateOnly) + "/" + start.AddDate(0, 0, 1).Format(time.DateOnly)
	for _, selection := range []string{"", "?connection=" + connection} {
		var days []core.IdentityMetricDay
		k.Call("GET", path+selection, nil, nil, &days)
		if len(days) != 1 || days[0].Day != start.Format(time.DateOnly) || days[0].Total != 10 {
			t.Fatalf("expected one observed day with 10 identities, got %#v", days)
		}
	}

}

// TestIdentityMetricsRefreshReturnsOperationalErrors verifies that maintenance
// mode and warehouse unavailability are exposed as the expected API errors.
func TestIdentityMetricsRefreshReturnsOperationalErrors(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	connection := k.CreateDummy("Identity metrics source", krenalistester.Source)
	k.CreatePipeline(connection, "User", krenalistester.PipelineToSet{
		Name: "Identity metrics pipeline",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{"email": "email"},
		},
	})

	t.Run("maintenance mode", func(t *testing.T) {

		k.Call("PUT", "/v1/warehouse/mode", nil, map[string]any{
			"mode":                         krenalistester.Maintenance,
			"cancelIncompatibleOperations": false,
		}, nil)

		err := k.TryCall("POST", "/v1/metrics/identities/refresh", nil, nil, nil)
		if err != nil {
			statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
			if !ok {
				t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
			}
			if statusErr.Response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected HTTP status %d, got %d: %s",
					http.StatusUnprocessableEntity, statusErr.Response.Code, statusErr.Response.Text)
			}
			const expected = `{"error":{"code":"MaintenanceMode","message":"data warehouse is in maintenance mode"}}`
			if statusErr.Response.Text != expected {
				t.Fatalf("expected response %s, got %s", expected, statusErr.Response.Text)
			}
			return
		}
		t.Fatal("expected a maintenance mode error, got nil")

	})

	k.Call("PUT", "/v1/warehouse/mode", nil, map[string]any{
		"mode":                         krenalistester.Normal,
		"cancelIncompatibleOperations": false,
	}, nil)

	t.Run("unavailable warehouse", func(t *testing.T) {

		settingsJSON := krenalistester.PostgresWarehouseSettings()
		var settings krenalistester.DBSettings
		err := json.Unmarshal(settingsJSON, &settings)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		pool, err := krenalistester.ConnectionPool(t.Context(), &settings)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer pool.Close()
		_, err = pool.Exec(t.Context(), `DROP TABLE "krenalis_identities"`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		err = k.TryCall("POST", "/v1/metrics/identities/refresh", nil, nil, nil)
		if err != nil {
			statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
			if !ok {
				t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
			}
			if statusErr.Response.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected HTTP status %d, got %d: %s",
					http.StatusServiceUnavailable, statusErr.Response.Code, statusErr.Response.Text)
			}
			const expected = `{"error":{"code":"ServiceUnavailable","message":"data warehouse is unavailable"}}`
			if statusErr.Response.Text != expected {
				t.Fatalf("expected response %s, got %s", expected, statusErr.Response.Text)
			}
			return
		}
		t.Fatal("expected a warehouse unavailable error, got nil")

	})

}
