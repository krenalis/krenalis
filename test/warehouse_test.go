// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func TestWarehouse(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	settings := krenalistester.PostgresWarehouseSettings()

	// Call the TestWorkspaceCreation method, checking that it returns the
	// error that the data warehouse cannot be initialized (because it already
	// contains database objects).
	profileSchema := types.Object([]types.Property{
		{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
	})
	err := c.TestWorkspaceCreation("PostgreSQL", profileSchema, krenalistester.UIPreferences{},
		"PostgreSQL", settings, krenalistester.Normal)
	var gotErr string
	if err != nil {
		gotErr = err.Error()
	}
	const expectedErr = `unexpected HTTP status code 422: {"error":{"code":"WarehouseNotInitializable","message":"cannot initialize the data warehouse: the database is not empty (contains 6 tables, 6 indexes, 2 views, 2 sequences)","cause":"the database is not empty (contains 6 tables, 6 indexes, 2 views, 2 sequences)"}}`
	if expectedErr != gotErr {
		t.Fatalf("expected error '%s', got '%s'", expectedErr, gotErr)
	}

	// The call to TestWarehouseUpdate should succeed, as the warehouse
	// being attempted to connect to is the same as the one currently connected
	// to.
	c.TestWarehouseUpdate(settings)

	// The call to UpdateWarehouse should also succeed, as the warehouse
	// to connect to is the same as the one currently connected to.
	c.UpdateWarehouse("Normal", settings)

}
