// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/krenalis/krenalis/test/meergotester"
)

func TestRepair(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// This test verifies that the Repair operation can be run multiple times on
	// a PostgreSQL data warehouse without any error.
	for range 3 {
		c.RepairWarehouse()
	}

}
