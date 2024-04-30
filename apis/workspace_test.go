//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

import (
	"testing"

	"github.com/open2b/chichi/types"
)

func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc", true},
		{"123", true},
		{"aBc123", true},
		{"abc123", true},
		{"firstName", true},
		{"firstname", true},
		{"123abc456", true},

		{"", false},
		{"123!", false},
		{"abc!", false},
		{"ab c123", false},
		{"email-", false},
		{"_email", false},
		{"email\n", false},
		{"123🚀abc", false},
		{"\tabc123", false},
		{"  lastname  ", false},
		{"!@#$%^&*()", false},
		{"first_name", false},
	}

	for _, test := range tests {
		got := isAlphaNumeric(test.input)
		if got != test.expected {
			t.Errorf("s: %q, expected: %t, got: %t", test.input, test.expected, got)
		}
	}
}

func Test_onlyAlphaNumericPropertyNames(t *testing.T) {
	tests := []struct {
		s        types.Type
		expected bool
	}{
		{
			s: types.Object([]types.Property{
				{Name: "firstName", Type: types.Text()},
			}),
			expected: true,
		},
		{
			s: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			expected: false,
		},
		{
			s: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "firstName", Type: types.Text()},
				})},
			}),
			expected: true,
		},
		{
			s: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "first_name", Type: types.Text()},
				})},
			}),
			expected: false,
		},
	}
	for _, test := range tests {
		got := onlyAlphaNumericPropertyNames(test.s)
		if got != test.expected {
			t.Errorf("expected %t, got %t", test.expected, got)
		}
	}
}
