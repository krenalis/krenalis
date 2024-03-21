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
	"testing"

	"chichi/test/chichitester"
	"chichi/types"
)

func TestExampleUsersSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t, chichitester.DoNotPopulateUsersSchema)
	defer c.Stop()

	f, err := os.Open("example_users_schema.json")
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

	queries := c.ChangeUsersSchemaQueries(file.Schema, file.RePaths)
	if len(queries) != 4 {
		t.Fatalf("expected 4 queries, got %d", len(queries))
	}
	c.ChangeUsersSchema(file.Schema, file.RePaths)

}
