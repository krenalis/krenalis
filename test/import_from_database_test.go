// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestImportFromDatabase(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	pgSQL := c.CreateSourcePostgreSQL()

	importUsers := c.CreatePipeline(pgSQL, "User", meergotester.PipelineToSet{
		Name:    "Import users",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "id", Type: types.Int(32), Nullable: true},
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		Query:                `SELECT id, 'a@b' as "email", 'ABC123' as "customer_id" FROM members LIMIT ${limit}`,
		IdentityColumn:       "id",
		LastChangeTimeColumn: "",
		LastChangeTimeFormat: "",
	})

	exec := c.ExecutePipeline(importUsers)

	c.WaitForExecutionsCompletion(pgSQL, exec)

	identities, total := c.ConnectionIdentities(pgSQL, 0, 100)

	const expectedCount = 1
	if total != expectedCount {
		t.Fatalf("expected %d identities, got %d", expectedCount, total)
	}

	for _, identity := range identities {
		if identity.Pipeline != importUsers {
			t.Fatalf("expected identity pipeline %d, got %d", importUsers, identity.Pipeline)
		}
	}
}
