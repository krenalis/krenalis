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
	"path/filepath"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

// TestUIInitialUserSchema tests the correctness of the user schema that is
// initially created when a workspace is created through the UI.
func TestUIInitialUserSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t, meergotester.DoNotPopulateUserSchema)
	defer c.Stop()

	f, err := os.Open(filepath.Join("..", "assets/src/components/routes/WorkspaceCreate/InitialSchema.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var schema types.Type
	err = dec.Decode(&schema)
	if err != nil {
		t.Fatal(err)
	}

	queries := c.PreviewAlterUserSchema(schema, nil)
	const expectedQueriesCount = 6
	if len(queries) != expectedQueriesCount {
		t.Fatalf("expected %d queries, got %d", expectedQueriesCount, len(queries))
	}
	c.AlterUserSchema(schema, nil, nil)

}
