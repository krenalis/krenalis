//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types

import (
	"strings"
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
			role: Source,
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
			role: Destination,
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
					Name:         "ID",
					Type:         Text(),
					ReadOptional: true,
				},
				{
					Name:           "FirstName",
					Type:           Text(),
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					ReadOptional:   true,
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
			role: Source,
			expected: Object([]Property{
				{
					Name:         "ID",
					Type:         Text(),
					ReadOptional: true,
				},
				{
					Name: "FirstName",
					Type: Text(),
				},
				{
					Name:         "LastName",
					Type:         Text(),
					ReadOptional: true,
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name:         "ID",
					Type:         Text(),
					ReadOptional: true,
				},
				{
					Name:           "FirstName",
					Type:           Text(),
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					ReadOptional:   true,
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
			role: Destination,
			expected: Object([]Property{
				{
					Name: "ID",
					Type: Text(),
				},
				{
					Name:           "FirstName",
					Type:           Text(),
					CreateRequired: true,
					UpdateRequired: true,
				},
				{
					Name:           "LastName",
					Type:           Text(),
					CreateRequired: true,
					UpdateRequired: true,
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "Address",
					Type: Object([]Property{
						{Name: "Street", Type: Text(), ReadOptional: true},
						{Name: "City", Type: Text()},
						{Name: "Country", Type: Text(), UpdateRequired: true},
					}),
					ReadOptional: true,
				},
			}),
			role: Destination,
			expected: Object([]Property{
				{
					Name: "Address",
					Type: Object([]Property{
						{Name: "Street", Type: Text()},
						{Name: "City", Type: Text()},
						{Name: "Country", Type: Text(), UpdateRequired: true},
					}),
				},
			}),
		},
		{
			object: Object([]Property{
				{
					Name: "Address",
					Type: Object([]Property{
						{Name: "Street", Type: Text(), ReadOptional: true},
						{Name: "City", Type: Text(), CreateRequired: true, UpdateRequired: true},
						{Name: "Country", Type: Text(), ReadOptional: true, UpdateRequired: true},
					}),
				},
			}),
			role: Source,
			expected: Object([]Property{
				{
					Name: "Address",
					Type: Object([]Property{
						{Name: "Street", Type: Text(), ReadOptional: true},
						{Name: "City", Type: Text()},
						{Name: "Country", Type: Text(), ReadOptional: true},
					}),
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
				t.Fatalf("expected an invalid schema, got %#v", got)
			}
			if !Equal(cas.expected, got) {
				t.Fatalf("expected schema %#v != got %#v", cas.expected, got)
			}
		})
	}

}

func Test_DecodeUUID(t *testing.T) {
	tests := []struct {
		bytes       []byte
		expectedStr string
		expectedOk  bool
	}{
		{
			bytes:       []byte{},
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       nil,
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{100},
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{211, 124, 89, 213, 136, 127, 68, 248, 143, 250, 126, 36, 49, 79, 71, 62},
			expectedStr: "d37c59d5-887f-44f8-8ffa-7e24314f473e",
			expectedOk:  true,
		},
		{
			bytes:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expectedStr: "00000000-0000-0000-0000-000000000000",
			expectedOk:  true,
		},
		{
			bytes:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // 15 bytes instead of 16
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // 17 bytes instead of 16
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			expectedStr: "ffffffff-ffff-ffff-ffff-ffffffffffff",
			expectedOk:  true,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotStr, gotOk := DecodeUUID(test.bytes)
			if gotStr != test.expectedStr {
				t.Fatalf("expected %q, got %q", test.expectedStr, gotStr)
			}
			if gotOk != test.expectedOk {
				t.Fatalf("expected ok = %t, got ok = %t", test.expectedOk, gotOk)
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

func Test_ParseUUID(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		id, ok := ParseUUID("F47AC10B-58CC-4372-A567-0E02B2C3D479")
		if !ok || id != "f47ac10b-58cc-4372-a567-0e02b2c3d479" {
			t.Fatalf("unexpected result %q %t", id, ok)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		if id, ok := ParseUUID("invalid"); ok || id != "" {
			t.Fatalf("expected failure, got %q %t", id, ok)
		}
	})
}

func Test_PruneAtPath(t *testing.T) {

	testObject := Object([]Property{
		{Name: "a", Type: Text(), Prefilled: "pref-a", Description: "description-a", CreateRequired: true, UpdateRequired: true, ReadOptional: true, Nullable: true},
		{Name: "branch", Description: "branch description", Type: Object([]Property{
			{Name: "leaf", Prefilled: "leaf-pref", Description: "leaf description", CreateRequired: true, UpdateRequired: true, ReadOptional: true, Nullable: true, Type: Object([]Property{
				{Name: "target", Prefilled: "target-pref", Description: "target description", Nullable: true, Type: Text()},
				{Name: "other_target", Type: Text()},
			})},
			{Name: "plain", Type: Text(), Prefilled: "plain-pref"},
			{Name: "other", Type: Text()},
		})},
	})

	t.Run("top level property", func(t *testing.T) {
		got, err := PruneAtPath(testObject, "a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := Object([]Property{
			{Name: "a", Type: Text(), Prefilled: "pref-a", Description: "description-a", CreateRequired: true, UpdateRequired: true, ReadOptional: true, Nullable: true},
		})
		if !Equal(got, expected) {
			t.Fatalf("unexpected subset for top level property: %v", got)
		}
	})

	t.Run("nested path preserves hierarchy", func(t *testing.T) {
		got, err := PruneAtPath(testObject, "branch.leaf.target")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Valid() {
			t.Fatalf("expected valid subset, got invalid")
		}
		branch, ok := got.Properties().ByName("branch")
		if !ok {
			t.Fatalf("missing branch property in subset")
		}
		if branch.Description != "branch description" {
			t.Fatalf("branch description mismatch: expected %q, got %q", "branch description", branch.Description)
		}
		branchProps := branch.Type.Properties()
		if branchProps.Len() != 1 {
			t.Fatalf("expected branch type to contain 1 property, got %d", branchProps.Len())
		}
		leaf, ok := branchProps.ByName("leaf")
		if !ok {
			t.Fatalf("missing leaf property in branch subset")
		}
		if leaf.Prefilled != "leaf-pref" || leaf.Description != "leaf description" || !leaf.CreateRequired || !leaf.UpdateRequired || !leaf.ReadOptional || !leaf.Nullable {
			t.Fatalf("leaf property metadata not preserved: %+v", leaf)
		}
		leafProps := leaf.Type.Properties()
		if leafProps.Len() != 1 {
			t.Fatalf("expected leaf type to contain 1 property, got %d", leafProps.Len())
		}
		target, ok := leafProps.ByName("target")
		if !ok {
			t.Fatalf("missing target property in leaf subset")
		}
		if target.Prefilled != "target-pref" || target.Description != "target description" || !target.Nullable {
			t.Fatalf("target property metadata not preserved: %+v", target)
		}
		if target.Type.kind != TextKind {
			t.Fatalf("expected target type to be text, got %v", target.Type.kind)
		}
	})

	t.Run("missing path returns invalid type", func(t *testing.T) {
		got, err := PruneAtPath(testObject, "branch.missing")
		if err == nil {
			t.Fatalf("expected error for missing path, got nil")
		}
		if got.Valid() {
			t.Fatalf("expected invalid type for missing path, got valid")
		}
		pathErr, ok := err.(PathNotExistError)
		if !ok {
			t.Fatalf("expected PathNotExistError, got %T", err)
		}
		if pathErr.Path != "branch.missing" {
			t.Fatalf("unexpected error path: %q", pathErr.Path)
		}
	})

	t.Run("non object intermediate returns invalid type", func(t *testing.T) {
		got, err := PruneAtPath(testObject, "branch.plain.deeper")
		if err == nil {
			t.Fatalf("expected error for non-object intermediate, got nil")
		}
		if got.Valid() {
			t.Fatalf("expected invalid type when traversing through non-object intermediate, got valid")
		}
		pathErr, ok := err.(PathNotExistError)
		if !ok {
			t.Fatalf("expected PathNotExistError, got %T", err)
		}
		if pathErr.Path != "branch.plain.deeper" {
			t.Fatalf("unexpected error path: %q", pathErr.Path)
		}
	})

	t.Run("invalid path panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for invalid path")
			}
		}()
		_, _ = PruneAtPath(testObject, "")
	})

}

func Test_Prune(t *testing.T) {
	testObject := Object([]Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
			{Name: "with_description", Type: Text(), Description: "Some description"},
		})},
		{Name: "c", Type: Array(Text())},
		{Name: "d", Type: Object([]Property{
			{Name: "b", Type: Object([]Property{
				{Name: "x", Type: Text(), Description: "Description of 'x'"},
				{Name: "with_description", Type: Text()},
			}), Description: "Description of 'b'"},
		}), Description: "Description of 'd'"},
		{Name: "e", Type: Array(Map(Object([]Property{
			{Name: "a", Type: Text()},
			{Name: "b", Type: Text()},
		})))},
		{Name: "f", Type: Object([]Property{
			{Name: "f1", Type: Text(), Prefilled: "Prefilled of f1", CreateRequired: true, UpdateRequired: true, Nullable: true},
		})},
	})
	tests := []struct {
		name     string
		f        func(path string) bool
		expected Type
	}{
		{
			name:     "Just a top-level property",
			f:        func(path string) bool { return path == "a" },
			expected: Object([]Property{{Name: "a", Type: Text()}}),
		},
		{
			name: "Two top level properties, one have descendants",
			f: func(path string) bool {
				return path == "a" || strings.HasPrefix(path, "b.")
			},
			expected: Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "b", Type: Object([]Property{
					{Name: "x", Type: Text()},
					{Name: "with_description", Type: Text(), Description: "Some description"},
				})},
			}),
		},
		{
			name: "Two top level properties, one is an array(text)",
			f:    func(path string) bool { return path == "a" || path == "c" },
			expected: Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "c", Type: Array(Text())},
			}),
		},
		{
			name: "A second level property",
			f:    func(path string) bool { return path == "b.x" },
			expected: Object([]Property{
				{Name: "b", Type: Object([]Property{
					{Name: "x", Type: Text()},
				})},
			}),
		},
		{
			name:     "Not existent properties, returning invalid schema",
			f:        func(path string) bool { return path == "not_existent_property" },
			expected: Type{},
		},
		{
			name: "Second level property, for which the description must be kept",
			f:    func(path string) bool { return strings.HasPrefix(path, "b.") },
			expected: Object([]Property{
				{Name: "b", Type: Object([]Property{
					{Name: "x", Type: Text()},
					{Name: "with_description", Type: Text(), Description: "Some description"},
				})},
			}),
		},
		{
			name: "Top level property and third level property (with description)",
			f:    func(path string) bool { return path == "c" || path == "d.b.x" },
			expected: Object([]Property{
				{Name: "c", Type: Array(Text())},
				{Name: "d", Type: Object([]Property{
					{Name: "b", Type: Object([]Property{
						{Name: "x", Type: Text(), Description: "Description of 'x'"},
					}), Description: "Description of 'b'"},
				}), Description: "Description of 'd'"},
			}),
		},
		{
			name: "Top level property of type array(object)",
			f:    func(path string) bool { return path == "e" },
			expected: Object([]Property{
				{Name: "e", Type: Array(Map(Object([]Property{
					{Name: "a", Type: Text()},
					{Name: "b", Type: Text()},
				})))},
			}),
		},
		{
			name: "Referencing a top-level object and its children, which has Prefilled, CreateRequired, etc...",
			f:    func(path string) bool { return strings.HasPrefix(path, "f.") },
			expected: Object([]Property{
				{Name: "f", Type: Object([]Property{
					{Name: "f1", Type: Text(), Prefilled: "Prefilled of f1", CreateRequired: true, UpdateRequired: true, Nullable: true},
				})},
			}),
		},
		{
			name: "Removing all the properties of an object",
			f:    func(path string) bool { return path != "b.x" && path != "b.with_description" },
			expected: Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "c", Type: Array(Text())},
				{Name: "d", Type: Object([]Property{
					{Name: "b", Type: Object([]Property{
						{Name: "x", Type: Text(), Description: "Description of 'x'"},
						{Name: "with_description", Type: Text()},
					}), Description: "Description of 'b'"},
				}), Description: "Description of 'd'"},
				{Name: "e", Type: Array(Map(Object([]Property{
					{Name: "a", Type: Text()},
					{Name: "b", Type: Text()},
				})))},
				{Name: "f", Type: Object([]Property{
					{Name: "f1", Type: Text(), Prefilled: "Prefilled of f1", CreateRequired: true, UpdateRequired: true, Nullable: true},
				})},
			}),
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := Prune(testObject, test.f)
			if err := sameType(got, test.expected); err != nil {
				t.Fatalf("\nexpected: %#v\ngot:      %#v", test.expected, got)
			}
		})
	}
}

func Test_Filter(t *testing.T) {

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

	t.Run("Valid object expected (1)", func(t *testing.T) {
		expected := Object([]Property{
			{Name: "a", Type: Text()},
			{Name: "c", Type: Array(Text())},
		})
		got := Filter(o, func(p Property) bool {
			return p.Name == "a" || p.Name == "c"
		})
		if err := sameType(expected, got); err != nil {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run("Valid object expected (2)", func(t *testing.T) {
		expected := Object([]Property{
			{Name: "a", Type: Text()},
			{Name: "b", Type: Object([]Property{
				{Name: "x", Type: Text()},
			})},
			{Name: "c", Type: Array(Text())},
		})
		got := Filter(o, func(p Property) bool {
			return p.Name != "d"
		})
		if err := sameType(expected, got); err != nil {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run("Invalid type expected", func(t *testing.T) {
		got := Filter(o, func(p Property) bool {
			return false
		})
		if got.Valid() {
			t.Fatalf("expected invalid type, got %v", got)
		}
	})

	t.Run("Original object expected", func(t *testing.T) {
		got := Filter(o, func(p Property) bool {
			return true
		})
		if err := sameType(o, got); err != nil {
			t.Fatalf("expected %v, got %v", o, got)
		}
	})

}
