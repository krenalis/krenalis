// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// Test_validateProfileRoleAssignments verifies Profile role assignment
// validation.
func Test_validateProfileRoleAssignments(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "given_name", Type: types.String(), ReadOptional: true},
		{Name: "family_name", Type: types.String(), ReadOptional: true},
		{Name: "email", Type: types.String(), ReadOptional: true, Semantic: types.Email()},
		{Name: "country", Type: types.String(), ReadOptional: true, Semantic: types.Country(types.ISO3166Alpha2)},
		{Name: "emails", Type: types.Array(types.String()), ReadOptional: true, Semantic: types.Email()},
		{Name: "countries", Type: types.Array(types.String()), ReadOptional: true,
			Semantic: types.Country(types.ISO3166Alpha2)},
		{Name: "photo_url", Type: types.String(), ReadOptional: true, Semantic: types.URL()},
		{Name: "photo_urls", Type: types.Array(types.String()), ReadOptional: true, Semantic: types.URL()},
		{Name: "subscribed", Type: types.Boolean(), ReadOptional: true},
		{Name: "details", Type: types.Object([]types.Property{
			{Name: "name", Type: types.String(), ReadOptional: true},
		}), ReadOptional: true},
	})

	tests := []struct {
		name        string
		assignments ProfileRoleAssignments
		err         string
	}{
		{name: "No assignments"},
		{
			name: "All roles",
			assignments: ProfileRoleAssignments{
				FirstName: "given_name",
				LastName:  "family_name",
				Email:     "email",
				Country:   "country",
				Photo:     "photo_url",
			},
		},
		{name: "Nested property", assignments: ProfileRoleAssignments{FirstName: "details.name"}},
		{
			name:        "Invalid path",
			assignments: ProfileRoleAssignments{FirstName: "invalid-path"},
			err:         `profile role "first_name" has invalid property path "invalid-path"`,
		},
		{
			name:        "Missing property",
			assignments: ProfileRoleAssignments{FirstName: "missing"},
			err:         `profile role "first_name" refers to property "missing", which does not exist`,
		},
		{
			name:        "Two roles on one property",
			assignments: ProfileRoleAssignments{FirstName: "given_name", LastName: "given_name"},
			err:         `property "given_name" is assigned to both profile roles "first_name" and "last_name"`,
		},
		{
			name:        "Name with semantic",
			assignments: ProfileRoleAssignments{FirstName: "email"},
			err:         `property "email" is not compatible with profile role "first_name"`,
		},
		{
			name:        "String email",
			assignments: ProfileRoleAssignments{Email: "given_name"},
			err:         `property "given_name" is not compatible with profile role "email"`,
		},
		{
			name:        "Array email",
			assignments: ProfileRoleAssignments{Email: "emails"},
			err:         `property "emails" is not compatible with profile role "email"`,
		},
		{
			name:        "Email country",
			assignments: ProfileRoleAssignments{Country: "email"},
			err:         `property "email" is not compatible with profile role "country"`,
		},
		{
			name:        "Array country",
			assignments: ProfileRoleAssignments{Country: "countries"},
			err:         `property "countries" is not compatible with profile role "country"`,
		},
		{
			name:        "Boolean photo",
			assignments: ProfileRoleAssignments{Photo: "subscribed"},
			err:         `property "subscribed" is not compatible with profile role "photo"`,
		},
		{
			name:        "Array photo",
			assignments: ProfileRoleAssignments{Photo: "photo_urls"},
			err:         `property "photo_urls" is not compatible with profile role "photo"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			err := validateProfileRoleAssignments(schema, test.assignments)
			var got string
			if err != nil {
				got = err.Error()
			}
			if got != test.err {
				t.Fatalf("expected error %q, got %q", test.err, got)
			}

		})
	}

}
