// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestAdminInitialProfileSchema verifies that the initial profile schema and
// property roles used by the Admin are applied to a workspace and its roles
// persist.
func TestAdminInitialProfileSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	const fixturesDir = "../admin/src/components/routes/WorkspaceCreate"
	schemaPath := filepath.Join(fixturesDir, "InitialSchema.json")
	encodedSchema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("expected readable schema file %q, got %s", schemaPath, err)
	}
	var schema types.Type
	err = json.Unmarshal(encodedSchema, &schema)
	if err != nil {
		t.Fatalf("expected valid schema JSON in %q, got %s", schemaPath, err)
	}

	rolesPath := filepath.Join(fixturesDir, "InitialPropertyRoles.json")
	encodedRoles, err := os.ReadFile(rolesPath)
	if err != nil {
		t.Fatalf("expected readable property roles file %q, got %s", rolesPath, err)
	}
	var propertyRoles krenalistester.PropertyRoles
	err = json.Unmarshal(encodedRoles, &propertyRoles)
	if err != nil {
		t.Fatalf("expected valid property roles JSON in %q, got %s", rolesPath, err)
	}
	expectedRoles := krenalistester.PropertyRoles{
		FirstName: "first_name",
		LastName:  "last_name",
		Email:     "email",
		Country:   "address.country",
	}
	if propertyRoles != expectedRoles {
		t.Fatalf("expected initial property roles %#v, got %#v", expectedRoles, propertyRoles)
	}

	k := krenalistester.NewKrenalisInstance(t)
	k.SetProfileSchema(schema)
	k.SetPropertyRoles(propertyRoles)
	k.Start()
	defer k.Stop()

	workspace := k.Workspace()
	t.Run("property roles", func(t *testing.T) {
		if workspace.PropertyRoles != expectedRoles {
			t.Errorf("expected property roles %#v, got %#v", expectedRoles, workspace.PropertyRoles)
		}
	})

	var persistedRoles []string
	k.QueryRowTestDatabase(t.Context(), &persistedRoles, `SELECT ARRAY[
		profile_role_first_name, profile_role_last_name, profile_role_email, profile_role_country, profile_role_photo]
		FROM workspaces WHERE id = $1`, k.WorkspaceID())
	expectedPersistedRoles := []string{
		expectedRoles.FirstName, expectedRoles.LastName, expectedRoles.Email,
		expectedRoles.Country, expectedRoles.Photo,
	}
	t.Run("persisted roles", func(t *testing.T) {
		if !slices.Equal(persistedRoles, expectedPersistedRoles) {
			t.Errorf("expected persisted roles %v, got %v", expectedPersistedRoles, persistedRoles)
		}
	})

	got := workspace.ProfileSchema
	if !got.IsObject() {
		t.Fatalf("expected object profile schema, got %#v", got)
	}
	t.Run("profile schema", func(t *testing.T) {
		if !types.Equal(schema, got) {
			t.Errorf("expected profile schema %#v, got %#v", schema.Properties().Slice(), got.Properties().Slice())
		}
	})

	for _, test := range []struct {
		path     string
		expected types.Type
	}{
		{"email", types.String().WithMaxLength(254).AsEmail()},
		{"phone_number", types.String().AsPhone()},
		{"address.country", types.String().AsCountry(types.ISO3166Alpha2)},
	} {
		t.Run(test.path, func(t *testing.T) {
			property, err := got.Properties().ByPath(test.path)
			if err != nil {
				t.Fatalf("expected property %q, got %s", test.path, err)
			}
			if !types.Equal(test.expected, property.Type) {
				t.Errorf("expected type %#v for property %q, got %#v", test.expected, test.path, property.Type)
			}
		})
	}

}
