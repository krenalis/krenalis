// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestProfileRoleAssignments verifies that profile role assignments are
// validated and persisted, and that rejected updates preserve the workspace's
// in-memory schema and assignments.
func TestProfileRoleAssignments(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	workspace := k.Workspace()
	if workspace.AssignedRoles != (krenalistester.ProfileRoleAssignments{}) {
		t.Fatalf("expected no assigned roles when omitted at creation, got %#v", workspace.AssignedRoles)
	}
	var initialWorkspaceCount int
	k.QueryRowTestDatabase(t.Context(), &initialWorkspaceCount, "SELECT COUNT(*) FROM workspaces")

	// Invalid roles must be rejected before checking the initialized warehouse.
	t.Run("workspace creation", func(t *testing.T) {

		headers := http.Header{"Krenalis-Workspace": nil}

		for _, endpoint := range []string{"/v1/workspaces", "/v1/workspaces/test"} {
			for _, test := range []struct {
				name        string
				assignments krenalistester.ProfileRoleAssignments
				message     string
			}{
				{"missing property", krenalistester.ProfileRoleAssignments{FirstName: "missing"}, "does not exist"},
				{"incompatible property", krenalistester.ProfileRoleAssignments{Email: "first_name"}, "not compatible"},
				{"duplicate property", krenalistester.ProfileRoleAssignments{FirstName: "first_name", LastName: "first_name"}, "assigned to both profile roles"},
			} {
				t.Run(endpoint+"/"+test.name, func(t *testing.T) {
					body := map[string]any{
						"name":          "Invalid roles",
						"profileSchema": workspace.ProfileSchema,
						"assignedRoles": test.assignments,
						"warehouse": map[string]any{
							"platform": "PostgreSQL",
							"settings": krenalistester.PostgresWarehouseSettings(),
						},
					}
					err := k.TryCall(http.MethodPost, endpoint, headers, body, nil)
					if err != nil {
						statusError, ok := errors.AsType[*krenalistester.StatusCodeError](err)
						if !ok {
							t.Fatalf("expected invalid role assignment response, got %v", err)
						}
						if statusError.Response.Code != http.StatusBadRequest {
							t.Fatalf("expected status code %d, got %v", http.StatusBadRequest, err)
						}
						if !strings.Contains(statusError.Response.Text, test.message) {
							t.Fatalf("expected response containing %q, got %s", test.message, statusError.Response.Text)
						}
						return
					}
					t.Fatal("expected invalid role assignment error, got nil")
				})
			}
		}

	})

	var workspaceCount int
	k.QueryRowTestDatabase(t.Context(), &workspaceCount, "SELECT COUNT(*) FROM workspaces")
	if workspaceCount != initialWorkspaceCount {
		t.Fatalf("expected %d workspaces after rejected creations, got %d", initialWorkspaceCount, workspaceCount)
	}

	properties := workspace.ProfileSchema.Properties().Slice()
	properties = append(properties, types.Property{
		Name:         "photo_url",
		Type:         types.String().AsURL(),
		ReadOptional: true,
	}, types.Property{
		Name:         "not_photo",
		Type:         types.Boolean(),
		ReadOptional: true,
	}, types.Property{
		Name:         "email_address",
		Type:         types.String().AsEmail(),
		ReadOptional: true,
	}, types.Property{
		Name:         "country",
		Type:         types.String().AsCountry(types.ISO3166Alpha2),
		ReadOptional: true,
	})
	schema := types.Object(properties)
	assignedRoles := krenalistester.ProfileRoleAssignments{
		FirstName: "first_name",
		LastName:  "last_name",
		Email:     "email_address",
		Country:   "country",
		Photo:     "photo_url",
	}
	k.AlterProfileSchemaWithAssignedRolesAndWait(schema, assignedRoles, nil, nil)

	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected assigned roles %#v, got %#v", assignedRoles, workspace.AssignedRoles)
	}

	// Reassign roles without changing the warehouse schema.
	assignedRoles.FirstName, assignedRoles.LastName = assignedRoles.LastName, assignedRoles.FirstName
	k.AlterProfileSchemaWithAssignedRolesAndWait(schema, assignedRoles, nil, nil)
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected reassigned roles %#v, got %#v", assignedRoles, workspace.AssignedRoles)
	}

	// Omitting assignedRoles clears the applied assignments.
	properties = schema.Properties().Slice()
	propertyIndex := slices.IndexFunc(properties, func(property types.Property) bool {
		return property.Name == "first_name"
	})
	if propertyIndex < 0 {
		t.Fatal("expected first_name property, got no matching property")
	}
	properties[propertyIndex].Description = "Updated description"
	schema = types.Object(properties)
	k.AlterProfileSchemaAndWait(schema, nil, nil)
	workspace = k.Workspace()
	if workspace.AssignedRoles != (krenalistester.ProfileRoleAssignments{}) {
		t.Fatalf("expected omitted assigned roles to clear all assignments, got %#v", workspace.AssignedRoles)
	}
	if !types.Equal(workspace.ProfileSchema, schema) {
		t.Fatalf("expected updated schema %#v, got %#v", schema.Properties().Slice(), workspace.ProfileSchema)
	}
	k.AlterProfileSchemaWithAssignedRolesAndWait(schema, assignedRoles, nil, nil)

	// Reject the schema change together with each invalid role assignment.
	changedSchema := types.Object(append(schema.Properties().Slice(), types.Property{
		Name:         "rejected_property",
		Type:         types.String(),
		ReadOptional: true,
	}))
	for _, test := range []struct {
		name        string
		assignments krenalistester.ProfileRoleAssignments
		message     string
	}{
		{"duplicate property", krenalistester.ProfileRoleAssignments{FirstName: "first_name", LastName: "first_name"}, "is assigned to both profile roles"},
		{"boolean email", krenalistester.ProfileRoleAssignments{Email: "not_photo"}, `is not compatible with profile role \"email\"`},
		{"boolean country", krenalistester.ProfileRoleAssignments{Country: "not_photo"}, `is not compatible with profile role \"country\"`},
		{"boolean photo", krenalistester.ProfileRoleAssignments{Photo: "not_photo"}, `is not compatible with profile role \"photo\"`},
		{"URL email", krenalistester.ProfileRoleAssignments{Email: "photo_url"}, `is not compatible with profile role \"email\"`},
		{"email country", krenalistester.ProfileRoleAssignments{Country: "email_address"}, `is not compatible with profile role \"country\"`},
		{"email photo", krenalistester.ProfileRoleAssignments{Photo: "email_address"}, `is not compatible with profile role \"photo\"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := k.TryAlterProfileSchemaWithAssignedRoles(changedSchema, test.assignments, nil, nil)
			if err != nil {
				statusError, ok := errors.AsType[*krenalistester.StatusCodeError](err)
				if !ok || statusError.Response.Code != http.StatusBadRequest {
					t.Fatalf("expected status code %d, got %v", http.StatusBadRequest, err)
				}
				if !strings.Contains(statusError.Response.Text, test.message) {
					t.Fatalf("expected error containing %q, got %v", test.message, err)
				}
			}
			if err == nil {
				t.Fatal("expected invalid role assignment error, got nil")
			}
			var after krenalistester.Workspace
			err = k.TryCall(http.MethodGet, "/v1/workspaces/current", nil, nil, &after)
			if err != nil {
				t.Fatalf("expected workspace after rejected update, got %v", err)
			}
			if after.AssignedRoles != assignedRoles {
				t.Errorf("expected unchanged assigned roles %#v, got %#v", assignedRoles, after.AssignedRoles)
			}
			if !types.Equal(after.ProfileSchema, schema) {
				t.Errorf("expected unchanged schema %#v, got %#v", schema.Properties().Slice(), after.ProfileSchema)
			}
		})
	}

	properties = slices.DeleteFunc(schema.Properties().Slice(), func(property types.Property) bool {
		return property.Name == "photo_url"
	})
	schemaWithoutPhoto := types.Object(properties)
	err := k.TryAlterProfileSchemaWithAssignedRoles(schemaWithoutPhoto, assignedRoles, nil, nil)
	if err != nil {
		statusError, ok := errors.AsType[*krenalistester.StatusCodeError](err)
		if !ok || statusError.Response.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %v", http.StatusBadRequest, err)
		}
		if !strings.Contains(statusError.Response.Text, `refers to property \"photo_url\", which does not exist`) {
			t.Fatalf("expected dangling Photo assignment error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected dangling Photo assignment error, got nil")
	}
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected rejected assignments to leave %#v unchanged, got %#v", assignedRoles, workspace.AssignedRoles)
	}
	if !types.Equal(workspace.ProfileSchema, schema) {
		t.Fatalf("expected unchanged schema %#v, got %#v", schema.Properties().Slice(), workspace.ProfileSchema)
	}

	assignedRoles.Photo = ""
	k.AlterProfileSchemaWithAssignedRolesAndWait(schemaWithoutPhoto, assignedRoles, nil, nil)
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected assigned roles %#v after deleting photo, got %#v", assignedRoles, workspace.AssignedRoles)
	}
	if !types.Equal(workspace.ProfileSchema, schemaWithoutPhoto) {
		t.Fatalf("expected schema without photo %#v, got %#v",
			schemaWithoutPhoto.Properties().Slice(), workspace.ProfileSchema)
	}
	var persistedRoles []string
	k.QueryRowTestDatabase(t.Context(), &persistedRoles, `SELECT ARRAY[
		profile_role_first_name, profile_role_last_name, profile_role_email, profile_role_country, profile_role_photo]
		FROM workspaces WHERE id = $1`, k.WorkspaceID())
	expectedPersistedRoles := []string{
		assignedRoles.FirstName, assignedRoles.LastName, assignedRoles.Email, assignedRoles.Country, assignedRoles.Photo,
	}
	if !slices.Equal(persistedRoles, expectedPersistedRoles) {
		t.Fatalf("expected persisted roles %v, got %v", expectedPersistedRoles, persistedRoles)
	}

}
