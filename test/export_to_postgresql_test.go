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

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestExportToPostgreSQL(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Load some users in the data warehouse.
	{
		dummySrc := c.AddDummy("Dummy (source)", meergotester.Source)
		importUsersID := c.AddAction(dummySrc, "Users", meergotester.ActionToSet{
			Name: "Import users from Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "first_name", Type: types.Text()},
				{Name: "last_name", Type: types.Text()},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
			}),
			Transformation: meergotester.Transformation{
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
				email text NOT NULL DEFAULT '',
				full_name text NOT NULL DEFAULT '',
				PRIMARY KEY (email)
			)
		`)

	pgsql := c.AddDestinationPostgreSQL()

	// Check if the schema is correct.
	{
		schema := c.TableSchema(pgsql, "test_export_to_db")
		expectedSchema := types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Required: true, Nullable: false},
			{Name: "full_name", Type: types.Text(), Required: true, Nullable: false},
		})
		if !types.Equal(expectedSchema, schema) {
			t.Fatalf("\nexpecting:  %#v\ngot:        %#v", types.Properties(expectedSchema), types.Properties(schema))
		}
	}

	// Export to PostgreSQL.
	exportAction := c.AddAction(pgsql, "Users", meergotester.ActionToSet{
		Name:             "Export users to PostgreSQL",
		TableName:        "test_export_to_db",
		TableKeyProperty: "email",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "first_name", Type: types.Text()},
			{Name: "last_name", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Required: true},
			{Name: "full_name", Type: types.Text(), Required: true},
		}),
		Transformation: meergotester.Transformation{
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

	// Change the action to export the empty string for full_name.
	c.SetAction(pgsql, exportAction, meergotester.ActionToSet{
		Name:             "Export users to PostgreSQL",
		TableName:        "test_export_to_db",
		TableKeyProperty: "email",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Required: true},
			{Name: "full_name", Type: types.Text(), Required: true},
		}),
		Transformation: meergotester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"full_name": `""`,
			},
		},
	})
	c.ExecuteAction(pgsql, exportAction, false)
	c.WaitActionsToFinish(pgsql)

	// Check if the export completed successfully.
	c.QueryRowTestDatabase(ctx, &count,
		`SELECT COUNT(*) FROM test_export_to_db WHERE email <> '' AND full_name = ''`,
	)
	if expectedCount != count {
		t.Fatalf("expecting count %d, got %d", expectedCount, count)
	}

}
