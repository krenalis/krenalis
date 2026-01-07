// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"testing"

	"github.com/meergo/meergo/core/internal/state"
)

func TestCompatibleMode(t *testing.T) {
	tests := []struct {
		modes    allowedMode
		mode     state.WarehouseMode
		expected bool
	}{
		{normalMode, state.Normal, true},
		{normalMode, state.Inspection, false},
		{maintenanceMode, state.Normal, false},
		{inspectionMode, state.Inspection, true},
		{maintenanceMode, state.Maintenance, true},
		{normalMode | inspectionMode, state.Inspection, true},
		{normalMode | inspectionMode, state.Maintenance, false},
		{normalMode | maintenanceMode, state.Inspection, false},
		{normalMode | maintenanceMode, state.Maintenance, true},
		{inspectionMode | maintenanceMode, state.Normal, false},
		{inspectionMode | maintenanceMode, state.Inspection, true},
		{normalMode | inspectionMode | maintenanceMode, state.Normal, true},
		{normalMode | inspectionMode | maintenanceMode, state.Inspection, true},
		{normalMode | inspectionMode | maintenanceMode, state.Maintenance, true},
	}

	for _, test := range tests {
		got := compatibleMode(test.modes, test.mode)
		if got != test.expected {
			t.Errorf("containsMode(%v, %v) = %v; expected %v", test.modes, test.mode, got, test.expected)
		}
	}
}

func TestModes(t *testing.T) {
	// Check that the number of elements in 'warehouseModes' is correct.
	if len(warehouseModes) != 3 {
		t.Fatalf("warehouseModes should have 3 elements, got %d", len(warehouseModes))
	}
	// Test 'anyMode'.
	for _, mode := range warehouseModes {
		if !compatibleMode(anyMode, mode) {
			t.Fatalf("the constant 'anyMode' should contain the mode %q, but it doesn't", mode)
		}
	}
}
