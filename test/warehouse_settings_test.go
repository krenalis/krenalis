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
)

func TestWarehouseSettings(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	settings := meergotester.PostgresWarehouseSettings()

	// Call the CanInitializeWarehouse method, checking that it returns the
	// error that the data warehouse cannot be initialized (because it already
	// contains database objects).
	err := c.CanInitializeWarehouse("PostgreSQL", settings)
	var gotErr string
	if err != nil {
		gotErr = err.Error()
	}
	const expectedErr = `unexpected HTTP status code 422: {"error":{"code":"WarehouseNotInitializable","message":"data warehouse is not initializable: database contains these objects: table '_destinations_users', index '_destinations_users_pkey', table '_operations', sequence '_operations_id_seq', index '_operations_pkey', table '_user_identities', sequence '_user_identities___cluster___seq', sequence '_user_identities___pk___seq', index '_user_identities_pkey', table '_users_0', table 'events', index 'events_pkey', view 'users'","cause":"data warehouse is not initializable: database contains these objects: table '_destinations_users', index '_destinations_users_pkey', table '_operations', sequence '_operations_id_seq', index '_operations_pkey', table '_user_identities', sequence '_user_identities___cluster___seq', sequence '_user_identities___pk___seq', index '_user_identities_pkey', table '_users_0', table 'events', index 'events_pkey', view 'users'"}}`
	if expectedErr != gotErr {
		t.Fatalf("expected error '%s', got '%s'", expectedErr, gotErr)
	}

	// The call to CanChangeWarehouseSettings should succeed, as the warehouse
	// being attempted to connect to is the same as the one currently connected
	// to.
	c.CanChangeWarehouseSettings(settings)

	// The call to ChangeWarehouseSettings should also succeed, as the warehouse
	// to connect to is the same as the one currently connected to.
	c.ChangeWarehouseSettings("Normal", settings)

}
