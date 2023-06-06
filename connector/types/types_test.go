//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types_test

import (
	"math"
	"testing"

	"chichi/connector/types"
)

func TestLen(t *testing.T) {

	type Expected struct {
		OK  bool
		Len int
	}

	tests := []struct {
		Type     types.Type
		Expected Expected
	}{
		{types.Text(), Expected{false, 0}},
		{types.Text().WithByteLen(1).WithCharLen(1), Expected{true, 1}},
		{types.Text().WithByteLen(math.MaxInt32).WithCharLen(math.MaxInt32), Expected{true, math.MaxInt32}},
		{types.Text().WithByteLen(types.MaxTextLen).WithCharLen(types.MaxTextLen), Expected{true, types.MaxTextLen}},
	}

	for _, test := range tests {
		got, ok := test.Type.ByteLen()
		if ok == test.Expected.OK {
			if got != test.Expected.Len {
				t.Errorf("ByteLen(%d): expected %d, got %d", test.Expected.Len, test.Expected.Len, got)
			}
		} else {
			t.Errorf("ByteLen(%d): expected %t, got %t", test.Expected.Len, test.Expected.OK, ok)
		}
		got, ok = test.Type.CharLen()
		if ok == test.Expected.OK {
			if got != test.Expected.Len {
				t.Errorf("CharLen(%d): expected %d, got %d", test.Expected.Len, test.Expected.Len, got)
			}
		} else {
			t.Errorf("CharLen(%d): expected %t, got %t", test.Expected.Len, test.Expected.OK, ok)
		}
	}

}

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

func TestHasFlatProperties(t *testing.T) {

	tests := []struct {
		Type     types.Type
		Expected bool
	}{
		{types.Boolean(), false},
		{types.Object([]types.Property{{Name: "email", Type: types.Text()}}), false},
		{types.Object([]types.Property{
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street1", Type: types.Text()},
				{Name: "street2", Type: types.Text()},
			})},
		}), false},
		{types.Object([]types.Property{
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street1", Type: types.Text()},
				{Name: "street2", Type: types.Text()},
			}), Flat: true},
		}), true},
		{types.Array(types.Float()), false},
		{types.Array(types.Object([]types.Property{{Name: "email", Type: types.Text()}})), false},
		{types.Array(types.Object([]types.Property{
			{Name: "name", Type: types.Object([]types.Property{
				{Name: "first", Type: types.Text()},
			})},
		})), false},
		{types.Array(types.Object([]types.Property{
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street1", Type: types.Text()},
				{Name: "street2", Type: types.Text()},
			}), Flat: true},
		})), true},
		{types.Map(types.Int()), false},
		{types.Map(types.Object([]types.Property{{Name: "email", Type: types.Text()}})), false},
		{types.Map(types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street1", Type: types.Text()},
				{Name: "street2", Type: types.Text()},
			}), Flat: true},
		})), true},
		{types.Map(types.Array(types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street", Type: types.Object([]types.Property{
					{Name: "line1", Type: types.Text()},
					{Name: "line2", Type: types.Text()},
				}), Flat: true},
				{Name: "City", Type: types.Text()},
			}), Flat: true},
		}))), true},
		{types.Map(types.Array(types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street", Type: types.Object([]types.Property{
					{Name: "line1", Type: types.Text()},
					{Name: "line2", Type: types.Text()},
				}), Flat: true},
				{Name: "City", Type: types.Text()},
			}), Flat: true},
		}))).Unflatten(), false},
	}

	for i, test := range tests {
		if got := test.Type.HasFlatProperties(); got != test.Expected {
			t.Errorf("test %d: expected %t, got %t", i, test.Expected, got)
		}
	}

}

func Test_PropertyByPath(t *testing.T) {
	cases := []struct {
		name         string
		t            types.Type
		path         types.Path
		expectedProp types.Property
		expectedOk   bool
	}{
		{
			name: "path with single component - property (of type Text) exists",
			t: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			path:         types.Path{"first_name"},
			expectedProp: types.Property{Name: "first_name", Type: types.Text()},
			expectedOk:   true,
		},
		{
			name: "path with single component - property does not exist",
			t: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			path:         types.Path{"email"},
			expectedProp: types.Property{},
			expectedOk:   false,
		},
		{
			name: "path with single component - property (of type Object) exists",
			t: types.Object([]types.Property{
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
				})},
			}),
			path: types.Path{"billing_address"},
			expectedProp: types.Property{
				Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
				})},
			expectedOk: true,
		},
		{
			name: "path with two components - property (of type Text) exists",
			t: types.Object([]types.Property{
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
				})},
			}),
			path:         types.Path{"billing_address", "street"},
			expectedProp: types.Property{Name: "street", Type: types.Text()},
			expectedOk:   true,
		},
		{
			name: "path with two components - property does not exist",
			t: types.Object([]types.Property{
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
				})},
			}),
			path:         types.Path{"billing_address", "city"},
			expectedProp: types.Property{},
			expectedOk:   false,
		},
		{
			name: "path with three components - property (Text within an Object within an Object) exists",
			t: types.Object([]types.Property{
				{Name: "movie", Type: types.Object([]types.Property{
					{Name: "director", Type: types.Object([]types.Property{
						{Name: "name", Type: types.Text()},
						{Name: "last_name", Type: types.Text()},
					})},
				})},
			}),
			path:         types.Path{"movie", "director", "last_name"},
			expectedProp: types.Property{Name: "last_name", Type: types.Text()},
			expectedOk:   true,
		},
		{
			name: "path with three components - second component of path is not an Object",
			t: types.Object([]types.Property{
				{Name: "movie", Type: types.Object([]types.Property{
					{Name: "director", Type: types.Object([]types.Property{
						{Name: "name", Type: types.Text()},
						{Name: "last_name", Type: types.Text()},
					})},
					{Name: "writer", Type: types.Text()},
				})},
			}),
			path:         types.Path{"movie", "writer", "last_name"},
			expectedProp: types.Property{},
			expectedOk:   false,
		},
	}
	for _, cas := range cases {
		t.Run(cas.name, func(t *testing.T) {
			gotProp, gotOk := cas.t.PropertyByPath(cas.path)
			if cas.expectedOk != gotOk {
				t.Fatalf("expecting ok = %t, got %t", cas.expectedOk, gotOk)
			}
			if !equalProps(cas.expectedProp, gotProp) {
				t.Fatalf("expecting property: %#v, got: %#v", cas.expectedProp, gotProp)
			}
		})
	}
}

// equalProps reports whether the properties p1 and p2 are equal.
func equalProps(p1, p2 types.Property) bool {
	if p1.Name != p2.Name {
		return false
	}
	if p1.Label != p2.Label {
		return false
	}
	if p1.Description != p2.Description {
		return false
	}
	if p1.Role != p2.Role {
		return false
	}
	if !p1.Type.EqualTo(p2.Type) {
		return false
	}
	if p1.Required != p2.Required {
		return false
	}
	if p1.Nullable != p2.Nullable {
		return false
	}
	if p1.Flat != p2.Flat {
		return false
	}
	return true
}
