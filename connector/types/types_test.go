//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types_test

import (
	"testing"

	"chichi/connector/types"
)

func TestAsRole(t *testing.T) {
	cases := []struct {
		object   types.Type
		role     types.Role
		expected types.Type
	}{
		{
			object: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
				},
			}),
			role: types.BothRole,
			expected: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
				},
			}),
		},
		{
			object: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
				},
			}),
			role: types.SourceRole,
			expected: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
				},
			}),
		},
		{
			object: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
				},
			}),
			role: types.DestinationRole,
			expected: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
				},
			}),
		},
		{
			object: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
					Role: types.SourceRole,
				},
			}),
			role:     types.DestinationRole,
			expected: types.Type{},
		},
		{
			object: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
					Role: types.DestinationRole,
				},
			}),
			role:     types.SourceRole,
			expected: types.Type{},
		},
		{
			object: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
					Role: types.SourceRole,
				},
				{
					Name: "LastName",
					Type: types.Text(),
					Role: types.SourceRole,
				},
			}),
			role:     types.DestinationRole,
			expected: types.Type{},
		},
		{
			object: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
					Role: types.BothRole,
				},
				{
					Name: "LastName",
					Type: types.Text(),
					Role: types.SourceRole,
				},
			}),
			role: types.DestinationRole,
			expected: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
					Role: types.BothRole,
				},
			}),
		},
		{
			object: types.Object([]types.Property{
				{
					Name: "ID",
					Type: types.Text(),
					Role: types.SourceRole,
				},
				{
					Name: "FirstName",
					Type: types.Text(),
				},
				{
					Name: "LastName",
					Type: types.Text(),
				},
			}),
			role: types.DestinationRole,
			expected: types.Object([]types.Property{
				{
					Name: "FirstName",
					Type: types.Text(),
				},
				{
					Name: "LastName",
					Type: types.Text(),
				},
			}),
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			got := cas.object.AsRole(cas.role)
			gotValid := got.Valid()
			expectedValid := cas.expected.Valid()
			if !expectedValid && !gotValid {
				// Ok.
				return
			}
			if expectedValid && !gotValid {
				t.Fatal("unexpected invalid schema")
			}
			if !expectedValid && gotValid {
				t.Fatalf("expecting an invalid schema, got %v", got)
			}
			if !cas.expected.EqualTo(got) {
				t.Fatalf("expected schema %v != got %v", cas.expected, got)
			}
		})
	}

}
