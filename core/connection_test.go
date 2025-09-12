//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"testing"

	"github.com/meergo/meergo/core/types"
)

func Test_isMetaProperty(t *testing.T) {
	tests := []struct {
		p        types.Property
		expected bool
	}{
		{types.Property{}, false}, // invalid property, shouldn't happen.
		{types.Property{Name: "a", Type: types.Int(32)}, false},
		{types.Property{Name: "hello", Type: types.Int(32)}, false},
		{types.Property{Name: "_hello_", Type: types.Int(32)}, false},
		{types.Property{Name: "__hello", Type: types.Int(32)}, false},
		{types.Property{Name: "__", Type: types.Int(32)}, false},
		{types.Property{Name: "____", Type: types.Int(32)}, false},
		{types.Property{Name: "__hello__", Type: types.Int(32)}, true},
		{types.Property{Name: "__h__", Type: types.Int(32)}, true},
		{types.Property{Name: "__hey_test__", Type: types.Int(32)}, true},
	}
	for _, test := range tests {
		got := isMetaProperty(test.p.Name)
		if test.expected != got {
			t.Errorf("%#v: expected %t, got %t", test.p, test.expected, got)
		}
	}
}
