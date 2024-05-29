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
	"slices"
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestChangeUsersSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	ws := c.Workspace()
	if n := len(types.Properties(ws.UsersSchema)); n != 11 {
		t.Fatalf("expected 11 properties in the \"users\" schema, got %d", n)
	}

	// Read the schema in "tests_users_schema.json".
	f, err := os.Open("tests_users_schema.json")
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

	// The schema of "tests_users_schema.json" has already been applied by the
	// tests framework.
	queries := c.ChangeUsersSchemaQueries(file.Schema, file.RePaths)
	if len(queries) != 6 {
		t.Fatalf("expected 6 queries, got %d", len(queries))
	}
	c.ChangeUsersSchema(file.Schema, file.RePaths) // this should do nothing.

	ws = c.Workspace()
	if n := len(types.Properties(ws.UsersSchema)); n != 11 {
		t.Fatalf("expected 11 properties in the \"users\" schema, got %d", n)
	}

	// Add a single property.
	schema := types.Object(append(types.Properties(file.Schema), types.Property{
		Name: "new_prop", Type: types.Text(), Nullable: true,
	}))
	queries = c.ChangeUsersSchemaQueries(schema, nil)
	expectedQueries := []string{"BEGIN;",
		"DROP VIEW \"users\";",
		"DROP VIEW \"users_identities\";",
		"ALTER TABLE \"_users\"\n\tADD COLUMN \"new_prop\" varchar;",
		"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"new_prop\" varchar;",
		"CREATE VIEW \"users\" AS SELECT\n\t\"__id__\",\n\t\"email\",\n\t\"dummy_id\",\n" +
			"\t\"android_id\",\n\t\"android_idfa\",\n\t\"android_push_token\",\n" +
			"\t\"ios_id\",\n\t\"ios_idfa\",\n\t\"ios_push_token\",\n\t\"first_name\",\n" +
			"\t\"last_name\",\n\t\"gender\",\n\t\"food_preferences_drink\",\n" +
			"\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n\t\"favorite_movie_title\",\n" +
			"\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n" +
			"\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n" +
			"\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"_users\";",
		"CREATE VIEW \"users_identities\" AS SELECT\n\t\"__pk__\",\n\t\"__connection__\",\n" +
			"\t\"__identity_id__\",\n\t\"__is_anonymous__\",\n\t\"__displayed_property__\",\n\t\"__anonymous_ids__\",\n" +
			"\t\"__last_change_time__\",\n\t\"__gid__\",\n\t\"email\",\n\t\"dummy_id\",\n\t\"android_id\",\n" +
			"\t\"android_idfa\",\n\t\"android_push_token\",\n\t\"ios_id\",\n\t\"ios_idfa\",\n" +
			"\t\"ios_push_token\",\n\t\"first_name\",\n\t\"last_name\",\n\t\"gender\",\n" +
			"\t\"food_preferences_drink\",\n\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n" +
			"\t\"favorite_movie_title\",\n\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n" +
			"\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n" +
			"\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"_users_identities\";",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %#v, got %#v", expectedQueries, queries)
	}
	c.ChangeUsersSchema(schema, nil)

	ws = c.Workspace()
	if n := len(types.Properties(ws.UsersSchema)); n != 12 {
		t.Fatalf("expected 12 properties in the \"users\" schema, got %d", n)
	}

	// Create a schema with two properties that would conflict each other.
	schema = types.Object(append(types.Properties(file.Schema),
		types.Property{Name: "a_b", Type: types.Text(), Nullable: true},
		types.Property{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: types.Text(), Nullable: true},
		})},
	))
	_, err = c.ChangeUsersSchemaQueriesErr(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedErr := `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"schema contains conflicting properties: two or more properties cannot have the same representation as column \"a_b\""}}`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
	err = c.ChangeUsersSchemaErr(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Create a schema with a non-null property.
	schema = types.Object(append(types.Properties(file.Schema),
		types.Property{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: types.Text(), Nullable: false},
		})},
	))
	_, err = c.ChangeUsersSchemaQueriesErr(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedErr = `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"not allowed property in schema: property with type Text must be nullable"}}`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
	err = c.ChangeUsersSchemaErr(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

}
