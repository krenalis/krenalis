//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package userswarehouse

import (
	"reflect"
	"testing"
)

func Test_deduplicate(t *testing.T) {
	tests := []struct {
		s        []any
		expected []any
	}{
		{nil, nil},
		{[]any{}, []any{}},
		{[]any{1}, []any{1}},
		{[]any{10, 10}, []any{10}},
		{[]any{10, 3, 10}, []any{10, 3}},
		{[]any{1, 2, 3}, []any{1, 2, 3}},
		{[]any{"a", "x", "x"}, []any{"a", "x"}},
		{[]any{10, 3, 10, 50, 50}, []any{10, 3, 50}},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := deduplicate(test.s)
			if !reflect.DeepEqual(test.expected, got) {
				t.Fatalf("expected %#v, got %#v", test.expected, got)
			}
		})
	}
}
