//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"reflect"
	"testing"

	"github.com/open2b/chichi/types"
)

func Test_unusedProperties(t *testing.T) {
	cases := []struct {
		schema   types.Type
		paths    []types.Path
		expected []string
	}{
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			paths: []types.Path{
				{"first_name"},
			},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "last_name", Type: types.Text()},
			}),
			paths: []types.Path{
				{"first_name"},
				{"last_name"},
			},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			expected: []string{"first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
			}),
			expected: []string{"email", "first_name"},
		},
	}
	for _, cas := range cases {
		got := unusedProperties(cas.schema, cas.paths)
		if !reflect.DeepEqual(cas.expected, got) {
			t.Fatalf("expecting %#v, got %#v", cas.expected, got)
		}
	}
}

func Test_validateLastChangeTimeFormat(t *testing.T) {
	tests := []struct {
		format string
		err    string
	}{
		// Valid.
		{format: "DateTime"},
		{format: "DateOnly"},
		{format: "'%Y'"},
		{format: "Excel"},
		{format: "ISO8601"},
		{format: "'%'"},

		// Invalid.
		{format: "'", err: `last change time format "'" is not a valid format`},
		{format: "'''", err: `last change time format "'''" is not a valid format`},
		{format: "", err: "last change time format cannot be empty"},
		{format: "Date", err: `last change time format "Date" is not a valid format`},
		{format: "excel", err: `last change time format "excel" is not a valid format`},
		{format: "iso8601", err: `last change time format "iso8601" is not a valid format`},
		{format: "%Y", err: `last change time strptime format must be enclosed between "'" characters`},
		{format: "'%Y", err: `last change time strptime format must be enclosed between "'" characters`},
		{format: "%Y'", err: `last change time strptime format must be enclosed between "'" characters`},
		{format: "\xc3\x28", err: "last change time format must be UTF-8 valid"},
		{format: "''", err: `last change time format "''" is not a valid format`},
		{format: "'YYYY-MM-DD'", err: `last change time format "'YYYY-MM-DD'" is not a valid format`},
		{format: "'%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y'", err: "last change time format is longer than 64 runes"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			got := validateLastChangeTimeFormat(test.format)
			var gotStr string
			if got != nil {
				gotStr = got.Error()
			}
			if test.err != gotStr {
				t.Fatalf("expecting %q, got %q", test.err, gotStr)
			}
		})
	}
}
