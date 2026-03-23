// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestAdminInitialProfileSchema tests the correctness of the profile schema that is
// initially created when a workspace is created through the Admin.
func TestAdminInitialProfileSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewMeergoInstance(t)
	c.PopulateProfileSchema(false)
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

	queries := c.PreviewAlterProfileSchema(schema, nil)
	const expectedQueriesCount = 6
	if len(queries) != expectedQueriesCount {
		t.Fatalf("expected %d queries, got %d", expectedQueriesCount, len(queries))
	}
	c.AlterProfileSchema(schema, nil, nil)

}
