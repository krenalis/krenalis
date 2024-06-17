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
		paths    []string
		expected []string
	}{
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			paths: []string{"first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "last_name", Type: types.Text()},
			}),
			paths: []string{"first_name", "last_name"},
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
		{format: "ISO8601"},
		{format: "Excel"},
		{format: "%Y-%m-%d %H:%M:%S"},
		{format: "%Y-%m-%d"},
		{format: "%Y"},
		{format: "%"},

		// Invalid.
		{format: "", err: "last change time format is empty"},
		{format: "iso8601", err: `last change time format "iso8601" is not valid`},
		{format: "excel", err: `last change time format "excel" is not valid`},
		{format: "Y-m-d", err: `last change time format "Y-m-d" is not valid`},
		{format: "%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y", err: "last change time format is longer than 64 runes"},
		{format: "%Y-%m-%d\x00%H:%M:%S", err: "last change time format contains the NUL rune"},
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
