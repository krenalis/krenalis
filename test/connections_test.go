//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/test/meergotester"
)

func TestConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	wsID := c.WorkspaceID()

	// Ensure that there are no connections.
	var connections []any
	c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(wsID)+"/connections", nil, &connections)
	if len(connections) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(connections))
	}

	// Create a Dummy (source) connection.
	dummyID := c.CreateDummy("Dummy (source)", meergotester.Source)

	// Check if the Dummy connection has been created successfully.
	connections = nil
	c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(wsID)+"/connections", nil, &connections)
	if len(connections) != 1 {
		t.Fatalf("expected 1 connections, got %d", len(connections))
	}
	dummy := connections[0].(map[string]any)
	expectedName := "Dummy (source)"
	gotName := dummy["name"].(string)
	if expectedName != gotName {
		t.Fatalf("expected name %q, got %q", expectedName, gotName)
	}

	// Retrieve the input and the output schema, which must me both valid.
	schemas := c.ActionSchemas(dummyID, core.Users, "")
	if err := isValidSchema(schemas["in"]); err != nil {
		t.Fatal(err)
	}
	if err := isValidSchema(schemas["out"]); err != nil {
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
		return fmt.Errorf("expected name %q, got %q", "Object", name)
	}
	props := s["properties"].([]any)
	if len(props) == 0 {
		return fmt.Errorf("expected at least one property")
	}
	return nil
}
