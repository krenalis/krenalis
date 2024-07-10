//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestImportFromDatabase(t *testing.T) {
	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}

	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	pgSQL := c.AddSourcePostgreSQL()

	importUsers := c.AddAction(pgSQL, "Users", chichitester.ActionToSet{
		Name: "Import users",
		InSchema: types.Object([]types.Property{
			{Name: "id", Type: types.Int(32), Required: true},
			{Name: "email", Type: types.Text(), Required: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		Query:                  `SELECT id, 'a@b' as "email", 'ABC123' as "customer_id" FROM members LIMIT ${limit}`,
		IdentityProperty:       "id",
		LastChangeTimeProperty: "",
		LastChangeTimeFormat:   "",
	})

	c.ExecuteAction(pgSQL, importUsers, false)

	c.WaitActionsToFinish(pgSQL)

	identities, count := c.ConnectionIdentities(pgSQL, 0, 100)

	const expectedCount = 1
	if count != expectedCount {
		t.Fatalf("expected %d identities, got %d", expectedCount, count)
	}

	for _, identity := range identities {
		if identity.Action != importUsers {
			t.Fatalf("expected identity action %d, got %d", importUsers, identity.Action)
		}
	}
}
