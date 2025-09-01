//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
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

	importUsers := c.CreateAction(pgSQL, "User", meergotester.ActionToSet{
		Name:    "Import users",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "id", Type: types.Int(32), Nullable: true},
			{Name: "email", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
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

	exec := c.ExecuteAction(importUsers)

	c.WaitForExecutionsCompletion(pgSQL, exec)

	identities, total := c.ConnectionIdentities(pgSQL, 0, 100)

	const expectedCount = 1
	if total != expectedCount {
		t.Fatalf("expected %d identities, got %d", expectedCount, total)
	}

	for _, identity := range identities {
		if identity.Action != importUsers {
			t.Fatalf("expected identity action %d, got %d", importUsers, identity.Action)
		}
	}
}
