// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestWarehouse(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	settings := meergotester.PostgresWarehouseSettings()

	// Call the TestWorkspaceCreation method, checking that it returns the
	// error that the data warehouse cannot be initialized (because it already
	// contains database objects).
	profileSchema := types.Object([]types.Property{
		{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
	})
	err := c.TestWorkspaceCreation("PostgreSQL", profileSchema, meergotester.UIPreferences{},
		"PostgreSQL", settings, meergotester.Normal)
	var gotErr string
	if err != nil {
		gotErr = err.Error()
	}
	const expectedErr = `unexpected HTTP status code 422: {"error":{"code":"WarehouseNotInitializable","message":"data warehouse is not initializable: database is not empty (it contains 2 sequences, 2 views, 6 indexes, 6 tables)","cause":"database is not empty (it contains 2 sequences, 2 views, 6 indexes, 6 tables)"}}`
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
