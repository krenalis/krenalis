//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestWarehouse(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	settings := meergotester.PostgresWarehouseSettings()

	// Call the TestWorkspaceCreation method, checking that it returns the
	// error that the data warehouse cannot be initialized (because it already
	// contains database objects).
	userSchema := types.Object([]types.Property{
		{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
	})
	err := c.TestWorkspaceCreation("PostgreSQL", userSchema, meergotester.DisplayedProperties{},
		"PostgreSQL", settings, meergotester.Normal)
	var gotErr string
	if err != nil {
		gotErr = err.Error()
	}
	const expectedErr = `unexpected HTTP status code 422: {"error":{"code":"WarehouseNonInitializable","message":"data warehouse is not initializable: database is not empty (it contains 1 view, 3 sequences, 5 indexes, 6 tables)","cause":"database is not empty (it contains 1 view, 3 sequences, 5 indexes, 6 tables)"}}`
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
