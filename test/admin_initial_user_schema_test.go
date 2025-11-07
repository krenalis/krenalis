// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

// TestAdminInitialUserSchema tests the correctness of the user schema that is
// initially created when a workspace is created through the Admin.
func TestAdminInitialUserSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.PopulateUserSchema(false)
	c.Start()
	defer c.Stop()

	f, err := os.Open(filepath.Join("..", "admin/src/components/routes/WorkspaceCreate/InitialSchema.json"))
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
