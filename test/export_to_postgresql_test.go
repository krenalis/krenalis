//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"context"
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestExportToPostgreSQL(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Load some users in the data warehouse.
	{
		dummySrc := c.AddDummy("Dummy (source)", chichitester.Source)
		importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
			Name: "Import users from Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "first_name", Type: types.Text(), Nullable: true},
				{Name: "last_name", Type: types.Text(), Nullable: true},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other"), Nullable: true},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":      "coalesce(email, 'default.email@example.com')",
					"first_name": "firstName",
					"last_name":  "lastName",
					"gender":     "'male'",
				},
			},
		})
		c.ExecuteAction(dummySrc, importUsersID, true)
		c.WaitActionsToFinish(dummySrc)
	}

	ctx := context.Background()

	c.ExecQueryTestDatabase(ctx, `
		CREATE TABLE test_export_to_db
			(
				id uuid,
				email text NOT NULL DEFAULT '',
				full_name text NOT NULL DEFAULT '',
				PRIMARY KEY (id)
			)
		`)

	pgsql := c.AddDestinationPostgreSQL()

	// Check if the schema is correct.
	{
		schema := c.TableSchema(pgsql, "test_export_to_db")
		expectedSchema := types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: false},
			{Name: "full_name", Type: types.Text(), Nullable: false},
		})
		if !types.Equal(expectedSchema, schema) {
			t.Fatalf("\nexpecting:  %#v\ngot:        %#v", types.Properties(expectedSchema), types.Properties(schema))
		}
	}

	// Export to PostgreSQL.
	exportAction := c.AddAction(pgsql, "Users", chichitester.ActionToSet{
		Name:      "Export users to PostgreSQL",
		TableName: "test_export_to_db",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "first_name", Type: types.Text(), Nullable: true},
			{Name: "last_name", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "full_name", Type: types.Text()},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"full_name": `first_name " " last_name`,
			},
		},
	})
	c.ExecuteAction(pgsql, exportAction, false)
	c.WaitActionsToFinish(pgsql)

	// Check if the export completed successfully.
	const expectedCount = 10
	var count int
	c.QueryRowTestDatabase(ctx, &count,
		`SELECT COUNT(*) FROM test_export_to_db WHERE email <> '' AND full_name <> ''`,
	)
	if expectedCount != count {
		t.Fatalf("expecting count %d, got %d", expectedCount, count)
	}

}
