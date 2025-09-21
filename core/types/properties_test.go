package types

import (
	"errors"
	"strings"
	"testing"
)

func Test_Properties_All(t *testing.T) {
	properties := []Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Boolean()},
	}
	pn := Object(properties).Properties()
	i := 0
	for k, p := range pn.All() {
		if k != i {
			t.Fatalf("expected i=%d, got i=%d", i, k)
		}
		if err := sameProperty(p, properties[i]); err != nil {
			t.Fatal(err)
		}
		i++
	}
}

// Test_Properties_ByName tests Properties.ByName.
func Test_Properties_ByName(t *testing.T) {
	schema := Object([]Property{
		{Name: "k", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "a", Type: Boolean()},
	})
	tests := []struct {
		name     string
		expected Property
		ok       bool
	}{
		{"k", Property{Name: "k", Type: Text()}, true},
		{"b", Property{Name: "b", Type: Object([]Property{{Name: "x", Type: Text()}})}, true},
		{"a", Property{Name: "a", Type: Boolean()}, true},
		{"x", Property{}, false},
		{"", Property{}, false},
		{"a.", Property{}, false},
		{"6", Property{}, false},
	}
	properties := schema.Properties()
	for _, test := range tests {
		got, ok := properties.ByName(test.name)
		if ok {
			if !test.ok {
				t.Fatal("expected not found, got found")
			}
			return
		}
		if err := sameProperty(test.expected, got); err != nil {
			t.Fatal(err)
		}
	}
}

// Test_Properties_ByPath tests Properties.ByPath and Properties.ByPathSlice.
func Test_Properties_ByPath(t *testing.T) {
	cases := []struct {
		name     string
		t        Type
		path     string
		expected Property
		err      error
	}{
		{
			name: "path with single component - property (of type text) exists",
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
			name: "path with single component - property (of type object) exists",
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
			name: "path with two components - property (of type text) exists",
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
			path: "billing_address.city",
			expected: Property{
				Name: "billing_address", Type: Object([]Property{
					{Name: "street", Type: Text()},
				})},
			err: errors.New("property path \"billing_address.city\" does not exist"),
		},
		{
			name: "path with three components - property (text within an object within an object) exists",
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
			name: "path with four components - second component of path is not an object",
			t: Object([]Property{
				{Name: "movie", Type: Object([]Property{
					{Name: "writer", Type: Text()},
				})},
			}),
			path:     "movie.writer.address.last_name",
			expected: Property{Name: "writer", Type: Text()},
			err:      errors.New("property path \"movie.writer.address\" does not exist"),
		},
		{
			name: "path with three components - second component of path is not an object",
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
			expected: Property{Name: "writer", Type: Text()},
			err:      errors.New("property path \"movie.writer.last_name\" does not exist"),
		},
	}
	for _, cas := range cases {
		properties := cas.t.Properties()
		t.Run("Properties.ByPath: "+cas.name, func(t *testing.T) {
			// Test PropertyByPath.
			got, err := properties.ByPath(cas.path)
			if err != nil {
				if cas.err == nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if err := sameProperty(cas.expected, got); err != nil {
					t.Fatal(err)
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
		t.Run("Properties.ByPathSlice: "+cas.name, func(t *testing.T) {
			// Test PropertyByPathSlice.
			got, err := properties.ByPathSlice(strings.Split(cas.path, "."))
			if err != nil {
				if cas.err == nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if err := sameProperty(cas.expected, got); err != nil {
					t.Fatal(err)
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
	// Test ByPathSlice with nil and empty path values.
	properties := Object([]Property{{Name: "first_name", Type: Text()}}).Properties()
	for _, path := range [][]string{nil, {}} {
		func() {
			defer func() {
				err := recover()
				if err == nil {
					t.Fatalf("Properties.ByPathSlice(%#v): expected panic, got no panic", path)
				}
				if err != "path is empty" {
					t.Fatalf("Properties.ByPathSlice(%#v): expected panic message \"path is empty\", got panic message %q", path, err)
				}
			}()
			_, _ = properties.ByPathSlice(path)
		}()
	}
}

func Test_Properties_ContainsName(t *testing.T) {
	properties := []Property{
		{Name: "k", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "a", Type: Boolean()},
	}
	tests := []struct {
		name     string
		expected bool
	}{
		{"k", true},
		{"b", true},
		{"a", true},
		{"x", false},
		{"", false},
		{".", false},
		{"a.", false},
		{"2", false},
	}
	pn := Object(properties).Properties()
	for _, test := range tests {
		got := pn.ContainsName(test.name)
		if test.expected != got {
			t.Fatalf("expected %t, got %t", test.expected, got)
		}
	}
}

func Test_Properties_ContainsPath(t *testing.T) {
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
	properties := o.Properties()
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			got := properties.ContainsPath(test.path)
			if test.exists != got {
				t.Fatalf("expected %t, got %t", test.exists, got)
			}
		})
	}
	// Test invalid paths.
	for _, path := range []string{".", " ", "a.b.7", "a.", "a. ", "d.y.?.a"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("ContainsPath(%#v): expected panic, got no panic", path)
				}
			}()
			properties.ContainsPath(path)
		}()
	}
}

func Test_Properties_Len(t *testing.T) {
	properties := []Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Text()},
		{Name: "c", Type: Text()},
	}
	if got := Object(properties).Properties().Len(); len(properties) != got {
		t.Errorf("expected %d, got %d", len(properties), got)
	}
}

func Test_Properties_Slice(t *testing.T) {
	properties := []Property{
		{Name: "a", Type: Text()},
		{Name: "b", Type: Object([]Property{
			{Name: "x", Type: Text()},
		})},
		{Name: "c", Type: Boolean()},
	}
	pn := Object(properties).Properties()
	i := 0
	for k, p := range pn.Slice() {
		if k != i {
			t.Fatalf("expected i=%d, got i=%d", i, k)
		}
		if err := sameProperty(p, properties[i]); err != nil {
			t.Fatal(err)
		}
		i++
	}
}

func Test_WalkAll(t *testing.T) {
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
	walk := Object(properties).Properties().WalkAll()
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
	if i != len(iterations) {
		t.Fatalf("expected a total of %d iterations, got %d", len(iterations), i)
	}
}

func Test_WalkObjects(t *testing.T) {
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
		{Name: "e", Type: Map(Object([]Property{
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
		{"e", Property{Name: "e", Type: Map(Object([]Property{
			{Name: "x", Type: Map(Boolean())},
			{Name: "y", Type: Map(Object([]Property{
				{Name: "a", Type: Text()},
				{Name: "b", Type: Int(32)},
			}))},
			{Name: "z", Type: Text()},
		}))}},
	}
	walk := Object(properties).Properties().WalkObjects()
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
	if i != len(iterations) {
		t.Fatalf("expected a total of %d iterations, got %d", len(iterations), i)
	}
}
