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

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestExampleUserSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t, meergotester.DoNotPopulateUserSchema)
	defer c.Stop()

	f, err := os.Open("example_user_schema.json")
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

	queries := c.ChangeUserSchemaQueries(file.Schema, file.RePaths)
	if len(queries) != 6 {
		t.Fatalf("expected 6 queries, got %d", len(queries))
	}
	c.ChangeUserSchema(file.Schema, nil, file.RePaths)

}
