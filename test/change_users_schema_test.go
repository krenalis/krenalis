//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestChangeUsersSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Read the schema in "users_schema.json".
	f, err := os.Open("users_schema.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var file struct {
		Schema  types.Type
		RePaths map[string]any
	}
	err = dec.Decode(&file)
	if err != nil {
		t.Fatal(err)
	}

	// The schema of "users_schema.json" has already been applied by the tests
	// framework, so there should be no queries to execute.
	queries := c.ChangeUsersSchemaQueries(file.Schema, file.RePaths)
	if len(queries) > 0 {
		t.Fatalf("expected 0 queries, got %d", len(queries))
	}
	c.ChangeUsersSchema(file.Schema, file.RePaths) // this should do nothing.

	// Add a single property.
	schema := types.Object(append(file.Schema.Properties(), types.Property{
		Name: "newProp", Type: types.Text(),
	}))
	queries = c.ChangeUsersSchemaQueries(schema, nil)
	expectedQueries := []string{
		"BEGIN;",
		"ALTER TABLE \"users\"\n\tADD COLUMN \"new_prop\" varchar NOT NULL DEFAULT '';",
		"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"new_prop\" varchar NOT NULL DEFAULT '';",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %v, got %v", expectedQueries, queries)
	}
	c.ChangeUsersSchema(schema, nil)

}
