// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"fmt"
	"testing"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
)

func TestConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	// Ensure that there are no connections.
	var res struct {
		Connections []any
	}
	c.MustCall("GET", "/v1/connections", nil, nil, &res)
	if len(res.Connections) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(res.Connections))
	}

	// Create a Dummy (source) connection.
	dummyID := c.CreateDummy("Dummy (source)", krenalistester.Source)

	// Check if the Dummy connection has been created successfully.
	res.Connections = nil
	c.MustCall("GET", "/v1/connections", nil, nil, &res)
	if len(res.Connections) != 1 {
		t.Fatalf("expected 1 connections, got %d", len(res.Connections))
	}
	dummy := res.Connections[0].(map[string]any)
	expectedName := "Dummy (source)"
	gotName := dummy["name"].(string)
	if expectedName != gotName {
		t.Fatalf("expected name %q, got %q", expectedName, gotName)
	}

	// Retrieve the input and the output schema, which must be both valid.
	schemas := c.PipelineSchemas(dummyID, core.TargetUser, "")
	if err := isValidSchema(schemas["in"]); err != nil {
		t.Fatal(err)
	}
	if err := isValidSchema(schemas["out"]); err != nil {
		t.Fatal(err)
	}

	// Check that a message broker connection cannot be created.
	broker := &krenalistester.ConnectionToCreate{
		Name:      "Kafka",
		Role:      krenalistester.Source,
		Connector: "kafka",
	}
	var id int
	err := c.Call("POST", "/v1/connections", nil, broker, &id)
	if err == nil {
		t.Fatalf("expected Bad Request error, got no error")
	}
	errStatusCode, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError error, got %T error", err)
	}
	if errStatusCode.Code != 400 {
		t.Fatalf("expected error status 400, got error: %v", errStatusCode)
	}
	const expectedText = `{"error":{"code":"BadRequest","message":"message broker connectors are not currently supported"}}`
	if expectedText != errStatusCode.ResponseText {
		t.Fatalf("expected error text %q, got %q", expectedText, errStatusCode.ResponseText)
	}

}

func isValidSchema(schema any) error {
	s, ok := schema.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected type %T", schema)
	}
	kind, ok := s["kind"]
	if !ok {
		return fmt.Errorf("expected type kind, got no kind")
	}
	if kind != "object" {
		return fmt.Errorf("expected type kind %q, got %q", "object", kind)
	}
	props := s["properties"].([]any)
	if len(props) == 0 {
		return fmt.Errorf("expected at least one property")
	}
	return nil
}
