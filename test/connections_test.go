//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"fmt"
	"testing"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/test/chichitester"
)

func TestConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Ensure that there are no connections.
	var connections []any
	c.MustCall("GET", "/api/workspaces/1/connections", nil, &connections)
	if len(connections) != 0 {
		t.Fatalf("expecting 0 connections, got %d", len(connections))
	}

	// Create a Dummy (source) connection.
	dummyID := c.AddDummy("Dummy (source)", chichitester.Source)

	// Check if the Dummy connection has been created successfully.
	connections = nil
	c.MustCall("GET", "/api/workspaces/1/connections", nil, &connections)
	if len(connections) != 1 {
		t.Fatalf("expecting 1 connections, got %d", len(connections))
	}
	dummy := connections[0].(map[string]any)
	expectedName := "Dummy (source)"
	gotName := dummy["Name"].(string)
	if expectedName != gotName {
		t.Fatalf("expecting name %q, got %q", expectedName, gotName)
	}

	// Retrieve the input and the output schema, which must me both valid.
	schemas := c.ActionSchemas(dummyID, apis.Users, "")
	if err := isValidSchema(schemas["In"]); err != nil {
		t.Fatal(err)
	}
	if err := isValidSchema(schemas["Out"]); err != nil {
		t.Fatal(err)
	}

}

func isValidSchema(schema any) error {
	s, ok := schema.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected type %T", schema)
	}
	name := s["name"]
	if name != "Object" {
		return fmt.Errorf("expecting name %q, got %q", "Object", name)
	}
	props := s["properties"].([]any)
	if len(props) == 0 {
		return fmt.Errorf("expecting at least one property")
	}
	return nil
}
