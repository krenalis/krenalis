// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestChangeProfileSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	ws := c.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 10 {
		t.Fatalf("expected 10 properties in the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}

	identifiers := []string{"email", "android.id"}
	c.UpdateIdentityResolution(true, identifiers)

	// Read the schema in "testdata/change_profile_schema_test.json".
	f, err := os.Open("testdata/change_profile_schema_test.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var file struct {
		Schema         types.Type
		PrimarySources map[string]int
		RePaths        map[string]any
	}
	err = dec.Decode(&file)
	if err != nil {
		t.Fatal(err)
	}

	// Alter the profile schema.
	queries := c.PreviewAlterProfileSchema(file.Schema, file.RePaths)
	if len(queries) != 4 {
		t.Fatalf("expected 4 queries, got %d", len(queries))
	}
	c.AlterProfileSchema(file.Schema, file.PrimarySources, file.RePaths)

	ws = c.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 10 {
		t.Fatalf("expected 10 properties in the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}
	if !slices.Equal(identifiers, ws.Identifiers) {
		t.Fatalf("expected identifiers %v, got %v", identifiers, ws.Identifiers)
	}

	// Add a single property.
	schema := types.Object(append(file.Schema.Properties().Slice(), types.Property{
		Name: "new_prop", Type: types.Text(), ReadOptional: true,
	}))
	queries = c.PreviewAlterProfileSchema(schema, nil)
	expectedQueries := []string{"BEGIN;",
		"DROP VIEW \"profiles\";",
		"ALTER TABLE \"meergo_profiles_0\"\n\tADD COLUMN \"new_prop\" character varying;",
		"ALTER TABLE \"meergo_identities\"\n\tADD COLUMN \"new_prop\" character varying;",
		"CREATE VIEW \"profiles\" AS SELECT\n\t\"__mpid__\",\n\t\"__last_change_time__\",\n\t\"email\",\n\t\"dummy_id\",\n\t\"android_id\",\n\t\"android_idfa\",\n\t\"android_push_token\",\n\t\"ios_id\",\n\t\"ios_idfa\",\n\t\"ios_push_token\",\n\t\"first_name\",\n\t\"last_name\",\n\t\"gender\",\n\t\"food_preferences_drink\",\n\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n\t\"favorite_movie_title\",\n\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"meergo_profiles_0\";",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %#v, got %#v", expectedQueries, queries)
	}
	c.AlterProfileSchema(schema, nil, nil)

	ws = c.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 11 {
		t.Fatalf("expected 11 properties in the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}
	if !slices.Equal(identifiers, ws.Identifiers) {
		t.Fatalf("expected identifiers %v, got %v", identifiers, ws.Identifiers)
	}

	// Rename the property "android.id" to "android.identifier" and drop "email".
	var properties []types.Property
	for _, p := range schema.Properties().All() {
		switch p.Name {
		case "email":
			continue
		case "android":
			props := p.Type.Properties().Slice()
			for i := 0; i < len(props); i++ {
				if props[i].Name == "id" {
					props[i].Name = "identifier"
					break
				}
			}
			p.Type = types.Object(props)
		}
		properties = append(properties, p)
	}
	schema = types.Object(properties)
	rePaths := map[string]any{"android.identifier": "android.id"}
	queries = c.PreviewAlterProfileSchema(schema, rePaths)
	expectedQueries = []string{
		"BEGIN;",
		"DROP VIEW \"profiles\";", "ALTER TABLE \"meergo_profiles_0\"\n\tDROP COLUMN \"email\";",
		"ALTER TABLE \"meergo_identities\"\n\tDROP COLUMN \"email\";",
		"ALTER TABLE \"meergo_profiles_0\"\n\tRENAME COLUMN \"android_id\" TO \"android_identifier\";",
		"ALTER TABLE \"meergo_identities\"\n\tRENAME COLUMN \"android_id\" TO \"android_identifier\";",
		"CREATE VIEW \"profiles\" AS SELECT\n\t\"__mpid__\",\n\t\"__last_change_time__\",\n\t\"dummy_id\",\n\t\"android_identifier\",\n\t\"android_idfa\",\n\t\"android_push_token\",\n\t\"ios_id\",\n\t\"ios_idfa\",\n\t\"ios_push_token\",\n\t\"first_name\",\n\t\"last_name\",\n\t\"gender\",\n\t\"food_preferences_drink\",\n\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n\t\"favorite_movie_title\",\n\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"meergo_profiles_0\";",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %#v, got %#v", expectedQueries, queries)
	}
	c.AlterProfileSchema(schema, nil, rePaths)
	identifiers = []string{"android.identifier"}

	ws = c.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 10 {
		t.Fatalf("expected 10 properties in the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}
	if p, ok := ws.ProfileSchema.Properties().ByName("email"); ok {
		t.Fatalf("expected no \"email\" property, got property %#v", p)
	}
	if p, err := ws.ProfileSchema.Properties().ByPath("android.id"); err == nil {
		t.Fatalf("expected no \"android.id\" property, got property %#v", p)
	}
	if _, err := ws.ProfileSchema.Properties().ByPath("android.identifier"); err != nil {
		t.Fatalf("expected property \"android.identifier\", got no property: %s", err)
	}
	if !types.Equal(schema, ws.ProfileSchema) {
		t.Fatalf("expected equal schemas, got different schemas")
	}
	if !slices.Equal(identifiers, ws.Identifiers) {
		t.Fatalf("expected identifiers %v, got %v", identifiers, ws.Identifiers)
	}

	// Drop "android.identifier".
	properties = []types.Property{}
	for _, p := range schema.Properties().All() {
		switch p.Name {
		case "android":
			var props []types.Property
			for _, p := range p.Type.Properties().All() {
				if p.Name == "identifier" {
					continue
				}
				props = append(props, p)
			}
			p.Type = types.Object(props)
		}
		properties = append(properties, p)
	}
	schema = types.Object(properties)
	queries = c.PreviewAlterProfileSchema(schema, nil)
	expectedQueries = []string{
		"BEGIN;",
		"DROP VIEW \"profiles\";",
		"ALTER TABLE \"meergo_profiles_0\"\n\tDROP COLUMN \"android_identifier\";",
		"ALTER TABLE \"meergo_identities\"\n\tDROP COLUMN \"android_identifier\";",
		"CREATE VIEW \"profiles\" AS SELECT\n\t\"__mpid__\",\n\t\"__last_change_time__\",\n\t\"dummy_id\",\n\t\"android_idfa\",\n\t\"android_push_token\",\n\t\"ios_id\",\n\t\"ios_idfa\",\n\t\"ios_push_token\",\n\t\"first_name\",\n\t\"last_name\",\n\t\"gender\",\n\t\"food_preferences_drink\",\n\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n\t\"favorite_movie_title\",\n\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"meergo_profiles_0\";",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %#v, got %#v", expectedQueries, queries)
	}
	c.AlterProfileSchema(schema, nil, rePaths)

	ws = c.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 10 {
		t.Fatalf("expected 10 properties in the \"profiles\" schema, got %d", n)
	}
	p, _ := ws.ProfileSchema.Properties().ByName("android")
	if n := p.Type.Properties().Len(); n != 2 {
		t.Fatalf("expected 2 properties in the \"android\" object of the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}
	if p, err := ws.ProfileSchema.Properties().ByPath("android.identifier"); err == nil {
		t.Fatalf("expected no \"android.identifier\" property, got property %#v", p)
	}
	if !types.Equal(schema, ws.ProfileSchema) {
		t.Fatalf("expected equal schemas, got different schemas")
	}
	if ws.Identifiers == nil || len(ws.Identifiers) != 0 {
		t.Fatalf("expected no identifiers, got %v", ws.Identifiers)
	}

	// Create a schema with two properties that would conflict each other.
	schema = types.Object(append(file.Schema.Properties().Slice(),
		types.Property{Name: "a_b", Type: types.Text(), ReadOptional: true},
		types.Property{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: types.Text(), ReadOptional: true},
		}), ReadOptional: true},
	))
	_, err = c.PreviewAlterProfileSchemaErr(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedErr := `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"two profile pipeline schema properties would have the same column name \"a_b\" in the data warehouse, case-insensitively"}}`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
	err = c.AlterProfileSchemaErr(schema, nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Create a schema with a null property.
	schema = types.Object(append(file.Schema.Properties().Slice(),
		types.Property{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: types.Text(), ReadOptional: true, Nullable: true},
		}), ReadOptional: true},
	))
	_, err = c.PreviewAlterProfileSchemaErr(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedErr = `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"profile schema properties cannot be nullable"}}`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
	err = c.AlterProfileSchemaErr(schema, nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Create a primary source for the first property.
	firstProperty := file.Schema.Properties().Names()[0]
	primarySource := c.CreateDummy("Primary Source", meergotester.Source)
	primarySources := map[string]int{firstProperty: primarySource}
	c.AlterProfileSchema(file.Schema, primarySources, nil)
	ws = c.Workspace()
	if !maps.Equal(primarySources, ws.PrimarySources) {
		t.Fatalf("expected primary sources %#v, got %#v", primarySources, ws.PrimarySources)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}

	// Set a primary source for a not existent property.
	primarySources = map[string]int{"not_existent_property": primarySource}
	err = c.AlterProfileSchemaErr(file.Schema, primarySources, nil)
	expectedErr = `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"primary sources are not valid: property path \"not_existent_property\" does not exist","cause":"property path \"not_existent_property\" does not exist"}}`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Set a not existing primary source for the first property.
	notExistentSource := primarySource - 1
	if notExistentSource == 0 {
		notExistentSource = 2
	}
	primarySources = map[string]int{firstProperty: notExistentSource}
	err = c.AlterProfileSchemaErr(file.Schema, primarySources, nil)
	expectedErr = fmt.Sprintf(`unexpected HTTP status code 422: {"error":{"code":"ConnectionNotExist","message":"primary source %d does not exist"}}`, notExistentSource)
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

}

// checkSchemaProperties is used internally by the tests and checks that the
// profiles schema does not contain 'nullable' or 'required' properties.
func checkSchemaProperties(schema types.Type) error {
	for path, p := range schema.Properties().WalkAll() {
		if p.Nullable {
			return fmt.Errorf("unexpected nullable property %q", path)
		}
		if !p.ReadOptional {
			return fmt.Errorf("unexpected non-optional property %q", path)
		}
	}
	return nil
}
