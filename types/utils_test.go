//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types

import (
	"errors"
	"testing"
)

func Test_AsRole(t *testing.T) {
	cases := []struct {
		object   Type
		role     Role
		expected Type
	}{
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
			role: BothRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
			role: SourceRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
			role: DestinationRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: SourceRole,
				},
			}),
			role:     DestinationRole,
			expected: Type{},
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: DestinationRole,
				},
			}),
			role:     SourceRole,
			expected: Type{},
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: SourceRole,
				},
				{
					Name: "LastName",
					Type: Text(),
					Role: SourceRole,
				},
			}),
			role:     DestinationRole,
			expected: Type{},
		},
		{
			object: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: BothRole,
				},
				{
					Name: "LastName",
					Type: Text(),
					Role: SourceRole,
				},
			}),
			role: DestinationRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
					Role: BothRole,
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "ID",
					Type: Text(),
					Role: SourceRole,
				},
				{
					Name: "FirstName",
					Type: Text(),
				},
				{
					Name: "LastName",
					Type: Text(),
				},
			}),
			role: DestinationRole,
			expected: Object([]Property{
				{
					Name: "FirstName",
					Type: Text(),
				},
				{
					Name: "LastName",
					Type: Text(),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name:         "ID",
					Type:         Text(),
					Role:         SourceRole,
					ReadOptional: true,
				},
				{
					Name:           "FirstName",
					Type:           Text(),
					Role:           DestinationRole,
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					Role:           BothRole,
					ReadOptional:   true,
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
			role: BothRole,
			expected: Object([]Property{
				{
					Name:         "ID",
					Type:         Text(),
					Role:         SourceRole,
					ReadOptional: true,
				},
				{
					Name:           "FirstName",
					Type:           Text(),
					Role:           DestinationRole,
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					Role:           BothRole,
					ReadOptional:   true,
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name:         "ID",
					Type:         Text(),
					Role:         SourceRole,
					ReadOptional: true,
				},
				{
					Name:           "FirstName",
					Type:           Text(),
					Role:           DestinationRole,
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					Role:           BothRole,
					ReadOptional:   true,
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
			role: SourceRole,
			expected: Object([]Property{
				{
					Name:         "ID",
					Type:         Text(),
					Role:         SourceRole,
					ReadOptional: true,
				},
				{
					Name:         "LastName",
					Type:         Text(),
					Role:         BothRole,
					ReadOptional: true,
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name:         "ID",
					Type:         Text(),
					Role:         SourceRole,
					ReadOptional: true,
				},
				{
					Name:           "FirstName",
					Type:           Text(),
					Role:           DestinationRole,
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					Role:           BothRole,
					ReadOptional:   true,
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
			role: DestinationRole,
			expected: Object([]Property{
				{
					Name:           "FirstName",
					Type:           Text(),
					Role:           DestinationRole,
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					Role:           BothRole,
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			got := AsRole(cas.object, cas.role)
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
				t.Fatalf("expecting an invalid schema, got %#v", got)
			}
			if !Equal(cas.expected, got) {
				t.Fatalf("expected schema %#v != got %#v", cas.expected, got)
			}
		})
	}

}

func Test_IsValidPropertyPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"", false},
		{".", false},
		{"a", true},
		{"a.b", true},
		{"a.b.c", true},
		{"a..b", false},
		{"a.b.", false},
		{".a.b", false},
	}
	for _, test := range tests {
		if got := IsValidPropertyPath(test.path); got != test.expected {
			t.Errorf("test %q: expected %t, got %t", test.path, test.expected, got)
		}
	}
}

func Test_NumProperties(t *testing.T) {
	properties := []Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Text()},
		{Name: "c", Type: Text()},
	}
	if got := NumProperties(Object(properties)); len(properties) != got {
		t.Errorf("expected %d, got %d", len(properties), got)
	}
}

func Test_Properties_Func(t *testing.T) {
	properties := []Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Boolean()},
	}
	i := 0
	for k, p := range Properties(Object(properties)) {
		if k != i {
			t.Fatalf("expected i=%d, got i=%d", i, k)
		}
		if err := sameProperty(p, properties[i]); err != nil {
			t.Fatal(err)
		}
		i++
	}
}

func Test_PropertyByPath(t *testing.T) {
	cases := []struct {
		name     string
		t        Type
		path     string
		expected Property
		err      error
	}{
		{
			name: "path with single component - property (of type Text) exists",
			t: Object([]Property{
				{Name: "first_name", Type: Text()},
			}),
			path:     "first_name",
			expected: Property{Name: "first_name", Type: Text()},
			err:      nil,
		},
		{
			name: "path with single component - property does not exist",
			t: Object([]Property{
				{Name: "first_name", Type: Text()},
			}),
			path:     "email",
			expected: Property{},
			err:      errors.New("property path \"email\" does not exist"),
		},
		{
			name: "path with single component - property (of type Object) exists",
			t: Object([]Property{
				{Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			}),
			path: "billing_address",
			expected: Property{
				Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			err: nil,
		},
		{
			name: "path with two components - property (of type Text) exists",
			t: Object([]Property{
				{Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			}),
			path:     "billing_address.street",
			expected: Property{Name: "street", Type: Text()},
			err:      nil,
		},
		{
			name: "path with two components - property does not exist",
			t: Object([]Property{
				{Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			}),
			path:     "billing_address.city",
			expected: Property{},
			err:      errors.New("property path \"billing_address.city\" does not exist"),
		},
		{
			name: "path with three components - property (Text within an Object within an Object) exists",
			t: Object([]Property{
				{Name: "movie", Type: Object([]Property{
					{Name: "director", Type: Object([]Property{
						{Name: "name", Type: Text()},
						{Name: "last_name", Type: Text()},
					})},
				})},
			}),
			path:     "movie.director.last_name",
			expected: Property{Name: "last_name", Type: Text()},
			err:      nil,
		},
		{
			name: "path with four components - second component of path is not an Object",
			t: Object([]Property{
				{Name: "movie", Type: Object([]Property{
					{Name: "writer", Type: Text()},
				})},
			}),
			path:     "movie.writer.address.last_name",
			expected: Property{},
			err:      errors.New("property path \"movie.writer.address\" does not exist"),
		},
		{
			name: "path with three components - second component of path is not an Object",
			t: Object([]Property{
				{Name: "movie", Type: Object([]Property{
					{Name: "director", Type: Object([]Property{
						{Name: "name", Type: Text()},
						{Name: "last_name", Type: Text()},
					})},
					{Name: "writer", Type: Text()},
				})},
			}),
			path:     "movie.writer.last_name",
			expected: Property{},
			err:      errors.New("property path \"movie.writer.last_name\" does not exist"),
		},
	}
	for _, cas := range cases {
		t.Run(cas.name, func(t *testing.T) {
			got, err := PropertyByPath(cas.t, cas.path)
			if err != nil {
				if cas.err == nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if err.Error() != cas.err.Error() {
					t.Fatalf("expected error %q, got error %q", cas.err.Error(), err.Error())
				}
				return
			}
			if cas.err != nil {
				t.Fatalf("expected error %q, got no error", cas.err)
			}
			if err := sameProperty(cas.expected, got); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_PropertyExists(t *testing.T) {
	o := Object([]Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Array(Text())},
		{Name: "d", Type: Array(Object([]Property{
			{Name: "x", Type: Map(Boolean())},
			{Name: "y", Type: Map(Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "b", Type: Int(32)},
			}))},
		}))},
	})
	tests := []struct {
		path   string
		exists bool
	}{
		{"foo", false},
		{"a.foo", false},
		{"b.foo", false},
		{"c.foo", false},
		{"d.x.foo", false},
		{"d.y.a.foo", false},
		{"d.foo.y.a", false},
		{"a", true},
		{"b.x", true},
		{"d.y", true},
		{"d.y.a", true},
		{"d.y.b", true},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			got := PropertyExists(o, test.path)
			if test.exists != got {
				t.Fatalf("expected %t, got %t", test.exists, got)
			}
		})
	}
}

func Test_SubsetFunc(t *testing.T) {
	o := Object([]Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Array(Text())},
		{Name: "d", Type: Array(Object([]Property{
			{Name: "x", Type: Map(Boolean())},
			{Name: "y", Type: Map(Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "b", Type: Int(32)},
			}))},
			{Name: "z", Type: Text()},
		}))},
	})
	expected := Object([]Property{
		{Name: "a", Type: Text()},
		{Name: "c", Type: Array(Text())},
	})
	got := SubsetFunc(o, func(p Property) bool {
		return p.Name == "a" || p.Name == "c"
	})
	expected = Object([]Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Array(Text())},
	})
	got = SubsetFunc(o, func(p Property) bool {
		return p.Name != "d"
	})
	if err := sameType(expected, got); err != nil {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	got = SubsetFunc(o, func(p Property) bool {
		return false
	})
	if got.Valid() {
		t.Fatalf("expected invalid type, got %v", got)
	}
	got = SubsetFunc(o, func(p Property) bool {
		return true
	})
	if err := sameType(o, got); err != nil {
		t.Fatalf("expected %v, got %v", o, got)
	}
}

func Test_Walk(t *testing.T) {
	properties := []Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Array(Text())},
		{Name: "d", Type: Array(Object([]Property{
			{Name: "x", Type: Map(Boolean())},
			{Name: "y", Type: Map(Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "b", Type: Int(32)},
			}))},
			{Name: "z", Type: Text()},
		}))},
	}
	type entry struct {
		path     string
		property Property
	}
	iterations := []entry{
		{"a", Property{Name: "a", Type: Text()}},
		{"b", Property{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})}},
		{"b.x", Property{Name: "x", Type: Text()}},
		{"c", Property{Name: "c", Type: Array(Text())}},
		{"d", Property{Name: "d", Type: Array(Object([]Property{
			{Name: "x", Type: Map(Boolean())},
			{Name: "y", Type: Map(Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "b", Type: Int(32)},
			}))},
			{Name: "z", Type: Text()},
		}))}},
		{"d.x", Property{Name: "x", Type: Map(Boolean())}},
		{"d.y", Property{Name: "y", Type: Map(Object([]Property{
			{Name: "a", Type: Text()},
			{Name: "b", Type: Int(32)},
		}))}},
		{"d.y.a", Property{Name: "a", Type: Text()}},
		{"d.y.b", Property{Name: "b", Type: Int(32)}},
		{"d.z", Property{Name: "z", Type: Text()}},
	}
	walk := Walk(Object(properties))
	var i = 0
	walk(func(path string, p Property) bool {
		if i > len(iterations) {
			t.Fatalf("expected %d iterations, got %d", len(iterations), i)
		}
		if path != iterations[i].path {
			t.Fatalf("expected path %q, got %q", iterations[i].path, path)
		}
		if err := sameProperty(p, iterations[i].property); err != nil {
			t.Fatal(err)
		}
		i++
		return true
	})
}
