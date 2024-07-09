//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types

import (
	"testing"
)

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
