// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// Test_validateProfilePropertyRoles verifies property role assignment
// validation.
func Test_validateProfilePropertyRoles(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "given_name", Type: types.String(), ReadOptional: true},
		{Name: "family_name", Type: types.String(), ReadOptional: true},
		{Name: "email", Type: types.String().AsEmail(), ReadOptional: true},
		{Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2), ReadOptional: true},
		{Name: "emails", Type: types.Array(types.String().AsEmail()), ReadOptional: true},
		{Name: "countries", Type: types.Array(types.String().AsCountry(types.ISO3166Alpha2)), ReadOptional: true},
		{Name: "photo_url", Type: types.String().AsURL(), ReadOptional: true},
		{Name: "photo_urls", Type: types.Array(types.String().AsURL()), ReadOptional: true},
		{Name: "subscribed", Type: types.Boolean(), ReadOptional: true},
		{Name: "details", Type: types.Object([]types.Property{
			{Name: "name", Type: types.String(), ReadOptional: true},
		}), ReadOptional: true},
	})

	tests := []struct {
		name        string
		assignments PropertyRoles
		err         string
	}{
		{name: "No assignments"},
		{
			name: "All roles",
			assignments: PropertyRoles{
				FirstName: "given_name",
				LastName:  "family_name",
				Email:     "email",
				Country:   "country",
				Photo:     "photo_url",
			},
		},
		{name: "Nested property", assignments: PropertyRoles{FirstName: "details.name"}},
		{
			name:        "Invalid path",
			assignments: PropertyRoles{FirstName: "invalid-path"},
			err:         `property role "firstName" has invalid property path "invalid-path"`,
		},
		{
			name:        "Missing property",
			assignments: PropertyRoles{FirstName: "missing"},
			err:         `property role "firstName" refers to property "missing", which does not exist`,
		},
		{
			name:        "Two roles on one property",
			assignments: PropertyRoles{FirstName: "given_name", LastName: "given_name"},
			err:         `property "given_name" is assigned to both roles "firstName" and "lastName"`,
		},
		{
			name:        "Name with semantic",
			assignments: PropertyRoles{FirstName: "email"},
			err:         `property "email" is not compatible with role "firstName"`,
		},
		{
			name:        "String email",
			assignments: PropertyRoles{Email: "given_name"},
			err:         `property "given_name" is not compatible with role "email"`,
		},
		{
			name:        "Array email",
			assignments: PropertyRoles{Email: "emails"},
			err:         `property "emails" is not compatible with role "email"`,
		},
		{
			name:        "Email country",
			assignments: PropertyRoles{Country: "email"},
			err:         `property "email" is not compatible with role "country"`,
		},
		{
			name:        "Array country",
			assignments: PropertyRoles{Country: "countries"},
			err:         `property "countries" is not compatible with role "country"`,
		},
		{
			name:        "Boolean photo",
			assignments: PropertyRoles{Photo: "subscribed"},
			err:         `property "subscribed" is not compatible with role "photo"`,
		},
		{
			name:        "Array photo",
			assignments: PropertyRoles{Photo: "photo_urls"},
			err:         `property "photo_urls" is not compatible with role "photo"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePropertyRoles(schema, test.assignments)
			if err != nil {
				if test.err == "" {
					t.Fatalf("expected no error, got %v", err)
				}
				if got := err.Error(); got != test.err {
					t.Fatalf("expected error %q, got %q", test.err, got)
				}
				return
			}
			if test.err != "" {
				t.Fatalf("expected error %q, got nil", test.err)
			}
		})
	}

}
