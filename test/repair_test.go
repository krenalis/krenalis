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

func TestRepair(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// The test of the warehouse connection and initialization is not performed
	// explicitly here, as it is already done implicitly by the 'InitAndLaunch'
	// function.

	// Disconnect the warehouse, then reconnect it, checking if it is correct.
	c.DisconnectWarehouse()
	c.ConnectWarehouse(meergotester.FailOnCheck)

	// Disconnect the warehouse, then reconnect it while repairing. Since there
	// should have been no database corruption in the meantime, this test checks
	// that the repair operation can be performed even if everything is already
	// correct.
	c.DisconnectWarehouse()
	c.ConnectWarehouse(meergotester.RepairWarehouse)

}
