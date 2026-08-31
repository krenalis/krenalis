// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestProfileRoleAssignments verifies validation, persistence and atomic
// replacement of Profile role assignments.
func TestProfileRoleAssignments(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	workspace := k.Workspace()
	properties := workspace.ProfileSchema.Properties().Slice()
	properties = append(properties, types.Property{
		Name:         "photo_url",
		Type:         types.String(),
		ReadOptional: true,
		Semantic:     types.URL(),
	}, types.Property{
		Name:         "not_photo",
		Type:         types.Boolean(),
		ReadOptional: true,
	}, types.Property{
		Name:         "email_address",
		Type:         types.String(),
		ReadOptional: true,
		Semantic:     types.Email(),
	}, types.Property{
		Name:         "country",
		Type:         types.String(),
		ReadOptional: true,
		Semantic:     types.Country(types.ISO3166Alpha2),
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
	properties[0].Description = "Updated description"
	schema = types.Object(properties)
	k.AlterProfileSchemaAndWait(schema, nil, nil)
	workspace = k.Workspace()
	if workspace.AssignedRoles != (krenalistester.ProfileRoleAssignments{}) {
		t.Fatalf("expected omitted assigned roles to clear all assignments, got %#v", workspace.AssignedRoles)
	}
	k.AlterProfileSchemaWithAssignedRolesAndWait(schema, assignedRoles, nil, nil)

	invalidAssignments := assignedRoles
	invalidAssignments.LastName = invalidAssignments.FirstName
	err := k.TryAlterProfileSchemaWithAssignedRoles(schema, invalidAssignments, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "is assigned to both profile roles") {
		t.Fatalf("expected duplicate property assignment error, got %v", err)
	}
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected rejected assignments to leave %#v unchanged, got %#v", assignedRoles, workspace.AssignedRoles)
	}

	invalidAssignments = assignedRoles
	invalidAssignments.Email = "not_photo"
	err = k.TryAlterProfileSchemaWithAssignedRoles(schema, invalidAssignments, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `is not compatible with profile role \"email\"`) {
		t.Fatalf("expected incompatible Email assignment error, got %v", err)
	}
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected rejected assignments to leave %#v unchanged, got %#v", assignedRoles, workspace.AssignedRoles)
	}

	invalidAssignments = assignedRoles
	invalidAssignments.Country = "not_photo"
	err = k.TryAlterProfileSchemaWithAssignedRoles(schema, invalidAssignments, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `is not compatible with profile role \"country\"`) {
		t.Fatalf("expected incompatible Country assignment error, got %v", err)
	}
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected rejected assignments to leave %#v unchanged, got %#v", assignedRoles, workspace.AssignedRoles)
	}

	invalidAssignments = assignedRoles
	invalidAssignments.Photo = "not_photo"
	err = k.TryAlterProfileSchemaWithAssignedRoles(schema, invalidAssignments, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `is not compatible with profile role \"photo\"`) {
		t.Fatalf("expected incompatible Photo assignment error, got %v", err)
	}
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected rejected assignments to leave %#v unchanged, got %#v", assignedRoles, workspace.AssignedRoles)
	}

	properties = slices.DeleteFunc(schema.Properties().Slice(), func(property types.Property) bool {
		return property.Name == "photo_url"
	})
	schemaWithoutPhoto := types.Object(properties)
	err = k.TryAlterProfileSchemaWithAssignedRoles(schemaWithoutPhoto, assignedRoles, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `refers to property \"photo_url\", which does not exist`) {
		t.Fatalf("expected dangling Photo assignment error, got %v", err)
	}
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected rejected assignments to leave %#v unchanged, got %#v", assignedRoles, workspace.AssignedRoles)
	}

	assignedRoles.Photo = ""
	k.AlterProfileSchemaWithAssignedRolesAndWait(schemaWithoutPhoto, assignedRoles, nil, nil)
	workspace = k.Workspace()
	if workspace.AssignedRoles != assignedRoles {
		t.Fatalf("expected assigned roles %#v after deleting photo, got %#v", assignedRoles, workspace.AssignedRoles)
	}

}
