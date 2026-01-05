// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"slices"
	"testing"
)

func TestAddAndRemoveLinkedConnection(t *testing.T) {

	tests := []struct {
		id      int
		with    []int
		without []int
	}{
		{1, []int{1}, []int{}},
		{1, []int{1, 2}, []int{2}},
		{2, []int{1, 2}, []int{1}},
		{8, []int{2, 5, 8, 15, 16}, []int{2, 5, 15, 16}},
		{16, []int{1, 8, 15, 16}, []int{1, 8, 15}},
	}

	// Test the addLinkedConnection function.
	for _, test := range tests {
		without := slices.Clone(test.without)
		got := addLinkedConnection(test.without, test.id)
		if got == nil {
			t.Fatalf("expected %#v, got nil", test.with)
		}
		if !slices.Equal(test.with, got) {
			t.Fatalf("expected %#v, got %#v", test.with, got)
		}
		if !slices.Equal(without, test.without) {
			t.Fatalf("the 'without' slice has been changed")
		}
	}

	// Test the removeLinkedConnection function.
	for _, test := range tests {
		with := slices.Clone(test.with)
		got := removeLinkedConnection(test.with, test.id)
		if got == nil {
			t.Fatal("unexpected nil")
		}
		if !slices.Equal(test.without, got) {
			t.Fatalf("expected %#v, got %#v", test.without, got)
		}
		if !slices.Equal(with, test.with) {
			t.Fatalf("the 'with' slice has been changed")
		}
	}

}
