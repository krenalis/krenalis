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

	"chichi/connector/types"
)

func Test_parsePropertyExpression(t *testing.T) {
	cases := []struct {
		p             string
		expectedSlice types.Path
		expectedOk    bool
	}{
		// Valid property expressions.
		{"street1", types.Path{"street1"}, true},
		{"address_street1", types.Path{"address_street1"}, true},
		{"address.street1", types.Path{"address", "street1"}, true},

		// Invalid property expressions.
		{"", nil, false},
		{".", nil, false},
		{"222.", nil, false},
		{".222", nil, false},
		{"x.", nil, false},
		{".x", nil, false},
		{"32", nil, false},
		{"traits.32", nil, false},
		{"traits..", nil, false},
	}
	for _, cas := range cases {
		t.Run(cas.p, func(t *testing.T) {
			gotSlice, gotOk := parsePropertyExpression(cas.p)
			if !reflect.DeepEqual(cas.expectedSlice, gotSlice) {
				t.Fatalf("expected %#v, got %#v", cas.expectedSlice, gotSlice)
			}
			if cas.expectedOk != gotOk {
				t.Fatalf("expected %t, got %t", cas.expectedOk, gotOk)
			}
		})
	}
}
