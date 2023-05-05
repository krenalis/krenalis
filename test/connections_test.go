//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"testing"

	"chichi/test/chichitester"
)

func TestConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Ensure that there are no connections.
	connections := c.MustCall("GET", "/api/connections", nil).([]any)
	if len(connections) != 0 {
		t.Fatalf("expecting 0 connections, got %d", len(connections))
	}

	// Create a Dummy (source) connection.
	c.AddConnection(map[string]any{
		"Connector": 3, // Dummy.
		"Role":      "Source",
		"Options": map[string]any{
			"Name":    "Dummy (source)",
			"Enabled": true,
		},
	})

	// Check if the Dummy connection has been created successfully.
	connections = c.MustCall("GET", "/api/connections", nil).([]any)
	if len(connections) != 1 {
		t.Fatalf("expecting 1 connections, got %d", len(connections))
	}
	dummy := connections[0].(map[string]any)
	expectedName := "Dummy (source)"
	gotName := dummy["Name"].(string)
	if expectedName != gotName {
		t.Fatalf("expecting name %q, got %q", expectedName, gotName)
	}

}
