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
		linkedConnectionA = "2Qn5zBpR9YH7"
		linkedConnectionB = "5zBpR9Y2QnM3"
		linkedConnectionC = "8QaT3mN7KxP5"
		linkedConnectionD = "B7mN9qK2xAC3"
		linkedConnectionE = "G3mN7Kx8QaD4"
	)

	tests := []struct {
		id      string
		with    []string
		without []string
	}{
		{linkedConnectionA, []string{linkedConnectionA}, []string{}},
		{linkedConnectionA, []string{linkedConnectionA, linkedConnectionB}, []string{linkedConnectionB}},
		{linkedConnectionB, []string{linkedConnectionA, linkedConnectionB}, []string{linkedConnectionA}},
		{linkedConnectionC, []string{linkedConnectionB, linkedConnectionC, linkedConnectionD, linkedConnectionE}, []string{linkedConnectionB, linkedConnectionD, linkedConnectionE}},
		{linkedConnectionE, []string{linkedConnectionA, linkedConnectionC, linkedConnectionD, linkedConnectionE}, []string{linkedConnectionA, linkedConnectionC, linkedConnectionD}},
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
