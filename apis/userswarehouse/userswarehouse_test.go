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

func Test_concatSlices(t *testing.T) {
	tests := []struct {
		slice    any
		elems    any
		expected any
	}{
		{[]int{}, []int{}, []int{}},
		{[]int{}, []int{4, 5, 6}, []int{4, 5, 6}},
		{[]int{1, 2, 3}, []int{}, []int{1, 2, 3}},
		{[]int{1, 2, 3}, []int{4, 5, 6}, []int{1, 2, 3, 4, 5, 6}},
		{[]string{"1", "2", "3"}, []string{"4", "5", "6"}, []string{"1", "2", "3", "4", "5", "6"}},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := concatSlices(test.slice, test.elems)
			if !reflect.DeepEqual(test.expected, got) {
				t.Fatalf("expected %#v, got %#v", test.expected, got)
			}
		})
	}
}

func Test_deduplicate(t *testing.T) {
	tests := []struct {
		s        any
		expected any
	}{
		{[]int{}, []int{}},
		{[]int{1}, []int{1}},
		{[]int{10, 10}, []int{10}},
		{[]int{10, 3, 10}, []int{10, 3}},
		{[]int{1, 2, 3}, []int{1, 2, 3}},
		{[]int{10, 3, 10, 50, 50}, []int{10, 3, 50}},
		{[]string{"a", "x", "x"}, []string{"a", "x"}},
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
