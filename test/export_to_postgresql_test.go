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
		dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
		importUsersID := c.CreateAction(dummySrc, "Users", meergotester.ActionToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "firstName", Type: types.Text(), Nullable: true},
				{Name: "lastName", Type: types.Text(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "gender", Type: types.Text(), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":      "coalesce(email, 'default.email@example.com')",
					"first_name": "firstName",
					"last_name":  "lastName",
					"gender":     "'male'",
				},
			},
		})
		exec := c.ExecuteAction(importUsersID)
		c.WaitForExecutionsCompletion(dummySrc, exec)
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

	pgsql := c.CreateDestinationPostgreSQL()

	// Check if the schema is correct.
	{
		schema := c.TableSchema(pgsql, "test_export_to_db")
		expectedSchema := types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "full_name", Type: types.Text()},
		})
		if !types.Equal(expectedSchema, schema) {
			t.Fatalf("\nexpected:  %#v\ngot:        %#v", types.Properties(expectedSchema), types.Properties(schema))
		}
	}

	// Export to PostgreSQL.
	exportAction := c.CreateAction(pgsql, "Users", meergotester.ActionToSet{
		Name:      "Export users to PostgreSQL",
		Enabled:   true,
		TableName: "test_export_to_db",
		TableKey:  "email",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), CreateRequired: true},
			{Name: "full_name", Type: types.Text()},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"full_name": `first_name " " last_name`,
			},
		},
	})
	exec := c.ExecuteAction(exportAction)
	c.WaitForExecutionsCompletion(pgsql, exec)

	// Check if the export completed successfully.
	const expectedCount = 10
	var count int
	c.QueryRowTestDatabase(ctx, &count,
		`SELECT COUNT(*) FROM test_export_to_db WHERE email <> '' AND full_name <> ''`,
	)
	if expectedCount != count {
		t.Fatalf("expected count %d, got %d", expectedCount, count)
	}

	// Update the action to export the empty string for full_name.
	c.UpdateAction(exportAction, meergotester.ActionToSet{
		Name:      "Export users to PostgreSQL",
		Enabled:   true,
		TableName: "test_export_to_db",
		TableKey:  "email",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), CreateRequired: true},
			{Name: "full_name", Type: types.Text()},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"full_name": `""`,
			},
		},
	})
	exec = c.ExecuteAction(exportAction)
	c.WaitForExecutionsCompletion(pgsql, exec)

	// Check if the export completed successfully.
	c.QueryRowTestDatabase(ctx, &count,
		`SELECT COUNT(*) FROM test_export_to_db WHERE email <> '' AND full_name = ''`,
	)
	if expectedCount != count {
		t.Fatalf("expected count %d, got %d", expectedCount, count)
	}

}
