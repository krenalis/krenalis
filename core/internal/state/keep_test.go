// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"slices"
	"testing"
)

func TestAddAndRemoveLinkedConnection(t *testing.T) {
	const (
		connA = "2Qn5zBpR9YH7"
		connB = "5zBpR9Y2QnM3"
		connC = "8QaT3mN7KxP5"
		connD = "B7mN9qK2xAC3"
		connE = "G3mN7Kx8QaD4"
	)

	tests := []struct {
		id      string
		with    []string
		without []string
	}{
		{connA, []string{connA}, []string{}},
		{connA, []string{connA, connB}, []string{connB}},
		{connB, []string{connA, connB}, []string{connA}},
		{connC, []string{connB, connC, connD, connE}, []string{connB, connD, connE}},
		{connE, []string{connA, connC, connD, connE}, []string{connA, connC, connD}},
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
