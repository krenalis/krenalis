// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"errors"
	"os"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// Test_QueryReadOnly_rejectsBeforeOpeningConnection verifies validation runs
// before Snowflake connection setup.
func Test_QueryReadOnly_rejectsBeforeOpeningConnection(t *testing.T) {
	wh := &Snowflake{}

	rows, columnCount, err := wh.QueryReadOnly(t.Context(), "DELETE FROM t")
	if rows != nil {
		t.Fatalf("expected nil rows, got %#v", rows)
	}
	if columnCount != 0 {
		t.Fatalf("expected column count 0, got %d", columnCount)
	}
	var target *warehouses.RejectedReadOnlyQueryError
	if !errors.As(err, &target) {
		t.Fatalf("expected warehouses.RejectedReadOnlyQueryError, got %T (%v)", err, err)
	}
	if target.Function != "" {
		t.Fatalf("expected rejected function name %q, got %q", "", target.Function)
	}
}

// Test_QueryReadOnly_live verifies QueryReadOnly against a Snowflake warehouse.
func Test_QueryReadOnly_live(t *testing.T) {
	settingsFile, ok := os.LookupEnv(settingsEnvKey)
	if !ok {
		t.Skipf("the %s environment variable is not present", settingsEnvKey)
	}

	settings, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("expected to read settings from %q, got %s", settingsFile, err)
	}
	wh := New(newTestSettingsLoader(json.Value(settings)))
	defer wh.Close()

	rows, columnCount, err := wh.QueryReadOnly(t.Context(), "SELECT 1")
	if err != nil {
		t.Fatalf("expected QueryReadOnly to accept SELECT 1, got %s", err)
	}
	defer rows.Close()

	if columnCount != 1 {
		t.Fatalf("expected column count 1, got %d", columnCount)
	}
	if !rows.Next() {
		t.Fatal("expected one row, got no rows")
	}
	var got int
	if err := rows.Scan(&got); err != nil {
		t.Fatalf("expected to scan one integer, got %s", err)
	}
	if got != 1 {
		t.Fatalf("expected value 1, got %d", got)
	}
	if rows.Next() {
		t.Fatal("expected one row, got more rows")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("expected rows without error, got %s", err)
	}
}
