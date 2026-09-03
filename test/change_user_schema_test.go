// Copyright 2026 Open2b. All rights reserved.
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

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func TestChangeProfileSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	ws := k.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 10 {
		t.Fatalf("expected 10 properties in the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}

	identifiers := []string{"email", "android.id"}
	k.UpdateIdentityResolutionSettings(true, identifiers)

	// Read the schema in "testdata/change_profile_schema_test.json".
	f, err := os.Open("testdata/change_profile_schema_test.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var file struct {
		Schema         types.Type
		PrimarySources map[string]string
		RePaths        map[string]any
	}
	err = dec.Decode(&file)
	if err != nil {
		t.Fatal(err)
	}

	// Alter the profile schema.
	queries := k.PreviewAlterProfileSchema(file.Schema, file.RePaths)
	if queries == nil {
		t.Fatal("expected an empty query list, got nil")
	}
	if len(queries) != 0 {
		t.Fatalf("expected no queries, got %#v", queries)
	}
	k.AlterProfileSchemaAndWait(file.Schema, file.PrimarySources, file.RePaths)

	ws = k.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 10 {
		t.Fatalf("expected 10 properties in the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}
	if !slices.Equal(identifiers, ws.Identifiers) {
		t.Fatalf("expected identifiers %v, got %v", identifiers, ws.Identifiers)
	}

	// Change only schema metadata.
	descriptionProperties := file.Schema.Properties().Slice()
	descriptionProperties[0].Description = "Updated description"
	descriptionSchema := types.Object(descriptionProperties)
	queries = k.PreviewAlterProfileSchema(descriptionSchema, nil)
	if len(queries) != 0 {
		t.Fatalf("expected no queries, got %#v", queries)
	}
	k.AlterProfileSchemaAndWait(descriptionSchema, file.PrimarySources, nil)

	ws = k.Workspace()
	if !types.Equal(descriptionSchema, ws.ProfileSchema) {
		t.Fatal("expected the metadata-only schema change to be persisted")
	}
	if !slices.Equal(identifiers, ws.Identifiers) {
		t.Fatalf("expected identifiers %v, got %v", identifiers, ws.Identifiers)
	}

	// Reject adding a semantic to a materialized property.
	semanticProperties := descriptionSchema.Properties().Slice()
	i := slices.IndexFunc(semanticProperties, func(property types.Property) bool {
		return property.Name == "phone_numbers"
	})
	if i == -1 {
		t.Fatal("phone_numbers property not found")
	}
	semanticProperties[i].Semantic = types.Phone()
	semanticSchema := types.Object(semanticProperties)
	_, err = k.TryPreviewAlterProfileSchema(semanticSchema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedSemanticPreviewErr := `PUT v1/profiles/schema/preview: unexpected status code 422: ` +
		`{"error":{"code":"InvalidAlterSchema","message":"cannot alter the schema as specified: ` +
		`semantic cannot be added to materialized profile schema property \"phone_numbers\""}} ` +
		`[request has body: true, response body expected: true]`
	if err.Error() != expectedSemanticPreviewErr {
		t.Fatalf("expected error %q, got %q", expectedSemanticPreviewErr, err.Error())
	}
	err = k.TryAlterProfileSchema(semanticSchema, nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedSemanticErr := `PUT v1/profiles/schema: unexpected status code 422: ` +
		`{"error":{"code":"InvalidAlterSchema","message":"cannot alter the schema as specified: ` +
		`semantic cannot be added to materialized profile schema property \"phone_numbers\""}} ` +
		`[request has body: true, response body expected: false]`
	if err.Error() != expectedSemanticErr {
		t.Fatalf("expected error %q, got %q", expectedSemanticErr, err.Error())
	}

	// Reject formatted datetime text in the profile schema.
	invalidSemanticProperties := semanticSchema.Properties().Slice()
	i = slices.IndexFunc(invalidSemanticProperties, func(property types.Property) bool {
		return property.Name == "email"
	})
	if i == -1 {
		t.Fatal("email property not found")
	}
	invalidSemanticProperties[i].Semantic = types.FormattedDateTime("2006-01-02 15:04:05")
	invalidSemanticSchema := types.Object(invalidSemanticProperties)
	_, err = k.TryPreviewAlterProfileSchema(invalidSemanticSchema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedSemanticPreviewErr = `PUT v1/profiles/schema/preview: unexpected status code 400: ` +
		`{"error":{"code":"BadRequest","message":"profile schema properties cannot have datetime semantic"}} ` +
		`[request has body: true, response body expected: true]`
	if err.Error() != expectedSemanticPreviewErr {
		t.Fatalf("expected error %q, got %q", expectedSemanticPreviewErr, err.Error())
	}
	err = k.TryAlterProfileSchema(invalidSemanticSchema, nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedSemanticErr = `PUT v1/profiles/schema: unexpected status code 400: ` +
		`{"error":{"code":"BadRequest","message":"profile schema properties cannot have datetime semantic"}} ` +
		`[request has body: true, response body expected: false]`
	if err.Error() != expectedSemanticErr {
		t.Fatalf("expected error %q, got %q", expectedSemanticErr, err.Error())
	}

	// Add a single property.
	schema := types.Object(append(descriptionSchema.Properties().Slice(), types.Property{
		Name: "new_prop", Type: types.String(), ReadOptional: true, Semantic: types.Email(),
	}))
	queries = k.PreviewAlterProfileSchema(schema, nil)
	expectedQueries := []string{"BEGIN;",
		"DROP VIEW \"profiles\";",
		"ALTER TABLE \"krenalis_profiles_0\"\n\tADD COLUMN \"new_prop\" character varying;",
		"ALTER TABLE \"krenalis_identities\"\n\tADD COLUMN \"new_prop\" character varying;",
		"CREATE VIEW \"profiles\" AS SELECT\n\t\"_kpid\",\n\t\"_updated_at\",\n\t\"email\",\n\t\"dummy_id\",\n\t\"android_id\",\n\t\"android_idfa\",\n\t\"android_push_token\",\n\t\"ios_id\",\n\t\"ios_idfa\",\n\t\"ios_push_token\",\n\t\"first_name\",\n\t\"last_name\",\n\t\"gender\",\n\t\"food_preferences_drink\",\n\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n\t\"favorite_movie_title\",\n\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"krenalis_profiles_0\";",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %#v, got %#v", expectedQueries, queries)
	}
	k.AlterProfileSchemaAndWait(schema, nil, nil)

	ws = k.Workspace()
	if n := ws.ProfileSchema.Properties().Len(); n != 11 {
		t.Fatalf("expected 11 properties in the \"profiles\" schema, got %d", n)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}
	if !slices.Equal(identifiers, ws.Identifiers) {
		t.Fatalf("expected identifiers %v, got %v", identifiers, ws.Identifiers)
	}

	// Remove a semantic without changing its materialized type.
	semanticProperties = schema.Properties().Slice()
	i = slices.IndexFunc(semanticProperties, func(property types.Property) bool {
		return property.Name == "new_prop"
	})
	if i == -1 {
		t.Fatal("new_prop property not found")
	}
	semanticProperties[i].Semantic = nil
	schema = types.Object(semanticProperties)
	queries = k.PreviewAlterProfileSchema(schema, nil)
	if len(queries) != 0 {
		t.Fatalf("expected no queries, got %#v", queries)
	}
	k.AlterProfileSchemaAndWait(schema, nil, nil)

	ws = k.Workspace()
	if !types.Equal(schema, ws.ProfileSchema) {
		t.Fatal("expected the semantic removal to be persisted")
	}

	// Rename the property "android.id" to "android.identifier" and drop "email".
	var properties []types.Property
	for _, p := range schema.Properties().All() {
		switch p.Name {
		case "email":
			continue
		case "android":
			props := p.Type.Properties().Slice()
			for i := range props {
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
	queries = k.PreviewAlterProfileSchema(schema, rePaths)
	expectedQueries = []string{
		"BEGIN;",
		"DROP VIEW \"profiles\";", "ALTER TABLE \"krenalis_profiles_0\"\n\tDROP COLUMN \"email\";",
		"ALTER TABLE \"krenalis_identities\"\n\tDROP COLUMN \"email\";",
		"ALTER TABLE \"krenalis_profiles_0\"\n\tRENAME COLUMN \"android_id\" TO \"android_identifier\";",
		"ALTER TABLE \"krenalis_identities\"\n\tRENAME COLUMN \"android_id\" TO \"android_identifier\";",
		"CREATE VIEW \"profiles\" AS SELECT\n\t\"_kpid\",\n\t\"_updated_at\",\n\t\"dummy_id\",\n\t\"android_identifier\",\n\t\"android_idfa\",\n\t\"android_push_token\",\n\t\"ios_id\",\n\t\"ios_idfa\",\n\t\"ios_push_token\",\n\t\"first_name\",\n\t\"last_name\",\n\t\"gender\",\n\t\"food_preferences_drink\",\n\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n\t\"favorite_movie_title\",\n\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"krenalis_profiles_0\";",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %#v, got %#v", expectedQueries, queries)
	}
	k.AlterProfileSchemaAndWait(schema, nil, rePaths)
	identifiers = []string{"android.identifier"}

	ws = k.Workspace()
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
	queries = k.PreviewAlterProfileSchema(schema, nil)
	expectedQueries = []string{
		"BEGIN;",
		"DROP VIEW \"profiles\";",
		"ALTER TABLE \"krenalis_profiles_0\"\n\tDROP COLUMN \"android_identifier\";",
		"ALTER TABLE \"krenalis_identities\"\n\tDROP COLUMN \"android_identifier\";",
		"CREATE VIEW \"profiles\" AS SELECT\n\t\"_kpid\",\n\t\"_updated_at\",\n\t\"dummy_id\",\n\t\"android_idfa\",\n\t\"android_push_token\",\n\t\"ios_id\",\n\t\"ios_idfa\",\n\t\"ios_push_token\",\n\t\"first_name\",\n\t\"last_name\",\n\t\"gender\",\n\t\"food_preferences_drink\",\n\t\"food_preferences_fruit\",\n\t\"phone_numbers\",\n\t\"favorite_movie_title\",\n\t\"favorite_movie_length\",\n\t\"favorite_movie_soundtrack_title\",\n\t\"favorite_movie_soundtrack_author\",\n\t\"favorite_movie_soundtrack_length\",\n\t\"favorite_movie_soundtrack_genre\",\n\t\"new_prop\"\nFROM \"krenalis_profiles_0\";",
		"COMMIT;",
	}
	if !slices.Equal(expectedQueries, queries) {
		t.Fatalf("expected queries %#v, got %#v", expectedQueries, queries)
	}
	k.AlterProfileSchemaAndWait(schema, nil, nil)

	ws = k.Workspace()
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
		types.Property{Name: "a_b", Type: types.String(), ReadOptional: true},
		types.Property{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: types.String(), ReadOptional: true},
		}), ReadOptional: true},
	))
	_, err = k.TryPreviewAlterProfileSchema(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedPreviewErr := `PUT v1/profiles/schema/preview: unexpected status code 400: {"error":{"code":"BadRequest","message":"two profile pipeline schema properties would have the same column name \"a_b\" in the data warehouse, case-insensitively"}} [request has body: true, response body expected: true]`
	if err.Error() != expectedPreviewErr {
		t.Fatalf("expected error %q, got %q", expectedPreviewErr, err.Error())
	}
	err = k.TryAlterProfileSchema(schema, nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedErr := `PUT v1/profiles/schema: unexpected status code 400: {"error":{"code":"BadRequest","message":"two profile pipeline schema properties would have the same column name \"a_b\" in the data warehouse, case-insensitively"}} [request has body: true, response body expected: false]`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Create a schema with a null property.
	schema = types.Object(append(file.Schema.Properties().Slice(),
		types.Property{Name: "a", Type: types.Object([]types.Property{
			{Name: "b", Type: types.String(), ReadOptional: true, Nullable: true},
		}), ReadOptional: true},
	))
	_, err = k.TryPreviewAlterProfileSchema(schema, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedPreviewErr = `PUT v1/profiles/schema/preview: unexpected status code 400: {"error":{"code":"BadRequest","message":"profile schema properties cannot be nullable"}} [request has body: true, response body expected: true]`
	if err.Error() != expectedPreviewErr {
		t.Fatalf("expected error %q, got %q", expectedPreviewErr, err.Error())
	}
	err = k.TryAlterProfileSchema(schema, nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	expectedErr = `PUT v1/profiles/schema: unexpected status code 400: {"error":{"code":"BadRequest","message":"profile schema properties cannot be nullable"}} [request has body: true, response body expected: false]`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Create a primary source for the first property.
	firstProperty := file.Schema.Properties().Names()[0]
	primarySource := k.CreateDummy("Primary Source", krenalistester.Source)
	primarySources := map[string]string{firstProperty: primarySource}
	k.AlterProfileSchemaAndWait(file.Schema, primarySources, nil)
	ws = k.Workspace()
	if !maps.Equal(primarySources, ws.PrimarySources) {
		t.Fatalf("expected primary sources %#v, got %#v", primarySources, ws.PrimarySources)
	}
	if err := checkSchemaProperties(ws.ProfileSchema); err != nil {
		t.Fatalf("invalid profile schema: %s", err)
	}

	// Set a primary source for a not existent property.
	primarySources = map[string]string{"not_existent_property": primarySource}
	err = k.TryAlterProfileSchema(file.Schema, primarySources, nil)
	expectedErr = `PUT v1/profiles/schema: unexpected status code 400: {"error":{"code":"BadRequest","message":"primary sources are not valid: property path \"not_existent_property\" does not exist","cause":"property path \"not_existent_property\" does not exist"}} [request has body: true, response body expected: false]`
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Set a not existing primary source for the first property.
	notExistentSource := "7B3mN9qK2xA4"
	primarySources = map[string]string{firstProperty: notExistentSource}
	err = k.TryAlterProfileSchema(file.Schema, primarySources, nil)
	expectedErr = fmt.Sprintf(`PUT v1/profiles/schema: unexpected status code 422: {"error":{"code":"ConnectionNotExist","message":"primary source %s does not exist"}} [request has body: true, response body expected: false]`, notExistentSource)
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
