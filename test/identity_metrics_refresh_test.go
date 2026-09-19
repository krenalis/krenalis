// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

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
		if err := json.Unmarshal(settingsJSON, &settings); err != nil {
			t.Fatal(err)
		}
		pool, err := krenalistester.ConnectionPool(t.Context(), &settings)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()
		if _, err := pool.Exec(t.Context(), `DROP TABLE "krenalis_identities"`); err != nil {
			t.Fatal(err)
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
