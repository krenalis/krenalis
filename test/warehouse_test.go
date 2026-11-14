// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestWarehouse(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	settings := krenalistester.PostgresWarehouseSettings()

	// Call the TestWorkspaceCreation method, checking that it returns the
	// error that the data warehouse cannot be initialized (because it already
	// contains database objects).
	profileSchema := types.Object([]types.Property{
		{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
	})
	err := k.TestWorkspaceCreation("PostgreSQL", profileSchema, "PostgreSQL", settings, krenalistester.Normal)
	var gotErr string
	if err != nil {
		gotErr = err.Error()
	}
	const expectedErr = `POST v1/workspaces/test: unexpected status code 422: {"error":{"code":"WarehouseNotInitializable","message":"cannot initialize the data warehouse: the database is not empty (contains 6 tables, 6 indexes, 2 views, 2 sequences)","cause":"the database is not empty (contains 6 tables, 6 indexes, 2 views, 2 sequences)"}} [request has body: true, response body expected: false]`
	if expectedErr != gotErr {
		t.Fatalf("expected error '%s', got '%s'", expectedErr, gotErr)
	}

	// The call to TestWarehouseUpdate should succeed, as the warehouse
	// being attempted to connect to is the same as the one currently connected
	// to.
	k.TestWarehouseUpdate(settings)

	// The call to UpdateWarehouse should also succeed, as the warehouse
	// to connect to is the same as the one currently connected to.
	k.UpdateWarehouse("Normal", settings)

	t.Run("no warehouse MCP credentials", func(t *testing.T) {
		var response map[string]json.Value
		k.Call("GET", "/v1/warehouse", nil, nil, &response)
		if _, present := response["mcpSettings"]; present {
			t.Fatal("expected no mcpSettings response field, got present")
		}
	})

	for _, endpoint := range []string{"/v1/warehouse/test", "/v1/warehouse"} {
		t.Run(endpoint+"/null", func(t *testing.T) {
			k.Call("PUT", endpoint, nil, map[string]any{
				"settings": settings, "mode": "Normal", "mcpSettings": nil,
			}, nil)
		})
		for _, test := range []struct {
			name  string
			value any
		}{
			{"object", map[string]any{"username": "retired"}},
			{"array", []any{}}, {"string", ""}, {"boolean", false}, {"number", 0},
		} {
			t.Run(endpoint+"/"+test.name, func(t *testing.T) {
				err := k.TryCall("PUT", endpoint, nil, map[string]any{
					"settings": settings, "mode": "Normal", "mcpSettings": test.value,
				}, nil)
				if err != nil {
					status, ok := err.(*krenalistester.StatusCodeError)
					if !ok || status.Response.Code != http.StatusBadRequest ||
						!strings.Contains(status.Response.Text, "mcpSettings has been removed") {
						t.Fatalf("expected removed-field HTTP 400, got %v", err)
					}
					return
				}
				t.Fatal("expected removed-field HTTP 400, got success")
			})
		}
	}

}
